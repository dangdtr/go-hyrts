// Package selector decides which tests can be affected by the difference
// between the collected snapshot and the current module.
//
// Rules, from finest to coarsest:
//   - a changed or deleted function selects the tests that executed it;
//   - a changed test function selects itself; a new test is always selected;
//   - a changed test helper, TestMain or test-file declaration selects every
//     test of that package;
//   - a changed, deleted or side-effecting declaration (type, var, const,
//     blank import), a new init or a new method on an existing type selects
//     every test whose binary links the package (static fallback);
//   - a changed non-Go file selects the tests that opened or stat'ed it, or a
//     directory containing it, and, if it is embedded or compiled into a
//     package, every test whose binary links that package;
//   - go.mod, go.sum, go.work, vendor/, the Go version or the build tags
//     changing selects everything.
package selector

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/dangdtr/go-hyrts/internal/db"
	"github.com/dangdtr/go-hyrts/internal/gotool"
	"github.com/dangdtr/go-hyrts/internal/snapshot"
)

// Test is a test that exists in the current module.
type Test struct {
	Pkg, Name string
	Seconds   float64 // from collection, 0 if unknown
}

// Result is a selection with the reason for every selected test.
type Result struct {
	All       bool
	AllReason string
	Tests     map[string]Test     // every current test, by id
	Selected  map[string][]string // id -> reasons
	Changed   []string            // changed files, module-relative
	Unused    []string            // changed non-Go files no test depends on
}

// Options tunes a selection.
type Options struct {
	Always *regexp.Regexp // tests matching are always selected
	Tags   string
}

