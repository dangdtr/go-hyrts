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

func collect() {
	// Define the directory to search for test files
	projectDir, _ := filepath.Abs(".") //os.Getwd()
	fmt.Println(projectDir)

	// Open output file
	outputFile, err := os.Create("test_dependencies.txt")
	if err != nil {
		fmt.Println("Error opening output file:", err)
		return
	}
	defer outputFile.Close()

	// Traverse the project directory
	err = filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "_test.go") { // Process only test files
			processTestFile(path, outputFile)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error traversing project directory:", err)
		return
	}
}

func processTestFile(filePath string, outputFile *os.File) {
	// Parse the test file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Println("Error parsing file:", err)
		return
	}

	// Extract dependencies for each test function
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok && strings.HasPrefix(fd.Name.Name, "Test") {
			testName := fd.Name.Name
			fmt.Fprintf(outputFile, "%s", testName)
			//fmt.Printf("%s", testName)

			// Assuming dependencies are method calls inside the test function body
			// You might need to customize this part based on your project's structure
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				if ce, ok := n.(*ast.CallExpr); ok {
					if se, ok := ce.Fun.(*ast.SelectorExpr); ok {
						dependency := fmt.Sprintf("%s.%s", se.X.(*ast.Ident).Name, se.Sel.Name)
						fmt.Fprintf(outputFile, ":%s", dependency)
						fmt.Println(ce.Fun)

					}
				}
				return true
			})
			fmt.Fprintln(outputFile) // Add newline after printing dependencies for a test function
		}
	}
}
