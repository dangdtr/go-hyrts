package diff

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dangdtr/go-hyrts/internal/core/checksum"
	"github.com/dangdtr/go-hyrts/internal/core/util"
)

func (v *versionDiff) deserializeOldContents() {

	filePath := util.ProgramPath + "/HyRTS-checksum.txt"
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		fileChecksumParts := strings.Split(parts[0], "=")
		if len(fileChecksumParts) != 2 {
			continue
		}

		v.oldFiles[fileChecksumParts[0]] = fileChecksumParts[1]

		methodMap := make(map[string]string)

		for _, part := range parts[1:] {
			funcChecksumParts := strings.Split(part, "=")
			if len(funcChecksumParts) != 2 {
				continue
			}
			methodMap[funcChecksumParts[0]] = funcChecksumParts[1]
		}

		v.oldFileMeths[fileChecksumParts[0]] = methodMap

	}
}

func (v *versionDiff) serializeNewContents() {

	filePath := util.ProgramPath + "/HyRTS-checksum.txt"
	outputFile, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Lỗi khi tạo file:", err)
		return
	}
	defer outputFile.Close()

	for key, value := range v.newFiles {
		_, err := outputFile.WriteString(fmt.Sprintf("%s=%s", key, value))
		if err != nil {
			return
		}
		for keyMeth, valueMeth := range v.newFileMeths[key] {
			outputFile.WriteString(fmt.Sprintf(",%s=%s", keyMeth, valueMeth))
		}
		outputFile.WriteString("\n")
	}
}

func (v *versionDiff) parseAndSerializeNewContents() {

	err := filepath.Walk(util.ProgramPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(path, "_test.go") {

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			fileChecksum := checksum.Calculate(content)

			meths, err := v.extractFileInfo(path)
			if err != nil {
				return err
			}

			v.newFiles[path] = fileChecksum
			v.newFileMeths[path] = meths

		}
		return nil
	})

	if err != nil {
		fmt.Println("Error walking directory:", err)
	}

	v.serializeNewContents()

}

func (v *versionDiff) extractFileInfo(filePath string) (map[string]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	meths := make(map[string]string)

	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					typeStart := fset.Position(typeSpec.Pos()).Offset
					typeEnd := fset.Position(typeSpec.End()).Offset
					typeContent := v.readFileContent(filePath, typeStart, typeEnd)
					typeChecksum := checksum.Calculate([]byte(typeContent))
					meths[typeSpec.Name.Name] = typeChecksum
				}
			}
		}
		switch d := decl.(type) {
		case *ast.FuncDecl:
			functionName := d.Name.Name
			functionStart := fset.Position(d.Pos()).Offset
			functionEnd := fset.Position(d.End()).Offset
			functionContent := v.readFileContent(filePath, functionStart, functionEnd)
			functionChecksum := checksum.Calculate([]byte(functionContent))

			meths[functionName] = functionChecksum
		}
	}

	return meths, nil
}

// Hàm đọc nội dung của một đoạn trong file từ vị trí start đến vị trí end
func (v *versionDiff) readFileContent(filePath string, start, end int) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	fileContent := make([]byte, end-start)
	_, err = file.ReadAt(fileContent, int64(start))
	if err != nil {
		return ""
	}

	return string(fileContent)
}

//
//type Visitor struct {
//	CallExprs []*ast.CallExpr
//}
//
//func (v Visitor) Visit(node ast.Node) (w ast.Visitor) {
//	return nil
//}
