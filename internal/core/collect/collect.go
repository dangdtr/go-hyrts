package collect

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

//type Deps map[string]bool

type cov struct {
	testCovMap    map[string]map[string]string
	newFileMeths  map[string]map[string]string
	testFuncCount int
}

type Cov interface {
	Run()
	GetTestCovMap() map[string]map[string]string
	GetTestFuncCount() int
}

func NewCov(newFileMeths map[string]map[string]string) Cov {
	return &cov{
		testCovMap:    make(map[string]map[string]string),
		newFileMeths:  newFileMeths,
		testFuncCount: 0,
	}
}

func (t *cov) GetTestCovMap() map[string]map[string]string {
	return t.testCovMap
}
func (t *cov) GetTestFuncCount() int {
	return t.testFuncCount
}

func (t *cov) Run() {
	t.collectTestCov(util.ProgramPath)
}

func (t *cov) collectTestCov(rootDir string) {
	t.testFuncCount = 0
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, util.GoTestExt) {
			return nil
		}

		//deps := t.collectDepsCov(path)
		fs := token.NewFileSet()
		node, err := parser.ParseFile(fs, path, nil, parser.AllErrors)
		if err != nil {
			fmt.Printf("Error parsing file %s: %v\n", path, err)
			return nil
		}

		shortPath := util.ShortPath(path)

		// Map to store imported packages
		imports := make(map[string]string)
		for _, imp := range node.Imports {
			path_t := strings.Trim(imp.Path.Value, `"`)
			name := ""
			if imp.Name != nil {
				name = imp.Name.Name
			} else {
				name = path_t[strings.LastIndex(path_t, "/")+1:]
			}
			imports[name] = path_t
		}

		deps := make(map[string]string)

		for _, decl := range node.Decls {

			switch d := decl.(type) {
			case *ast.FuncDecl:

				if strings.HasPrefix(strings.ToLower(d.Name.Name), "test") || strings.HasPrefix(strings.ToLower(d.Name.Name), "benchmark") || strings.HasPrefix(strings.ToLower(d.Name.Name), "fuzz") {
					t.testFuncCount++
					ast.Inspect(d.Body, func(n ast.Node) bool {
						// Check if the node is a function call expression
						//fmt.Println(n)
						callExpr, ok := n.(*ast.CallExpr)
						if !ok {
							return true
						}

						var funcName string
						var pkgAlias string

						switch fun := callExpr.Fun.(type) {
						case *ast.Ident:
							funcName = fun.Name
							pkgg := t.findPackage(imports, fun.Name)
							if len(pkgg) == 0 {
								pkgg = append(pkgg, node.Name.Name)
							}

							for _, p := range pkgg {
								if util.StandardLibraries[p] {
									continue
								}
								key := fmt.Sprintf("%s:%s:%d", p, funcName, fs.Position(fun.NamePos).Line)
								deps[key] = d.Name.Name
							}

						case *ast.SelectorExpr:
							if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
								switch selExpr.X.(type) {
								case *ast.Ident:
									if _, ok := selExpr.X.(*ast.Ident); ok {
										//fmt.Printf("Package: %s, Method: %s\n", ident.Name, selExpr.Sel.Name)
										pkgAlias = selExpr.X.(*ast.Ident).Name
										funcName = selExpr.Sel.Name

										pkgg := t.findPackage(imports, pkgAlias)
										if len(pkgg) == 0 {
											pkgg = append(pkgg, node.Name.Name)
										}

										for _, p := range pkgg {
											if util.StandardLibraries[p] {
												continue
											}
											key := fmt.Sprintf("%s:%s:%d", p, funcName, fs.Position(selExpr.Sel.NamePos).Line)
											deps[key] = d.Name.Name
										}

									}
								case *ast.SelectorExpr:
									funcName = selExpr.Sel.Name

									pkgg := t.findPackage(imports, funcName)
									if len(pkgg) == 0 {
										pkgg = append(pkgg, node.Name.Name)
									}

									for _, p := range pkgg {
										if util.StandardLibraries[p] {
											continue
										}
										key := fmt.Sprintf("%s:%s:%d", p, funcName, fs.Position(selExpr.Sel.NamePos).Line)
										deps[key] = d.Name.Name
									}
								}
							}

							funcName = fun.Sel.Name
							ident, ok := fun.X.(*ast.Ident)
							if !ok {
								return true
							}
							pkgAlias = ident.Name

						}
						_ = t.findPackage(imports, pkgAlias)
						_ = funcName

						return true
					})

				}

			}
		}

		t.testCovMap[shortPath] = deps
		return nil
	})
	if err != nil {
		return
	}

}

func (t *cov) findPackage(imports map[string]string, funcName string) []string {
	rs := make([]string, 0)
	for name, path := range imports {
		if name == funcName {
			rs = append(rs, path)
		}
	}

	for key, fnMap := range t.newFileMeths {
		if _, ok := fnMap[funcName]; ok {
			rs = append(rs, key)
		}
	}

	return rs
}
