package coverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/dangdtr/go-hyrts/internal/core/util"
)

type Deps map[string]string

type cov struct {
	testFileCov map[string]Deps
}

type Cov interface {
	Run()
}

func NewCov() Cov {
	return &cov{
		testFileCov: make(map[string]Deps),
	}
}

func (t *cov) Run() {
	t.collectTestCov(util.ProgramPath)
}

func (t *cov) collectTestCov(rootDir string) {

	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, util.GoExt) {
			return nil
		}

		fs := token.NewFileSet()
		node, err := parser.ParseFile(fs, path, nil, parser.AllErrors)
		if err != nil {
			fmt.Printf("Error parsing file %s: %v\n", path, err)
			return nil
		}
		deps := make(Deps)

		for _, decl := range node.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {

						deps[path] = typeSpec.Name.Name

					}
				}
			}
			switch d := decl.(type) {
			case *ast.FuncDecl:
				functionName := d.Name.Name

				deps[path] = functionName
			}

			//switch decl.(type) {
			//case *ast.FuncDecl:
			//	funcDecl, ok := decl.(*ast.FuncDecl)
			//	if !ok || funcDecl.Recv != nil {
			//		continue
			//	}
			//
			//	if strings.HasPrefix(funcDecl.Name.Name, util.TestPrefix) {
			//		deps := t.collectDependenciesCov(funcDecl, path)
			//		keyCov := fmt.Sprintf("%s-%s", path, funcDecl.Name.Name)
			//
			//		t.testFileCov[keyCov] = deps
			//
			//	}
			//}

			//funcDecl, ok := decl.(*ast.FuncDecl)
			//if !ok || funcDecl.Recv != nil {
			//	continue
			//}
			//
			//if strings.HasPrefix(funcDecl.Name.Name, util.TestPrefix) {
			//	deps := t.collectDependenciesCov(funcDecl, rootDir)
			//	keyCov := fmt.Sprintf("%s-%s", path, funcDecl.Name.Name)
			//	t.testFileCov[keyCov] = deps
			//}
		}

		return nil
	})

}

func (t *cov) collectDependenciesCov(funcDecl *ast.FuncDecl, packagePath string) Deps {
	deps := make(Deps)

	//ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
	//	callExpr, ok := node.(*ast.CallExpr)
	//	if !ok {
	//		return true
	//	}
	//
	//	if ident, ok := callExpr.Fun.(*ast.Ident); ok {
	//		deps[packagePath] = ident.Name
	//	}
	//
	//	return true
	//})

	//functionName := funcDecl.Name.Name
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if fun, ok := x.Fun.(*ast.Ident); ok {
				//pkg := fun.Sel.Obj.Decl.(*ast.FuncDecl).Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
				//fmt.Printf("Function call: %s.%s\n", pkg, fun.Sel.Name)
				deps[packagePath] = fun.Name
			}

		case *ast.GenDecl:
			//if strings.HasPrefix(x.Name.Name, functionName) {
			//	fmt.Printf("Type declaration: %s\n", x.Name.Name)
			//	deps[packagePath] = x.Name.Name
			//}

			if genDecl, ok := n.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {

						deps[packagePath] = typeSpec.Name.Name
					}
				}
			}
		}
		return true
	})

	return deps
}
