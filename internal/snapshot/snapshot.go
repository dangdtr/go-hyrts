// Package snapshot records the content of every file in a module so that a
// later version can be compared against it without needing git history.
package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/dangdtr/go-hyrts/internal/astinfo"
)

// File is the recorded state of one file.
type File struct {
	Hash string `json:"h"`
	// Go files only.
	Package string                  `json:"p,omitempty"` // import path of the directory's package
	Name    string                  `json:"n,omitempty"` // package clause (p or p_test)
	Funcs   map[string]string       `json:"f,omitempty"` // func key -> checksum
	Decls   map[string]astinfo.Decl `json:"d,omitempty"`
	Tests   []string                `json:"t,omitempty"` // test entry points declared in the file

	AST *astinfo.File `json:"-"` // parsed form, current snapshot only
}

// Snapshot maps module-relative slash paths to file state.
type Snapshot map[string]*File

// Skip reports paths hyrts never tracks.
func Skip(rel string) bool {
	return rel == ".hyrts" || strings.HasPrefix(rel, ".hyrts/") || rel == ".git" || strings.HasPrefix(rel, ".git/")
}

// Take records every file under root: tracked and untracked-but-not-ignored
// files when root is in a git work tree, every file otherwise. Files are read,
// hashed and parsed by GOMAXPROCS goroutines. Package import paths are filled
// in later by AssignPackages, so this can run alongside `go list`.
func Take(root string) (Snapshot, error) {
	files, err := list(root)
	if err != nil {
		return nil, err
	}
	recs := make([]*File, len(files))
	next := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < runtime.GOMAXPROCS(0); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				recs[i] = record(root, files[i])
			}
		}()
	}
	for i, rel := range files {
		if !Skip(rel) {
			next <- i
		}
	}
	close(next)
	wg.Wait()
	snap := make(Snapshot, len(files))
	for i, f := range recs {
		if f != nil {
			snap[files[i]] = f
		}
	}
	return snap, nil
}

// record hashes one file and, for Go files, summarises its declarations.
func record(root, rel string) *File {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return nil // deleted but still in the index, or unreadable
	}
	sum := sha256.Sum256(data)
	f := &File{Hash: hex.EncodeToString(sum[:12])}
	if strings.HasSuffix(rel, ".go") {
		if a, err := astinfo.Parse(rel, data); err == nil {
			f.AST = a
			f.Name = a.Package
			f.Funcs = map[string]string{}
			for k, fn := range a.Funcs {
				f.Funcs[k] = fn.Hash
				if fn.Test {
					f.Tests = append(f.Tests, k)
				}
			}
			sort.Strings(f.Tests)
			f.Decls = a.Decls
		}
	}
	return f
}

// AssignPackages records, for every Go file, the import path of the package
// in its directory (dirPkg maps a relative directory to an import path).
func (s Snapshot) AssignPackages(dirPkg map[string]string) {
	for rel, f := range s {
		if f.Funcs != nil {
			f.Package = dirPkg[pathDir(rel)]
		}
	}
}

func pathDir(rel string) string {
	d := filepath.ToSlash(filepath.Dir(rel))
	return d
}

// FuncID identifies a function independently of the file it is in, so that
// moving code between files of a package does not change identities:
// "importpath#clause|Name" ("clause" is p or p_test). init functions, which
// are anonymous and per file, keep the file name: "importpath#clause|x.go:init#0".
func FuncID(pkg, clause, rel, key string) string {
	if strings.HasPrefix(key, "init#") {
		key = filepath.Base(rel) + ":" + key
	}
	return pkg + "#" + clause + "|" + key
}

// Dir returns the slash directory of a relative path ("." at the root).
func Dir(rel string) string { return pathDir(rel) }

func list(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	cmd.Dir = root
	if out, err := cmd.Output(); err == nil {
		var files []string
		seen := map[string]bool{}
		for _, b := range bytes.Split(out, []byte{0}) {
			if s := string(b); s != "" && !seen[s] {
				seen[s] = true
				files = append(files, s)
			}
		}
		return files, nil
	}
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && (Skip(rel) || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}
