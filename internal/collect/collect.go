// Package collect runs every top-level test once, in its own process, and
// records the functions it executed (coverage) and the files and environment
// variables it touched (the testing package's testlog).
package collect

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dangdtr/go-hyrts/internal/db"
	"github.com/dangdtr/go-hyrts/internal/gotool"
	"github.com/dangdtr/go-hyrts/internal/snapshot"
)

// Options configures a collection run.
type Options struct {
	Root     string
	Tags     string
	Jobs     int           // packages processed in parallel
	Timeout  time.Duration // per test
	Packages []string      // limit to these import paths (empty = all)
	Log      io.Writer
}

// Run collects dependencies for every test in the module and returns the database.
func Run(ctx context.Context, o Options) (*db.DB, error) {
	if o.Jobs < 1 {
		o.Jobs = 1
	}
	if o.Log == nil {
		o.Log = io.Discard
	}
	g, err := gotool.Load(o.Root, o.Tags)
	if err != nil {
		return nil, err
	}
	snap, err := snapshot.Take(o.Root, g.DirPkg)
	if err != nil {
		return nil, err
	}
	d := &db.DB{
		Version:   db.Version,
		GoVersion: gotool.Version(o.Root),
		Tags:      o.Tags,
		Module:    g.Module,
		Commit:    gitHead(o.Root),
		Collected: time.Now().UTC(),
		Files:     snap,
		Tests:     map[string]*db.Test{},
	}
	c := &collector{o: o, g: g, snap: snap, d: d, funcIdx: map[string]int{}}
	c.buildSpans()

	tmp, err := os.MkdirTemp("", "hyrts-collect-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	var pkgs []string
	want := map[string]bool{}
	for _, p := range o.Packages {
		want[p] = true
	}
	for p := range g.TestDeps {
		if len(want) == 0 || want[p] {
			pkgs = append(pkgs, p)
		}
	}
	sort.Strings(pkgs)

	work := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < o.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range work {
				c.pkg(ctx, tmp, p)
			}
		}()
	}
	for _, p := range pkgs {
		work <- p
	}
	close(work)
	wg.Wait()
	d.Funcs = c.funcs
	return d, ctx.Err()
}

type span struct {
	start, end int
	id         string
}

type collector struct {
	o    Options
	g    *gotool.Graph
	snap snapshot.Snapshot
	d    *db.DB

	spans map[string][]span // import-path file name ("mod/pkg/x.go") -> funcs

	mu      sync.Mutex
	funcs   []string
	funcIdx map[string]int
}

func (c *collector) buildSpans() {
	c.spans = map[string][]span{}
	for rel, f := range c.snap {
		if f.AST == nil || f.Package == "" || strings.HasSuffix(rel, "_test.go") {
			continue
		}
		key := f.Package + "/" + filepath.Base(rel)
		for _, k := range f.AST.SortedFuncs() {
			fn := f.AST.Funcs[k]
			c.spans[key] = append(c.spans[key], span{fn.Start, fn.End, rel + "#" + k})
		}
	}
}

func (c *collector) logf(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fmt.Fprintf(c.o.Log, format, args...)
}

func (c *collector) funcID(id string) int {
	if i, ok := c.funcIdx[id]; ok {
		return i
	}
	c.funcIdx[id] = len(c.funcs)
	c.funcs = append(c.funcs, id)
	return c.funcIdx[id]
}

func (c *collector) add(t *db.Test) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.d.Tests[db.ID(t.Pkg, t.Name)] = t
}

func (c *collector) pkg(ctx context.Context, tmp, pkg string) {
	dir := c.g.Local[pkg].Dir
	bin := filepath.Join(tmp, strings.NewReplacer("/", "_", ".", "_").Replace(pkg)+".test")
	args := []string{"test", "-c", "-o", bin, "-cover", "-covermode=set", "-coverpkg=" + c.g.Module + "/..."}
	args = append(args, gotool.TagsArgs(c.o.Tags)...)
	args = append(args, pkg)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = c.o.Root
	if out, err := cmd.CombinedOutput(); err != nil {
		// The package does not build: its tests will always be selected.
		c.logf("hyrts: %s does not build, its tests will always run: %v\n%s", pkg, err, out)
		for _, name := range c.declaredTests(pkg) {
			c.add(&db.Test{Pkg: pkg, Name: name, Status: "fail"})
		}
		return
	}
	if _, err := os.Stat(bin); err != nil {
		return // no test files after build constraints
	}
	names, err := listTests(ctx, bin, dir)
	if err != nil {
		c.logf("hyrts: listing tests of %s: %v\n", pkg, err)
		return
	}
	start := time.Now()
	for _, name := range names {
		if ctx.Err() != nil {
			return
		}
		c.add(c.runOne(ctx, tmp, bin, dir, pkg, name))
	}
	c.logf("hyrts: %-60s %4d tests  %6.1fs\n", pkg, len(names), time.Since(start).Seconds())
}