// Select compares old against the current module described by g and cur.
func Select(old *db.DB, g *gotool.Graph, cur snapshot.Snapshot, goVersion string, o Options) *Result {
	s := &sel{old: old, g: g, cur: cur, r: &Result{Tests: map[string]Test{}, Selected: map[string][]string{}}}
	s.universe()
	switch {
	case old.GoVersion != goVersion:
		s.all(fmt.Sprintf("Go version changed (%s -> %s)", old.GoVersion, goVersion))
	case old.Tags != o.Tags:
		s.all(fmt.Sprintf("build tags changed (%q -> %q)", old.Tags, o.Tags))
	}
	s.diffFiles()
	if s.r.All {
		return s.r
	}
	for id := range s.r.Tests {
		ot, ok := old.Tests[id]
		switch {
		case !ok:
			s.pick(id, "new test (no collected data)")
		case ot.Status == "fail" || ot.Status == "skip":
			s.pick(id, map[string]string{"fail": "failed", "skip": "skipped"}[ot.Status]+" during collection")
		}
		if o.Always != nil && o.Always.MatchString(id) {
			s.pick(id, "matches -always")
		}
	}
	s.funcIndex()
	changedPkgs := map[string]bool{}
	for _, rel := range s.r.Changed {
		if strings.HasSuffix(rel, ".go") {
			for _, f := range []*snapshot.File{old.Files[rel], cur[rel]} {
				if f != nil && f.Package != "" {
					changedPkgs[f.Package] = true
				}
			}
		}
	}
	pkgs := make([]string, 0, len(changedPkgs))
	for p := range changedPkgs {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	for _, p := range pkgs {
		s.goPackage(p)
	}
	for _, rel := range s.r.Changed {
		s.fileDeps(rel)
	}
	return s.r
}

type sel struct {
	old    *db.DB
	g      *gotool.Graph
	cur    snapshot.Snapshot
	r      *Result
	byFunc map[string][]string // func id -> test ids
	byPkg  map[string][]string // pkg -> current test ids
}

func (s *sel) all(reason string) {
	if !s.r.All {
		s.r.All, s.r.AllReason = true, reason
	}
}

func (s *sel) pick(id, reason string) {
	if _, ok := s.r.Tests[id]; !ok {
		return // no longer exists
	}
	if len(s.r.Selected[id]) < 3 {
		s.r.Selected[id] = append(s.r.Selected[id], reason)
	}
}

// universe lists the tests in the current module from its test files.
func (s *sel) universe() {
	s.byPkg = map[string][]string{}
	for pkg, p := range s.g.Local {
		if !p.HasTests() {
			continue
		}
		dir := s.g.RelDir(pkg)
		for _, name := range append(append([]string{}, p.TestGoFiles...), p.XTestGoFiles...) {
			f := s.cur[path.Join(dir, name)]
			if f == nil {
				continue
			}
			for _, t := range f.Tests {
				id := db.ID(pkg, t)
				var sec float64
				if ot := s.old.Tests[id]; ot != nil {
					sec = ot.Seconds
				}
				s.r.Tests[id] = Test{Pkg: pkg, Name: t, Seconds: sec}
				s.byPkg[pkg] = append(s.byPkg[pkg], id)
			}
		}
	}
}

func (s *sel) funcIndex() {
	s.byFunc = map[string][]string{}
	for id, t := range s.old.Tests {
		for _, fi := range t.Funcs {
			if fi < len(s.old.Funcs) {
				f := s.old.Funcs[fi]
				s.byFunc[f] = append(s.byFunc[f], id)
			}
		}
	}
}

func (s *sel) diffFiles() {
	seen := map[string]bool{}
	for rel, f := range s.old.Files {
		seen[rel] = true
		if c := s.cur[rel]; c == nil || c.Hash != f.Hash {
			s.r.Changed = append(s.r.Changed, rel)
		}
	}
	for rel := range s.cur {
		if !seen[rel] {
			s.r.Changed = append(s.r.Changed, rel)
		}
	}
	sort.Strings(s.r.Changed)
	for _, rel := range s.r.Changed {
		switch {
		case rel == "go.mod" || rel == "go.sum" || rel == "go.work" || rel == "go.work.sum":
			s.all(rel + " changed")
		case strings.HasPrefix(rel, "vendor/"):
			s.all("vendored dependency changed: " + rel)
		}
	}
}

// pkgTests selects every test of pkg.
func (s *sel) pkgTests(pkg, reason string) {
	for _, id := range s.byPkg[pkg] {
		s.pick(id, reason)
	}
}

// linkers selects every test whose binary links pkg.
func (s *sel) linkers(pkg, reason string) {
	for _, t := range s.g.Dependents(pkg) {
		s.pkgTests(t, reason)
	}
}

type entry struct {
	hash, rel, key string
	test           bool // declared in a _test.go file
	entry          bool // a test entry point
	method         string
	effect         bool
	types          []string
}

// decls collects package-level function and declaration checksums of pkg in
// one snapshot. Keys are namespaced by package clause (p vs p_test) and, for
// init functions, by file, so moving code between files is not a change.
func decls(snap snapshot.Snapshot, pkg string) (funcs, others map[string]entry) {
	funcs, others = map[string]entry{}, map[string]entry{}
	for rel, f := range snap {
		if f.Package != pkg || !strings.HasSuffix(rel, ".go") || f.Funcs == nil {
			continue
		}
		isTest := strings.HasSuffix(rel, "_test.go")
		ns := path.Base(rel)
		entries := map[string]bool{}
		for _, t := range f.Tests {
			entries[t] = true
		}
		for k, h := range f.Funcs {
			key := k
			if strings.HasPrefix(k, "init#") {
				key = ns + ":" + k
			}
			recv := ""
			if i := strings.Index(k, "."); i > 0 {
				recv = k[:i]
			}
			funcs[f.Name+"|"+key] = entry{hash: h, rel: rel, key: k, test: isTest, entry: entries[k], method: recv}
		}
		for k, d := range f.Decls {
			others[f.Name+"|"+k] = entry{hash: d.Hash, rel: rel, key: k, test: isTest, effect: d.Effect, types: d.Types}
		}
	}
	return funcs, others
}

func (s *sel) goPackage(pkg string) {
	oldF, oldD := decls(s.old.Files, pkg)
	newF, newD := decls(s.cur, pkg)

	newTypes := map[string]bool{}
	oldTypes := map[string]bool{}
	for _, d := range oldD {
		for _, t := range d.types {
			oldTypes[t] = true
		}
	}
	for _, d := range newD {
		for _, t := range d.types {
			if !oldTypes[t] {
				newTypes[t] = true
			}
		}
	}

	for k, o := range oldD {
		n, ok := newD[k]
		if ok && n.hash == o.hash {
			continue
		}
		what := "changed"
		if !ok {
			what = "removed"
		}
		s.declChange(pkg, o, fmt.Sprintf("%s %s in %s", o.key, what, o.rel))
	}
	for k, n := range newD {
		if _, ok := oldD[k]; ok || !n.effect {
			continue
		}
		s.declChange(pkg, n, fmt.Sprintf("%s added in %s", n.key, n.rel))
	}

	for k, o := range oldF {
		n, ok := newF[k]
		if ok && n.hash == o.hash {
			continue
		}
		what := "changed"
		if !ok {
			what = "removed"
		}
		reason := fmt.Sprintf("%s %s (%s)", o.key, what, o.rel)
		switch {
		case o.test && o.entry:
			if ok {
				s.pick(db.ID(pkg, o.key), "test changed")
			}
		case o.test:
			s.pkgTests(pkg, "test helper "+reason)
		default:
			for _, id := range s.byFunc[o.rel+"#"+o.key] {
				s.pick(id, "executes "+reason)
			}
		}
	}
	for k, n := range newF {
		if _, ok := oldF[k]; ok {
			continue
		}
		reason := fmt.Sprintf("%s added (%s)", n.key, n.rel)
		switch {
		case n.test && n.entry:
			// new test: already selected by universe check
		case strings.HasPrefix(n.key, "init#"):
			if n.test {
				s.pkgTests(pkg, reason)
			} else {
				s.linkers(pkg, reason)
			}
		case n.method != "" && !newTypes[n.method]:
			// A new method can make an existing type satisfy an interface
			// (String, Error, MarshalJSON, ...) and change behaviour.
			if n.test {
				s.pkgTests(pkg, "new method "+reason)
			} else {
				s.linkers(pkg, "new method "+reason)
			}
		}
	}
}

func (s *sel) declChange(pkg string, e entry, reason string) {
	if e.test {
		s.pkgTests(pkg, reason)
	} else {
		s.linkers(pkg, reason)
	}
}

// fileDeps applies the file-level rules: tests that read the file at run
// time (Go sources included: some tests scan source code), and, for non-Go
// files, packages that compile or embed it.
func (s *sel) fileDeps(rel string) {
	isGo := strings.HasSuffix(rel, ".go")
	used := isGo
	for pkg, p := range s.g.Local {
		if isGo {
			break
		}
		dir := s.g.RelDir(pkg)
		for _, f := range p.NonGoSources() {
			if path.Join(dir, f) == rel {
				s.linkers(pkg, rel+" changed (compiled or embedded into "+pkg+")")
				used = true
			}
		}
	}
	for id, t := range s.old.Tests {
		hit := ""
		for _, p := range t.Paths {
			if p == rel {
				hit = "reads " + rel
				break
			}
		}
		if hit == "" {
			for _, d := range t.Dirs {
				if d == "." || strings.HasPrefix(rel, d+"/") {
					hit = "reads directory " + d + " (" + rel + " changed)"
					break
				}
			}
		}
		if hit != "" {
			s.pick(id, hit)
			used = true
		}
	}
	if !used {
		s.r.Unused = append(s.r.Unused, rel)
	}
}

// Summary returns counts and estimated seconds of the selection.
func (r *Result) Summary() (selected, total int, selSec, totSec float64) {
	for id, t := range r.Tests {
		totSec += t.Seconds
		if r.All || r.Selected[id] != nil {
			selected++
			selSec += t.Seconds
		}
	}
	return selected, len(r.Tests), selSec, totSec
}

// ByPackage returns selected test names grouped by package, sorted.
func (r *Result) ByPackage() map[string][]string {
	out := map[string][]string{}
	for id, t := range r.Tests {
		if r.All || r.Selected[id] != nil {
			out[t.Pkg] = append(out[t.Pkg], t.Name)
		}
	}
	for _, v := range out {
		sort.Strings(v)
	}
	return out
}
