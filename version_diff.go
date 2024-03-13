package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	_ "strings"
)

type FunctionInfo struct {
	Name     string
	Checksum string
}

type FileInfo struct {
	Path      string
	Checksum  string
	Functions []FunctionInfo
}

func main2() {
	rootDir := "/Users/dangdt/Documents/coding/go-hyrts/proposal2/src"

	oldVersion, err := readFileInfos("/Users/dangdt/Documents/coding/go-hyrts/proposal2/HyRTS-checksum.txt")
	if err != nil {
		fmt.Println("Lỗi khi đọc file:", err)
		return
	}

	//for _, fileInfo := range oldVersion {
	//	fmt.Printf("Path: %s, File Checksum: %s\n", fileInfo.Path, fileInfo.Checksum)
	//	for _, function := range fileInfo.Functions {
	//		fmt.Printf("\tFunction: %s, Checksum: %s\n", function.Name, function.Checksum)
	//	}
	//}

	outputFile, err := os.Create("HyRTS-checksum.txt")
	if err != nil {
		fmt.Println("Lỗi khi tạo file:", err)
		return
	}
	defer outputFile.Close()

	var newVersion []FileInfo

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go") {
			// Đọc nội dung của file
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			fileChecksum := calculateChecksum(content)

			functions, err := extractFunctionsInfo(path)
			if err != nil {
				return err
			}

			outputFile.WriteString(fmt.Sprintf("%s=%s", path, fileChecksum))
			for _, function := range functions {
				outputFile.WriteString(fmt.Sprintf(",%s=%s", function.Name, function.Checksum))
			}
			outputFile.WriteString("\n")

			fileInfo := FileInfo{
				Path:      path,
				Checksum:  fileChecksum,
				Functions: functions,
			}
			newVersion = append(newVersion, fileInfo)

			//fmt.Printf("%s=%s\n", fileInfo.Path, fileInfo.Checksum)
			//for _, function := range fileInfo.Functions {
			//	fmt.Printf("\t%s=%s\n", function.Name, function.Checksum)
			//}
		}
		return nil
	})

	// So sánh dữ liệu mới với dữ liệu cũ từ tệp
	addedFiles, changedFiles, deletedFiles := diff(oldVersion, newVersion)

	outputFileDiff, err := os.Create("HyRTS-diff.txt")
	if err != nil {
		fmt.Println("Lỗi khi tạo file:", err)
		return
	}
	defer outputFile.Close()

	fmt.Println("add: ")
	for _, file := range addedFiles {
		fmt.Println(file.Path, file.Functions)
		outputFileDiff.WriteString(fmt.Sprintf("%s%v\n", file.Path, file.Functions))

	}

	fmt.Println("\nchanged: ")
	for _, file := range changedFiles {
		fmt.Println(file.Path, file.Functions)
		outputFileDiff.WriteString(fmt.Sprintf("%s%v\n", file.Path, file.Functions))

	}

	fmt.Println("\ndelete: ")
	for _, file := range deletedFiles {
		fmt.Println(file.Path, file.Functions)
		outputFileDiff.WriteString(fmt.Sprintf("%s%v\n", file.Path, file.Functions))

	}

	if err != nil {
		fmt.Println("Lỗi:", err)
		return
	}

}

// Hàm tính checksum của một đoạn dữ liệu
func calculateChecksum(data []byte) string {
	hasher := md5.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

// trích xuất thông tin về các hàm từ một file Go
func extractFunctionsInfo(filePath string) ([]FunctionInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var functions []FunctionInfo

	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			functionName := d.Name.Name
			functionStart := fset.Position(d.Pos()).Offset
			functionEnd := fset.Position(d.End()).Offset
			functionContent := readFileContent(filePath, functionStart, functionEnd)
			functionChecksum := calculateChecksum([]byte(functionContent))
			functionInfo := FunctionInfo{
				Name:     functionName,
				Checksum: functionChecksum,
			}
			functions = append(functions, functionInfo)
		}
	}

	return functions, nil
}

