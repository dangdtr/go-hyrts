// parser parses the go programs in the given paths and prints
// the top five most common names of local variables and variables
// defined at package level.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"sort"

	"github.com/davecgh/go-spew/spew"
)

func main7() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage:\n\t%s [files]\n", os.Args[0])
		os.Exit(1)
	}

	fs := token.NewFileSet()
	//locals, globals := make(map[string]int), make(map[string]int)

	for _, arg := range os.Args[1:] {
		f, err := parser.ParseFile(fs, arg, nil, parser.AllErrors)
		if err != nil {
			log.Printf("could not parse %s: %v", arg, err)
			continue
		}

		// Tạo Visitor để tìm kiếm các CallExpr
		visitor := &CallExprVisitor{}

		// Duyệt qua AST
		ast.Walk(visitor, f)

		// In ra tên package và tên function được gọi
		for _, callExpr := range visitor.CallExprs {
			//fmt.Println("===============")
			//spew.Dump(callExpr.Fun)
			//fmt.Println("===============\n")

			if ident, ok := callExpr.Fun.(*ast.Ident); ok {
				fmt.Println("Function:", ident.Name) // No package name for same-package calls
			} else if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := selExpr.X.(*ast.Ident); ok {
					fmt.Println("Package:", ident.Name)
					fmt.Println("Function:", selExpr.Sel.Name)
				}
			}
		}
		//ast.Inspect(f, func(n ast.Node) bool {
		//	switch x := n.(type) {
		//	case *ast.CallExpr:
		//		id, ok := x.Fun.(*ast.Ident)
		//		if ok {
		//			fmt.Print(id, " ")
		//			fmt.Printf("Inspect found call to pred() at %s\n", fs.Position(n.Pos()))
		//		}
		//
		//		if selExpr, ok := x.Fun.(*ast.SelectorExpr); ok {
		//			if ident, ok := selExpr.X.(*ast.Ident); ok {
		//				fmt.Println(ident.Name)
		//			}
		//		}
		//	}
		//	return true
		//})

		//spew.Dump(f)
		//
		//v := newVisitor(f)
		//ast.Walk(v, f)
		//for k, v := range v.locals {
		//	locals[k] += v
		//}
		//for k, v := range v.globals {
		//	globals[k] += v
		//}
	}

	//fmt.Println("most common local variable names")
	//printTopFive(locals)
	//fmt.Println("most common global variable names")
	//printTopFive(globals)
}

// CallExprVisitor là Visitor để tìm kiếm các CallExpr
type CallExprVisitor struct {
	CallExprs []*ast.CallExpr
}

// Visit implements ast.Visitor interface
func (v *CallExprVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.CallExpr:
		v.CallExprs = append(v.CallExprs, node)
	}

	return v
}

/////

func printTopFive(counts map[string]int) {
	type pair struct {
		s string
		n int
	}
	pairs := make([]pair, 0, len(counts))
	for s, n := range counts {
		pairs = append(pairs, pair{s, n})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].n > pairs[j].n })

	for i := 0; i < len(pairs) && i < 5; i++ {
		fmt.Printf("%6d %s\n", pairs[i].n, pairs[i].s)
	}
}

type visitor struct {
	pkgDecl map[*ast.GenDecl]bool
	locals  map[string]int
	globals map[string]int
}

func newVisitor(f *ast.File) visitor {
	decls := make(map[*ast.GenDecl]bool)
	for _, decl := range f.Decls {
		if v, ok := decl.(*ast.GenDecl); ok {
			decls[v] = true
		}
	}

	return visitor{
		decls,
		make(map[string]int),
		make(map[string]int),
	}
}

func (v visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch d := n.(type) {
	case *ast.AssignStmt:

		if d.Tok != token.DEFINE {
			return v
		}
		for _, name := range d.Lhs {
			v.local(name)
		}
	case *ast.RangeStmt:
		v.local(d.Key)
		v.local(d.Value)
	case *ast.Ident:

	case *ast.CallExpr:
		fmt.Println("===============")
		spew.Dump(d.Fun)
		v.local(d.Fun)
		fmt.Println("===============\n")

	case *ast.BlockStmt:
		//for _, fe := range d.List {
		//	fmt.Println("===============")
		//	spew.Dump(fe)
		//	//fmt.Println(fe.(*ast.ExprStmt))
		//	fmt.Println("===============\n")
		//
		//}

	case *ast.FuncDecl:
		//fmt.Println(d.Body)

		if d.Recv != nil {
			v.localList(d.Recv.List)
		}
		v.localList(d.Type.Params.List)
		if d.Type.Results != nil {
			v.localList(d.Type.Results.List)
		}
	case *ast.GenDecl:
		if d.Tok != token.VAR {
			return v
		}
		for _, spec := range d.Specs {
			if value, ok := spec.(*ast.ValueSpec); ok {
				for _, name := range value.Names {
					if name.Name == "_" {
						continue
					}
					if v.pkgDecl[d] {
						v.globals[name.Name]++
					} else {
						v.locals[name.Name]++
					}
				}
			}
		}
	}

	return v
}

func (v visitor) local(n ast.Node) {
	ident, ok := n.(*ast.Ident)
	if !ok {
		return
	}
	if ident.Name == "_" || ident.Name == "" {
		return
	}
	if ident.Obj != nil && ident.Obj.Pos() == ident.Pos() {
		v.locals[ident.Name]++
	}
}

func (v visitor) localList(fs []*ast.Field) {
	for _, f := range fs {
		for _, name := range f.Names {
			v.local(name)
		}
	}
}
