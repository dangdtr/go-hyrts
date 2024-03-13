package main

import (
	"fmt"
	"go/parser"
	"log"
	"os"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var (
	once  sync.Once
	prog  *ssa.Program
	mainF *ssa.Function
)

func example() (*ssa.Program, *ssa.Function) {
	once.Do(func() {
		var conf loader.Config
		f, err := conf.ParseFile("<input>", nil)
		if err != nil {
			log.Fatal(err)
		}
		conf.CreateFromFiles(f.Name.Name, f)

		lprog, err := conf.Load()
		if err != nil {
			log.Fatalf("test 'package %s': Load: %s", f.Name.Name, err)
		}
		prog = ssautil.CreateProgram(lprog, ssa.InstantiateGenerics)
		prog.Build()

		mainF = prog.Package(lprog.Created[0].Pkg).Members["main"].(*ssa.Function)
	})
	return prog, mainF
}

// getProg returns an ssa representation of a program at `path`
func getProg(path string, mode ssa.BuilderMode) (*ssa.Program, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := loader.Config{
		ParserMode: parser.ParseComments,
	}

	f, err := conf.ParseFile(path, content)
	if err != nil {
		return nil, err
	}

	conf.CreateFromFiles("testdata", f)
	iprog, err := conf.Load()
	if err != nil {
		return nil, err
	}

	prog := ssautil.CreateProgram(iprog, mode)
	// Set debug mode to exercise DebugRef instructions.
	prog.Package(iprog.Created[0].Pkg).SetDebugMode(true)
	prog.Build()
	return prog, nil
}

// callGraphStr helper
func callGraphStr(g *callgraph.Graph) []string {
	var gs []string
	for f, n := range g.Nodes {
		c := make(map[string][]string)
		for _, edge := range n.Out {
			cs := edge.Site.String()
			c[cs] = append(c[cs], funcName(edge.Callee.Func))
		}

		var cs []string
		for site, fs := range c {
			sort.Strings(fs)
			entry := fmt.Sprintf("%v -> %v", site, strings.Join(fs, ", "))
			cs = append(cs, entry)
		}

		sort.Strings(cs)
		entry := fmt.Sprintf("%v: %v", funcName(f), strings.Join(cs, "; "))
		gs = append(gs, entry)
	}
	return gs
}

// funcName returns a name of the function `f`
func funcName(f *ssa.Function) string {
	recv := f.Signature.Recv()
	if recv == nil {
		return f.Name()
	}
	tp := recv.Type().String()
	return tp[strings.LastIndex(tp, ".")+1:] + "." + f.Name()
}