// Hàm đọc nội dung của một đoạn trong file từ vị trí start đến vị trí end
func readFileContent(filePath string, start, end int) string {
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

// Hàm để đọc thông tin từ file và lưu vào cấu trúc đã có
func readFileInfos(filePath string) ([]FileInfo, error) {
	var fileInfos []FileInfo

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		fileInfo := FileInfo{}

		fileChecksumParts := strings.Split(parts[0], "=")
		if len(fileChecksumParts) != 2 {
			continue
		}
		fileInfo.Path = fileChecksumParts[0]
		fileInfo.Checksum = fileChecksumParts[1]

		for _, part := range parts[1:] {
			funcInfo := FunctionInfo{}
			funcChecksumParts := strings.Split(part, "=")
			if len(funcChecksumParts) != 2 {
				continue
			}
			funcInfo.Name = funcChecksumParts[0]
			funcInfo.Checksum = funcChecksumParts[1]
			fileInfo.Functions = append(fileInfo.Functions, funcInfo)
		}

		fileInfos = append(fileInfos, fileInfo)
	}

	return fileInfos, nil
}

func diff(existingData, newData []FileInfo) ([]FileInfo, []FileInfo, []FileInfo) {
	existingMap := convertToMap(existingData)
	newMap := convertToMap(newData)

	var addedFuncs []FileInfo
	var changedFuncs []FileInfo
	var deletedFuncs []FileInfo

	// Check for added and changed functions
	for path, newFuncs := range newMap {
		existingFuncs, exists := existingMap[path]
		if !exists {
			// If file is not found in existing data, all functions in new file are considered added
			for name, checksum := range newFuncs {
				addedFuncs = append(addedFuncs, FileInfo{Path: path, Checksum: checksum, Functions: []FunctionInfo{{Name: name, Checksum: checksum}}})
			}
			continue
		}

		changedFile := FileInfo{Path: path}

		for name, newChecksum := range newFuncs {
			existingChecksum, exists := existingFuncs[name]
			if !exists {
				// If function is not found in existing file, it's considered added
				addedFuncs = append(addedFuncs, FileInfo{Path: path, Checksum: newChecksum, Functions: []FunctionInfo{{Name: name, Checksum: newChecksum}}})
			} else if newChecksum != existingChecksum {
				changedFuncs = append(changedFuncs, FileInfo{Path: path, Checksum: newChecksum, Functions: []FunctionInfo{{Name: name, Checksum: newChecksum}}})
			}

			// Remove processed functions from existingFuncs
			delete(existingFuncs, name)
		}

		// Functions left in existingFuncs are considered deleted
		for name, deletedChecksum := range existingFuncs {
			deletedFuncs = append(deletedFuncs, FileInfo{Path: path, Checksum: deletedChecksum, Functions: []FunctionInfo{{Name: name, Checksum: deletedChecksum}}})
		}

		if len(changedFile.Functions) > 0 {
			changedFuncs = append(changedFuncs, changedFile)
		}
	}

	// Check for deleted files
	for path := range existingMap {
		_, exists := newMap[path]
		if !exists {
			for name, checksum := range existingMap[path] {
				deletedFuncs = append(deletedFuncs, FileInfo{Path: path, Checksum: checksum, Functions: []FunctionInfo{{Name: name, Checksum: checksum}}})
			}
		}
	}

	return mergeFunctionInfos(addedFuncs), mergeFunctionInfos(changedFuncs), mergeFunctionInfos(deletedFuncs)
}

func mergeFunctionInfos(files []FileInfo) []FileInfo {
	mergedFiles := make([]FileInfo, 0)

	mergedFuncMap := make(map[string][]FunctionInfo)

	for _, file := range files {
		if funcs, ok := mergedFuncMap[file.Path]; ok {
			mergedFuncMap[file.Path] = append(funcs, file.Functions...)
		} else {
			mergedFuncMap[file.Path] = file.Functions
		}
	}

	for path, funcs := range mergedFuncMap {
		mergedFiles = append(mergedFiles, FileInfo{
			Path:      path,
			Functions: funcs,
		})
	}

	return mergedFiles
}

func convertToMap(data []FileInfo) map[string]map[string]string {
	result := make(map[string]map[string]string)
	for _, file := range data {
		funcMap := make(map[string]string)
		for _, f := range file.Functions {
			funcMap[f.Name] = f.Checksum
		}
		result[file.Path] = funcMap
	}
	return result
}
