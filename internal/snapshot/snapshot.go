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
	"strings"

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
// files when root is in a git work tree, every file otherwise.
// dirPkg maps a relative directory to its package import path.
func Take(root string, dirPkg map[string]string) (Snapshot, error) {
	files, err := list(root)
	if err != nil {
		return nil, err
	}
	snap := Snapshot{}
	for _, rel := range files {
		if Skip(rel) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue // deleted but still in the index, or unreadable
		}
		sum := sha256.Sum256(data)
		f := &File{Hash: hex.EncodeToString(sum[:12])}
		if strings.HasSuffix(rel, ".go") {
			if a, err := astinfo.Parse(rel, data); err == nil {
				f.AST = a
				f.Package = dirPkg[pathDir(rel)]
				f.Name = a.Package
				f.Funcs = map[string]string{}
				for k, fn := range a.Funcs {
					f.Funcs[k] = fn.Hash
					if fn.Test {
						f.Tests = append(f.Tests, k)
					}
				}
				f.Decls = a.Decls
			}
		}
		snap[rel] = f
	}
	return snap, nil
}

func pathDir(rel string) string {
	d := filepath.ToSlash(filepath.Dir(rel))
	return d
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