// declaredTests lists test names from the snapshot, for packages that do not build.
func (c *collector) declaredTests(pkg string) []string {
	var out []string
	for rel, f := range c.snap {
		if f.Package == pkg && strings.HasSuffix(rel, "_test.go") {
			out = append(out, f.Tests...)
		}
	}
	return out
}

func listTests(ctx context.Context, bin, dir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, bin, "-test.list", ".")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, l := range strings.Split(string(out), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "Benchmark") || strings.HasPrefix(l, "ok ") {
			continue
		}
		names = append(names, l)
	}
	return names, nil
}

var (
	reFail  = regexp.MustCompile(`(?m)^--- FAIL: (\S+)`)
	reSkip  = regexp.MustCompile(`(?m)^--- SKIP: (\S+)`)
	reNoRun = regexp.MustCompile(`(?m)^testing: warning: no tests to run`)
)

func (c *collector) runOne(ctx context.Context, tmp, bin, dir, pkg, name string) *db.Test {
	base := filepath.Join(tmp, strings.NewReplacer("/", "_", ".", "_").Replace(pkg+"."+name))
	cov, tlog := base+".cov", base+".testlog"
	tctx, cancel := context.WithTimeout(ctx, c.o.Timeout+30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(tctx, bin,
		"-test.run", "^"+regexp.QuoteMeta(name)+"$",
		"-test.count=1", "-test.v",
		"-test.timeout="+c.o.Timeout.String(),
		"-test.coverprofile="+cov,
		"-test.testlogfile="+tlog,
	)
	cmd.Dir = dir
	start := time.Now()
	out, err := cmd.CombinedOutput()
	t := &db.Test{Pkg: pkg, Name: name, Seconds: time.Since(start).Seconds(), Status: "pass"}
	switch {
	case err != nil || reFail.Match(out):
		t.Status = "fail"
		c.logf("hyrts: %s.%s failed during collection; it will always be selected\n", pkg, name)
	case reNoRun.Match(out):
		t.Status = "norun" // e.g. an Example without an Output comment
	case skipped(out, name):
		t.Status = "skip"
	}
	t.Funcs = c.coverage(cov)
	t.Paths, t.Dirs, t.Env = c.testlog(tlog, dir)
	return t
}

func skipped(out []byte, name string) bool {
	for _, m := range reSkip.FindAllSubmatch(out, -1) {
		if string(m[1]) == name {
			return true
		}
	}
	return false
}

// coverage maps the covered blocks of a profile to function ids.
func (c *collector) coverage(path string) []int {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	hit := map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		// mod/pkg/file.go:12.3,14.2 1 1
		line := sc.Text()
		colon := strings.LastIndex(line, ":")
		if colon < 0 || strings.HasSuffix(line, " 0") {
			continue
		}
		file := line[:colon]
		var sl, sc2, el, ec, n, cnt int
		if _, err := fmt.Sscanf(line[colon+1:], "%d.%d,%d.%d %d %d", &sl, &sc2, &el, &ec, &n, &cnt); err != nil || cnt == 0 {
			continue
		}
		for _, s := range c.spans[file] {
			if sl >= s.start && sl <= s.end {
				hit[s.id] = true
				break
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ids := make([]int, 0, len(hit))
	for id := range hit {
		ids = append(ids, c.funcID(id))
	}
	sort.Ints(ids)
	return ids
}

// testlog reads the testing package's access log: lines "open p", "stat p",
// "chdir p" and "getenv k". Paths inside the module become dependencies.
func (c *collector) testlog(path, dir string) (paths, dirs, env []string) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil
	}
	defer f.Close()
	cwd := dir
	ps, ds, es := map[string]bool{}, map[string]bool{}, map[string]bool{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		op, arg, ok := strings.Cut(sc.Text(), " ")
		if !ok {
			continue
		}
		switch op {
		case "getenv":
			es[arg] = true
		case "chdir":
			if !filepath.IsAbs(arg) {
				arg = filepath.Join(cwd, arg)
			}
			cwd = filepath.Clean(arg)
		case "open", "stat":
			p := arg
			if !filepath.IsAbs(p) {
				p = filepath.Join(cwd, p)
			}
			rel, ok := c.rel(p)
			if !ok {
				continue
			}
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				ds[rel] = true
			} else {
				ps[rel] = true
			}
		}
	}
	return keys(ps), keys(ds), keys(es)
}

func (c *collector) rel(p string) (string, bool) {
	p = filepath.Clean(p)
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	rel, err := filepath.Rel(c.o.Root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if snapshot.Skip(rel) {
		return "", false
	}
	return rel, true
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func gitHead(root string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
