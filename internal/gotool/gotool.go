// Package gotool wraps the go command: module discovery and the package graph
// of every test binary in the main module.
package gotool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Package is the subset of `go list -json` output hyrts uses.
type Package struct {
	ImportPath string
	Dir        string
	Name       string
	ForTest    string
	Standard   bool
	Module     *struct {
		Path string
		Main bool
	}

	GoFiles, CgoFiles, CFiles, CXXFiles, MFiles, HFiles, FFiles, SFiles   []string
	SwigFiles, SwigCXXFiles, SysoFiles, IgnoredGoFiles, IgnoredOtherFiles []string
	TestGoFiles, XTestGoFiles                                             []string
	EmbedFiles, TestEmbedFiles, XTestEmbedFiles                           []string

	Deps  []string
	Error *struct{ Err string }
}

// HasTests reports whether the package has a test binary.
func (p *Package) HasTests() bool { return len(p.TestGoFiles)+len(p.XTestGoFiles) > 0 }

// NonGoSources are files compiled or embedded into the package that are not
// Go source: a change to any of them can change any code in the package.
func (p *Package) NonGoSources() []string {
	var out []string
	for _, l := range [][]string{p.CFiles, p.CXXFiles, p.MFiles, p.HFiles, p.FFiles, p.SFiles,
		p.SwigFiles, p.SwigCXXFiles, p.SysoFiles, p.EmbedFiles, p.TestEmbedFiles, p.XTestEmbedFiles} {
		out = append(out, l...)
	}
	return out
}

// Graph is the main module's packages and what each test binary links.
type Graph struct {
	Module string
	Root   string // module root, symlinks resolved
	// Local holds the main module's packages keyed by import path.
	Local map[string]*Package
	// TestDeps maps a package with tests to the main-module packages linked
	// into its test binary, itself included.
	TestDeps map[string]map[string]bool
	// DirPkg maps a module-relative directory to its import path.
	DirPkg map[string]string
}

// ModuleRoot returns the directory holding go.mod for dir, symlinks resolved.
func ModuleRoot(dir string) (string, error) {
	out, err := Output(dir, "env", "GOMOD")
	if err != nil {
		return "", err
	}
	gomod := strings.TrimSpace(out)
	if gomod == "" || gomod == os.DevNull {
		return "", fmt.Errorf("%s is not inside a Go module", dir)
	}
	return filepath.EvalSymlinks(filepath.Dir(gomod))
}

// Version returns `go env GOVERSION`.
func Version(dir string) string {
	out, _ := Output(dir, "env", "GOVERSION")
	return strings.TrimSpace(out)
}

// Output runs the go command in dir and returns its stdout.
func Output(dir string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return string(out), nil
}

// TagsArgs returns the -tags flag for the go command, or nothing.
func TagsArgs(tags string) []string {
	if tags == "" {
		return nil
	}
	return []string{"-tags", tags}
}

// Load lists every package of the main module under root and the
// main-module dependencies of each test binary.
func Load(root, tags string) (*Graph, error) {
	args := append([]string{"list", "-e", "-test", "-deps", "-json"}, TagsArgs(tags)...)
	args = append(args, "./...")
	out, err := Output(root, args...)
	if err != nil {
		return nil, err
	}
	g := &Graph{Root: root, Local: map[string]*Package{}, TestDeps: map[string]map[string]bool{}, DirPkg: map[string]string{}}
	var bins []*Package
	dec := json.NewDecoder(strings.NewReader(out))
	for {
		var p Package
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decoding go list output: %v", err)
		}
		if p.Standard || p.Module == nil || !p.Module.Main {
			continue
		}
		g.Module = p.Module.Path
		switch {
		case strings.HasSuffix(p.ImportPath, ".test") && !strings.Contains(p.ImportPath, " "):
			pp := p
			bins = append(bins, &pp)
		case p.ForTest == "" && !strings.Contains(p.ImportPath, " "):
			pp := p
			g.Local[p.ImportPath] = &pp
			if rel, err := filepath.Rel(root, realDir(p.Dir)); err == nil {
				g.DirPkg[filepath.ToSlash(rel)] = p.ImportPath
			}
		}
	}
	for _, b := range bins {
		pkg := strings.TrimSuffix(b.ImportPath, ".test")
		deps := map[string]bool{pkg: true}
		for _, d := range b.Deps {
			d, _, _ = strings.Cut(d, " ")
			if _, ok := g.Local[d]; ok {
				deps[d] = true
			}
		}
		g.TestDeps[pkg] = deps
	}
	return g, nil
}

// RelDir returns the module-relative directory of a local package.
func (g *Graph) RelDir(pkg string) string {
	p := g.Local[pkg]
	if p == nil {
		return ""
	}
	rel, _ := filepath.Rel(g.Root, realDir(p.Dir))
	return filepath.ToSlash(rel)
}

// Dependents returns the packages whose test binary links pkg.
func (g *Graph) Dependents(pkg string) []string {
	var out []string
	for t, deps := range g.TestDeps {
		if deps[pkg] {
			out = append(out, t)
		}
	}
	return out
}

func realDir(d string) string {
	if r, err := filepath.EvalSymlinks(d); err == nil {
		return r
	}
	return d
}
