// Package cli implements the hyrts command line.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/dangdtr/go-hyrts/internal/collect"
	"github.com/dangdtr/go-hyrts/internal/db"
	"github.com/dangdtr/go-hyrts/internal/gotool"
	"github.com/dangdtr/go-hyrts/internal/runner"
	"github.com/dangdtr/go-hyrts/internal/selector"
)

const usage = `hyrts: hybrid regression test selection for Go

usage:
  hyrts collect [flags]            run every test once and record what it depends on
  hyrts select  [flags]            print the tests affected by changes since collect
  hyrts run     [flags] [-- go test flags]
                                   run only the affected tests

common flags:
  -C dir      module directory (default .)
  -db path    database (default <module>/.hyrts/db.json)
  -tags list  build tags, as for go test

collect flags:
  -update     re-collect only the tests affected since the last collect
              (falls back to a full collect when there is nothing to update)
  -j n        test processes at once, across packages (default GOMAXPROCS)
  -p n        test processes at once within one package (default 1: tests of
              a package often share fixtures; raise it if they are isolated)
  -timeout d  per-test timeout (default 10m)
  -pkg list   comma-separated import paths to collect (default all)

select/run flags:
  -explain    print why each test was selected
  -json       print the selection as JSON (select only)
  -always re  always select tests whose "importpath.Name" matches re
  -n          print the go test command without running it (run only)
  -update     after running, re-collect the selected tests (run only)
`

type common struct {
	dir, dbPath, tags string
}

func (c *common) register(fs *flag.FlagSet) {
	fs.StringVar(&c.dir, "C", ".", "")
	fs.StringVar(&c.dbPath, "db", "", "")
	fs.StringVar(&c.tags, "tags", "", "")
}

func (c *common) resolve() (root, dbPath string, err error) {
	root, err = gotool.ModuleRoot(c.dir)
	if err != nil {
		return "", "", err
	}
	dbPath = c.dbPath
	if dbPath == "" {
		dbPath = filepath.Join(root, db.DefaultPath)
	}
	return root, dbPath, nil
}

// Main runs the command and returns the exit code.
func Main(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "help" || args[0] == "--help" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	var err error
	switch args[0] {
	case "collect":
		err = cmdCollect(args[1:], stdout, stderr)
	case "select":
		err = cmdSelect(args[1:], stdout, stderr, false)
	case "run":
		err = cmdSelect(args[1:], stdout, stderr, true)
	default:
		err = fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		return exit.ExitCode()
	default:
		fmt.Fprintln(stderr, "hyrts:", err)
		return 2
	}
}

type collectFlags struct {
	jobs, perPkg int
	timeout      time.Duration
}

func (f *collectFlags) register(fs *flag.FlagSet) {
	fs.IntVar(&f.jobs, "j", runtime.GOMAXPROCS(0), "")
	fs.IntVar(&f.perPkg, "p", 1, "")
	fs.DurationVar(&f.timeout, "timeout", 10*time.Minute, "")
}

func (f *collectFlags) options(root, tags string, log io.Writer) collect.Options {
	return collect.Options{Root: root, Tags: tags, Jobs: f.jobs, PerPkg: f.perPkg,
		Builds: max(1, runtime.GOMAXPROCS(0)/2), Timeout: f.timeout, Log: log}
}

