package main

import (
	"fmt"
	"strings"

	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func VTACallGraph() {
	for _, file := range []string{
		"testdata/src/callgraph_static.go",
		"testdata/src/callgraph_interfaces.go",
		"testdata/src/callgraph_pointers.go",
		"testdata/src/callgraph_recursive_types.go",
		//"package1/file1_test.go",
	} {
		prog, err := getProg(file, ssa.BuilderMode(0))
		if err != nil {
			fmt.Printf("couldn't load file '%s': %s", file, err)
		}

		fmt.Println("===", file)

		g := vta.CallGraph(ssautil.AllFunctions(prog), cha.CallGraph(prog))
		//g := cha.CallGraph(prog)
		//fmt.Println(g)
		got := callGraphStr(g)
		for _, gg := range got {
			fmt.Println(gg)
		}
	}
}

func VTACallGraphFromSSA() {

	prog, _ := GetSSA()
	//fmt.Println(pkg[0].Prog, "===")

	g := vta.CallGraph(ssautil.AllFunctions(prog), cha.CallGraph(prog))
	//g := cha.CallGraph(prog)

	for i, n := range g.Nodes {
		if strings.HasPrefix(i.String(), "proposal") {
			fmt.Println(i.Name(), "=", n)
		}
	}

	//got := callGraphStr(g)
	//for _, g := range got {
	//	fmt.Println("+", g)
	//}

}

func VTACallGraphWithPackage() {
	prog, main := example()

	cg := vta.CallGraph(ssautil.AllFunctions(prog), cha.CallGraph(prog))
	fmt.Println(main)
	fmt.Println("===")
	fmt.Println(cg)
}
