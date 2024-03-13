package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type MethodInfo struct {
	Package string
	File    string
	Method  string
	Deps    []string
}

func main3() {
	rootDir := "/Users/dangdt/Documents/coding/go-hyrts/proposal2/src"

	covData := collectTestCoverage(rootDir)

	//for _, method := range covData {
	//	fmt.Printf("%s:%s:%s\n", method.Package, method.File, method.Method)
	//}

	outputDir := "meth-cov"
	err := os.Mkdir(outputDir, 0755)
	if err != nil && !os.IsExist(err) {
		fmt.Println("Error creating output directory:", err)
		return
	}

	for _, method := range covData {
		outputFileName := fmt.Sprintf("%s-%s", method.File, method.Method)
		outputFileName = strings.Replace(outputFileName, "/", "-", -1)
		outputPath := filepath.Join(outputDir, outputFileName)

		outputFile, err := os.Create(outputPath)
		if err != nil {
			fmt.Println("Error creating output file:", err)
			continue
		}
		defer outputFile.Close()

		//methodFile := strings.ReplaceAll(method.File, "/", ".")
		testName := fmt.Sprintf("%s-%s", strings.ReplaceAll(method.File, "/", "."), method.Method)
		outputFile.WriteString(testName + "\n")

		for _, dep := range method.Deps {
			fmt.Println(dep)
			outputFile.WriteString(dep + "\n")
		}

		fmt.Println("Wrote to file:", outputPath)
	}
}

func collectTestCoverage(rootDir string) []MethodInfo {
	var methods []MethodInfo

	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			fmt.Printf("Error parsing file %s: %v\n", path, err)
			return nil
		}

		var packageName string
		if node.Name != nil {
			packageName = node.Name.Name
		}

		// Duyệt các hàm trong file
		for _, decl := range node.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if !ok || funcDecl.Recv != nil {
				continue
			}
			//fmt.Println(funcDecl.Name)
			//
			//fmt.Println(funcDecl.Body.List[0])

			if strings.HasPrefix(funcDecl.Name.Name, "Test") {
				depMethods := collectDependenciesCov(funcDecl, rootDir)

				methodInfo := MethodInfo{
					Package: packageName,
					File:    path,
					Method:  funcDecl.Name.Name,
					Deps:    depMethods,
				}

				methods = append(methods, methodInfo)

			}
		}

		return nil
	})

	return methods
}

func collectDependenciesCov(funcDecl *ast.FuncDecl, packagePath string) []string {
	var deps []string

	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
			dep := fmt.Sprintf("%s:%s", packagePath, ident.Name)
			deps = append(deps, dep)
		}

		return true
	})

	return deps
}

//func main() {
//	rootDir := "/Users/dangdt/Documents/coding/go-hyrts/proposal2/src"
//	outputDir := "meth-cov"
//	err := os.Mkdir(outputDir, 0755)
//	if err != nil && !os.IsExist(err) {
//		fmt.Println("Error creating output directory:", err)
//		return
//	}
//
//	files, err := getGoFiles(rootDir)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for _, file := range files {
//		fset := token.NewFileSet()
//		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
//		if err != nil {
//			log.Fatalf("Error parsing file %s: %s\n", file, err)
//		}
//
//		tests := collectTests(node)
//		for _, test := range tests {
//			deps := collectDependenciesCov(test, node.Name.Name, file)
//			fmt.Printf("Test: %s\n", test.Name.Name)
//			fmt.Println("Dependencies:")
//			for _, dep := range deps {
//				fmt.Println(dep)
//			}
//			fmt.Println("-----------")
//
//			writeToOutputFile(outputDir, test, deps)
//
//		}
//	}
//}
//
//func getGoFiles(rootDir string) ([]string, error) {
//	var goFiles []string
//	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
//		if err != nil {
//			return err
//		}
//		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
//			goFiles = append(goFiles, path)
//		}
//		return nil
//	})
//	if err != nil {
//		return nil, err
//	}
//	return goFiles, nil
//}
//
//func collectTests(node *ast.File) []*ast.FuncDecl {
//	var tests []*ast.FuncDecl
//
//	for _, decl := range node.Decls {
//		if fd, ok := decl.(*ast.FuncDecl); ok {
//			if strings.HasPrefix(fd.Name.Name, "Test") {
//				tests = append(tests, fd)
//			}
//		}
//	}
//
//	return tests
//}
//
//func collectDependenciesCov(funcDecl *ast.FuncDecl, packageName string, filePath string) []string {
//	var deps []string
//
//	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
//		callExpr, ok := node.(*ast.CallExpr)
//		if !ok {
//			return true
//		}
//
//		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
//			//dep := fmt.Sprintf("%s:%s", packageName, ident.Name
//			dep := fmt.Sprintf("%s:%s:%s", packageName, filePath, ident.Name)
//
//			deps = append(deps, dep)
//		}
//
//		return true
//	})
//
//	return deps
//}
//
//func writeToOutputFile(outputDir string, funcDecl *ast.FuncDecl, deps []string) {
//	outputFileName := fmt.Sprintf("%s-%s", funcDecl.Name.Name, "dependencies.txt")
//	outputFileName = strings.Replace(outputFileName, "/", "-", -1)
//	outputPath := filepath.Join(outputDir, outputFileName)
//
//	outputFile, err := os.Create(outputPath)
//	if err != nil {
//		fmt.Println("Error creating output file:", err)
//		return
//	}
//	defer outputFile.Close()
//
//	outputFile.WriteString(fmt.Sprintf("Test: %s\n", funcDecl.Name.Name))
//	outputFile.WriteString("Dependencies:\n")
//	for _, dep := range deps {
//		outputFile.WriteString(dep + "\n")
//	}
//
//	fmt.Println("Wrote to file:", outputPath)
//}
