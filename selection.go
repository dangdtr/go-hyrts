package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TestDependency struct {
	TestName   string
	Dependency string
}

func main() {
	filePath := "HyRTS-diff.txt"
	fileInfos, err := readFileInfoFromFile(filePath)
	if err != nil {
		fmt.Println("Lỗi khi đọc file:", err)
		return
	}

	// meth-cov
	methCovDir := "./meth-cov"

	// Đọc các file từ thư mục meth-cov
	files, err := getTestDependencyFiles(methCovDir)
	if err != nil {
		fmt.Println("Lỗi khi đọc thư mục meth-cov:", err)
		return
	}

	// Đọc và phân tích các phụ thuộc của test từ các file
	testDependencies := make([]TestDependency, 0)
	for _, file := range files {
		fileTestDependencies, err := readTestDependenciesFromFile(file)
		if err != nil {
			fmt.Printf("Lỗi khi đọc file %s: %v\n", file, err)
			continue
		}
		testDependencies = append(testDependencies, fileTestDependencies...)
	}

	// Tìm và in ra test chứa các hàm từ HyRTS-diff.txt
	for _, functionInfo := range fileInfos {
		testName := findTestForFunction(functionInfo, testDependencies)
		if testName != "" {
			fmt.Printf("Func %s nằm trong test %s\n", functionInfo.Functions, testName)
		}
	}

}

func readFileInfoFromFile(filePath string) ([]FileInfo, error) {
	var fileInfos []FileInfo

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fileInfo, err := parseFileInfoLine(line)
		if err != nil {
			return nil, err
		}
		fileInfos = append(fileInfos, fileInfo)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return fileInfos, nil
}

func parseFileInfoLine(line string) (FileInfo, error) {
	var fileInfo FileInfo

	// Tách đường dẫn và các hàm từ dòng
	parts := strings.Split(line, "[")
	fileInfo.Path = parts[0]
	funcStr := strings.TrimSuffix(parts[1], "]")

	// Tách tên hàm và checksum từ phần thông tin của hàm
	funcParts := strings.Split(funcStr, " ")
	for i := 0; i < len(funcParts); i += 2 {
		function := FunctionInfo{
			Name:     strings.TrimPrefix(funcParts[i], "{"),
			Checksum: funcParts[i+1],
		}
		fileInfo.Functions = append(fileInfo.Functions, function)
	}
	//fmt.Println("===", fileInfo.Functions[0].Name)
	return fileInfo, nil
}

//test-cov

func findTestForFunction(functionInfo FileInfo, testDependencies []TestDependency) string {
	//var testList []string
	for _, dependency := range testDependencies {
		//fmt.Println(dependency.Dependency)
		for _, funcInfo := range functionInfo.Functions {
			//fmt.Println(functionInfo.Functions)
			//fmt.Println(funcInfo)

			if strings.Contains(dependency.Dependency, funcInfo.Name) {
				//testList = append(testList, dependency.TestName)
				return dependency.TestName
			}
		}

	}

	//fmt.Println(testList)
	return ""
}

func getTestDependencyFiles(dirPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func readTestDependenciesFromFile(filePath string) ([]TestDependency, error) {
	var testDependencies []TestDependency

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var testName string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "-Test") {
			testName = line
		} else {
			testDependencies = append(testDependencies, TestDependency{TestName: testName, Dependency: line})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return testDependencies, nil
}