func cmdCollect(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("collect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var c common
	var cf collectFlags
	c.register(fs)
	cf.register(fs)
	update := fs.Bool("update", false, "")
	pkgList := fs.String("pkg", "", "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, dbPath, err := c.resolve()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	o := cf.options(root, c.tags, stderr)
	if *pkgList != "" {
		o.Packages = strings.Split(*pkgList, ",")
	}
	start := time.Now()
	if *update {
		old, r, err := selectTests(root, dbPath, c.tags, "")
		if err == nil && !r.All {
			n, err := incremental(ctx, o, dbPath, old, r)
			if err != nil {
				return err
			}
			fmt.Fprintf(stdout, "hyrts: updated %d of %d tests in %s -> %s\n", n, len(r.Tests), time.Since(start).Round(time.Second), dbPath)
			return nil
		}
		if err == nil {
			fmt.Fprintf(stderr, "hyrts: full collect needed: %s\n", r.AllReason)
		}
	}
	if old, err := db.Load(dbPath); err == nil {
		o.Prior = prior(old)
	}
	d, err := collect.Run(ctx, o)
	if err != nil {
		return err
	}
	if err := d.Save(dbPath); err != nil {
		return err
	}
	counts := map[string]int{}
	for _, t := range d.Tests {
		counts[t.Status]++
	}
	fmt.Fprintf(stdout, "hyrts: collected %d tests (%d pass, %d fail, %d skip) in %s -> %s\n",
		len(d.Tests), counts["pass"], counts["fail"], counts["skip"], time.Since(start).Round(time.Second), dbPath)
	return nil
}

// incremental re-collects the tests selected in r and merges them into old.
// Tests that were not selected keep their records: nothing they depend on
// changed, so their dependencies hold for the new snapshot too.
func incremental(ctx context.Context, o collect.Options, dbPath string, old *db.DB, r *selector.Result) (int, error) {
	o.Only = map[string]map[string]bool{}
	n := 0
	for id, t := range r.Tests {
		if r.Selected[id] == nil {
			continue
		}
		if o.Only[t.Pkg] == nil {
			o.Only[t.Pkg] = map[string]bool{}
		}
		o.Only[t.Pkg][t.Name] = true
		n++
	}
	o.Prior = prior(old)
	fresh, err := collect.Run(ctx, o)
	if err != nil {
		return 0, err
	}
	keep := make(map[string]bool, len(r.Tests))
	for id := range r.Tests {
		keep[id] = true
	}
	return n, db.Merge(old, fresh, keep).Save(dbPath)
}

// prior sums collected seconds per package, to schedule slow packages first.
func prior(d *db.DB) map[string]float64 {
	out := map[string]float64{}
	for _, t := range d.Tests {
		out[t.Pkg] += t.Seconds
	}
	return out
}

func cmdSelect(args []string, stdout, stderr io.Writer, run bool) error {
	var goFlags []string
	for i, a := range args {
		if a == "--" {
			args, goFlags = args[:i], args[i+1:]
			break
		}
	}
	name := "select"
	if run {
		name = "run"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	var c common
	c.register(fs)
	var cf collectFlags
	cf.register(fs)
	update := fs.Bool("update", false, "")
	explain := fs.Bool("explain", false, "")
	asJSON := fs.Bool("json", false, "")
	always := fs.String("always", "", "")
	dry := fs.Bool("n", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}
	goFlags = append(fs.Args(), goFlags...)
	root, dbPath, err := c.resolve()
	if err != nil {
		return err
	}
	start := time.Now()
	old, r, err := selectTests(root, dbPath, c.tags, *always)
	if err != nil {
		return err
	}
	if *asJSON && !run {
		return json.NewEncoder(stdout).Encode(jsonResult(r))
	}
	report(stdout, r, *explain, time.Since(start))
	if !run {
		for p, names := range r.ByPackage() {
			if !r.All {
				fmt.Fprintf(stdout, "%s %s\n", p, strings.Join(names, " "))
			}
		}
		return nil
	}
	if c.tags != "" {
		goFlags = append([]string{"-tags", c.tags}, goFlags...)
	}
	gargs := runner.Args(r, goFlags)
	if gargs == nil {
		fmt.Fprintln(stdout, "hyrts: no test is affected; nothing to run")
		if *update {
			_, err := incremental(context.Background(), cf.options(root, c.tags, stderr), dbPath, old, r)
			return err
		}
		return nil
	}
	fmt.Fprintln(stderr, "hyrts: go", shellJoin(gargs))
	if *dry {
		return nil
	}
	cmd := exec.Command("go", gargs...)
	cmd.Dir, cmd.Stdout, cmd.Stderr, cmd.Stdin = root, stdout, stderr, os.Stdin
	runErr := cmd.Run()
	if *update {
		// Like HyRTS's offline mode: results first, dependencies afterwards.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		ustart := time.Now()
		o := cf.options(root, c.tags, stderr)
		if r.All {
			d, err := collect.Run(ctx, o)
			if err == nil {
				err = d.Save(dbPath)
			}
			if err != nil {
				return err
			}
		} else if _, err := incremental(ctx, o, dbPath, old, r); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "hyrts: dependencies updated in %s\n", time.Since(ustart).Round(time.Second))
	}
	return runErr
}

func selectTests(root, dbPath, tags, always string) (*db.DB, *selector.Result, error) {
	old, err := db.Load(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("%v (run `hyrts collect` first)", err)
	}
	g, cur, err := collect.Load(root, tags)
	if err != nil {
		return nil, nil, err
	}
	o := selector.Options{Tags: tags}
	if always != "" {
		if o.Always, err = regexp.Compile(always); err != nil {
			return nil, nil, fmt.Errorf("-always: %v", err)
		}
	}
	return old, selector.Select(old, g, cur, gotool.Version(root), o), nil
}

func report(w io.Writer, r *selector.Result, explain bool, took time.Duration) {
	n, total, sec, totSec := r.Summary()
	if r.All {
		fmt.Fprintf(w, "hyrts: running all %d tests: %s\n", total, r.AllReason)
		return
	}
	pct := 0.0
	if totSec > 0 {
		pct = sec / totSec * 100
	}
	fmt.Fprintf(w, "hyrts: %d changed files, selected %d/%d tests in %d packages (~%.0fs of %.0fs, %.0f%%) in %s\n",
		len(r.Changed), n, total, len(r.ByPackage()), sec, totSec, pct, took.Round(time.Millisecond))
	if !explain {
		return
	}
	ids := make([]string, 0, len(r.Selected))
	for id := range r.Selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Fprintf(w, "  %s\n", id)
		for _, why := range r.Selected[id] {
			fmt.Fprintf(w, "      %s\n", why)
		}
	}
	if len(r.Unused) > 0 {
		fmt.Fprintf(w, "changed files no collected test depends on:\n")
		for _, u := range r.Unused {
			fmt.Fprintf(w, "  %s\n", u)
		}
	}
}

func jsonResult(r *selector.Result) any {
	type sel struct {
		ID      string   `json:"id"`
		Package string   `json:"package"`
		Name    string   `json:"name"`
		Reasons []string `json:"reasons,omitempty"`
	}
	out := struct {
		All       bool     `json:"all"`
		AllReason string   `json:"all_reason,omitempty"`
		Changed   []string `json:"changed"`
		Unused    []string `json:"unused,omitempty"`
		Selected  []sel    `json:"selected"`
	}{All: r.All, AllReason: r.AllReason, Changed: r.Changed, Unused: r.Unused, Selected: []sel{}}
	for id, t := range r.Tests {
		if r.All || r.Selected[id] != nil {
			out.Selected = append(out.Selected, sel{id, t.Pkg, t.Name, r.Selected[id]})
		}
	}
	sort.Slice(out.Selected, func(i, j int) bool { return out.Selected[i].ID < out.Selected[j].ID })
	return out
}

func shellJoin(args []string) string {
	q := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " ()|$^*?[]\"'") {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		q[i] = a
	}
	return strings.Join(q, " ")
}
