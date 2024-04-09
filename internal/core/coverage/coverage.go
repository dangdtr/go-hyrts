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

//type Deps map[string]bool

type cov struct {
	testCovMap   map[string]map[string]bool
	newFileMeths map[string]map[string]string
}

type Cov interface {
	Run()
	GetTestCovMap() map[string]map[string]bool
}

func NewCov(newFileMeths map[string]map[string]string) Cov {
	return &cov{
		testCovMap:   make(map[string]map[string]bool),
		newFileMeths: newFileMeths,
	}
}

func (t *cov) GetTestCovMap() map[string]map[string]bool {
	return t.testCovMap
}

func (t *cov) Run() {
	t.collectTestCov(util.ProgramPath)
}

func (t *cov) collectTestCov(rootDir string) {

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, util.GoTestExt) {
			return nil
		}

		//if path != "/Users/dangdt/teko/footprint/golang/usersegmentv2/pkg/segment/repo_test.go" {
		//	return nil
		//}

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
			path := strings.Trim(imp.Path.Value, `"`)
			name := ""
			if imp.Name != nil {
				name = imp.Name.Name
			} else {
				name = path[strings.LastIndex(path, "/")+1:]
			}
			imports[name] = path
		}

		//deps := make(Deps)
		//covFunc := make(map[string]Deps)
		deps := make(map[string]bool)

		for _, decl := range node.Decls {

			switch d := decl.(type) {
			case *ast.FuncDecl:

				if strings.HasPrefix(d.Name.Name, util.TestPrefix) {
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
							if pkgg == "unknown" {
								pkgg = node.Name.Name
							}
							key := fmt.Sprintf("%s:%s", pkgg, funcName)
							deps[key] = true

						case *ast.SelectorExpr:
							if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
								switch selExpr.X.(type) {
								case *ast.Ident:
									if _, ok := selExpr.X.(*ast.Ident); ok {
										//fmt.Printf("Package: %s, Method: %s\n", ident.Name, selExpr.Sel.Name)
										funcName = selExpr.Sel.Name

										pkgg := t.findPackage(imports, funcName)
										if pkgg == "unknown" {
											pkgg = node.Name.Name
										}

										key := fmt.Sprintf("%s:%s", pkgg, funcName)

										deps[key] = true

										//deps[fmt.Sprint(pkgg+":"+selExpr.Sel.Name)] = true

									}
								case *ast.SelectorExpr:
									//if sel, ok := selExpr.X.(*ast.SelectorExpr); ok {
									funcName = selExpr.Sel.Name

									pkgg := t.findPackage(imports, selExpr.Sel.Name)

									if pkgg == "unknown" {
										pkgg = node.Name.Name
									}

									key := fmt.Sprintf("%s:%s", pkgg, funcName)

									deps[key] = true

									//fmt.Printf("Package: %s, Method: %s\n", sel.Name, selExpr.Sel.Name)
									//}
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

						//fmt.Printf("\tArguments:\n")
						//fmt.Printf("\t\tfuncName: %s\n", funcName)
						//fmt.Printf("\t\tPackage: %s\n", pkg)
						//
						//for _, arg := range callExpr.Args {
						//	fmt.Printf("\t\t\t%s\n", arg)
						//}

						_ = funcName
						//deps[keyDeps] = true

						return true
					})

				}

				//functionName := d.Name.Name
				//
				//deps[path] = functionName
				//keyDeps := shortPath + "-" + functionName
				//deps[keyDeps] = functionName
				//deps[keyDeps] = true
			}
		}

		t.testCovMap[shortPath] = deps
		return nil
	})
	if err != nil {
		return
	}

}

//
//func (t *cov) collectTestCov(rootDir string) {
//
//	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
//		if err != nil || info.IsDir() || !strings.HasSuffix(path, util.GoTestExt) {
//			return nil
//		}
//
//		//deps := t.collectDepsCov(path)
//		fs := token.NewFileSet()
//		node, err := parser.ParseFile(fs, path, nil, 0)
//		if err != nil {
//			fmt.Printf("Error parsing file %s: %v\n", path, err)
//			return nil
//		}
//
//		shortPath := util.ShortPath(path)
//
//		// Map to store imported packages
//		imports := make(map[string]string)
//		for _, imp := range node.Imports {
//			path := strings.Trim(imp.Path.Value, `"`)
//			name := ""
//			if imp.Name != nil {
//				name = imp.Name.Name
//			} else {
//				name = path[strings.LastIndex(path, "/")+1:]
//			}
//			imports[name] = path
//		}
//
//		deps := make(Deps)
//
//		for _, decl := range node.Decls {
//
//			switch d := decl.(type) {
//			case *ast.FuncDecl:
//				if strings.HasPrefix(d.Name.Name, util.TestPrefix) {
//					ast.Inspect(d.Body, func(n ast.Node) bool {
//						callExpr, ok := n.(*ast.CallExpr)
//						if !ok {
//							return true
//						}
//
//						var funcName string
//						var pkgAlias string
//
//						switch fun := callExpr.Fun.(type) {
//						case *ast.Ident:
//							funcName = fun.Name
//
//						case *ast.SelectorExpr:
//							funcName = fun.Sel.Name
//							ident, ok := fun.X.(*ast.Ident)
//							if !ok {
//								return true
//							}
//							pkgAlias = ident.Name
//
//							//ast.Inspect(fun, func(n ast.Node) bool {
//							//	fmt.Println(n)
//							//	return true
//							//})
//
//						}
//
//						_ = findPackage(imports, pkgAlias)
//						//
//						_ = funcName
//						//deps[keyDeps] = true
//
//						return true
//					})
//
//				}
//
//				functionName := d.Name.Name
//
//				//deps[path] = functionName
//				keyDeps := shortPath + "-" + functionName
//				//deps[keyDeps] = functionName
//				deps[keyDeps] = true
//			}
//		}
//
//		t.testCovMap[shortPath] = deps
//		return nil
//	})
//	if err != nil {
//		return
//	}
//
//}

//func (t *cov) collectDependenciesCov(funcDecl *ast.FuncDecl, packagePath string) Deps {
//	deps := make(Deps)
//
//	//ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
//	//	callExpr, ok := node.(*ast.CallExpr)
//	//	if !ok {
//	//		return true
//	//	}
//	//
//	//	if ident, ok := callExpr.Fun.(*ast.Ident); ok {
//	//		deps[packagePath] = ident.Name
//	//	}
//	//
//	//	return true
//	//})
//
//	//functionName := funcDecl.Name.Name
//	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
//		switch x := n.(type) {
//		case *ast.CallExpr:
//			if fun, ok := x.Fun.(*ast.Ident); ok {
//				//pkg := fun.Sel.Obj.Decl.(*ast.FuncDecl).Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name
//				//fmt.Printf("Function call: %s.%s\n", pkg, fun.Sel.Name)
//				//deps[packagePath] = fun.Name
//				keyDeps := packagePath + "-" + fun.Name
//
//				deps[keyDeps] = true
//			}
//
//		case *ast.GenDecl:
//			//if strings.HasPrefix(x.Name.Name, functionName) {
//			//	fmt.Printf("Type declaration: %s\n", x.Name.Name)
//			//	deps[packagePath] = x.Name.Name
//			//}
//
//			if genDecl, ok := n.(*ast.GenDecl); ok {
//				for _, spec := range genDecl.Specs {
//					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
//						keyDeps := packagePath + "-" + typeSpec.Name.Name
//
//						deps[keyDeps] = true
//
//						//deps[packagePath] = typeSpec.Name.Name
//					}
//				}
//			}
//		}
//		return true
//	})
//
//	return deps
//}

func (t *cov) collectDepsCov(filePath string) map[string]bool {
	//fset := token.NewFileSet()
	//node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	//if err != nil {
	//	return nil
	//}
	//
	//meths := make(map[string]bool)
	//
	//for _, decl := range node.Decls {
	//	switch d := decl.(type) {
	//	case *ast.FuncDecl:
	//		functionName := d.Name.Name
	//		functionStart := fset.Position(d.Pos()).Offset
	//		functionEnd := fset.Position(d.End()).Offset
	//		functionContent := v.readFileContent(filePath, functionStart, functionEnd)
	//		functionChecksum := checksum.Calculate([]byte(functionContent))
	//
	//		meths[functionName] = functionChecksum
	//	}
	//}

	return nil
}

func (t *cov) findPackage(imports map[string]string, funcName string) string {
	for name, path := range imports {
		if name == funcName {
			return path
		}
	}

	for key, fnMap := range t.newFileMeths {
		if _, ok := fnMap[funcName]; ok {
			return key
		}
	}
	return "unknown"
}
