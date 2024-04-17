package executor

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"strings"

	"github.com/dangdtr/go-hyrts/internal/core/util"
)

func ExecShell(include map[string]bool) {

	for testFile := range include {
		parts := strings.Split(testFile, ":")

		args := fmt.Sprintf("%s%s", util.ProgramPath, parts[0])

		lastIndex := strings.LastIndex(args, "/")
		if lastIndex != -1 {
			args = args[:lastIndex]
		}
		cmd := exec.Command(
			"go",
			"test",
			"-v",
			args,
			//"-run",
			"-testify.m",
			fmt.Sprintf("^%s$", parts[1]),
		)

		//fmt.Println(testFile)
		//cmd := exec.Command(
		//	"go",
		//	"test",
		//	"-v",
		//	"-run",
		//	//testFile,
		//	"^(TestJoinStrings|TestGetUserInfo)$",
		//	fmt.Sprint(util.ProgramPath, "/..."),
		//)
		fmt.Println(cmd.String())
		output, _ := cmd.CombinedOutput()
		fmt.Println(string(output))

	}
}

func findTestFuncInTestFile(testPath string) []string {
	list := make([]string, 0)
	fs := token.NewFileSet()
	node, err := parser.ParseFile(fs, testPath, nil, parser.AllErrors)
	if err != nil {
		return nil
	}
	for _, decl := range node.Decls {

		switch d := decl.(type) {
		case *ast.FuncDecl:
			if strings.HasPrefix(d.Name.Name, util.TestPrefix) {
				list = append(list, d.Name.Name)
			}
		}
	}
	return list
}

func formatTestFuncRun(list []string) string {
	preFormat := strings.Join(list, "|")
	return fmt.Sprintf("^(%s)$", preFormat)
}

// go test -v -run ^TestGetListEvent$ ./...
// sudo go test -v /Users/dangdt/teko/footprint/golang/usersegmentv2/pkg/segment -testify.m TestGetListEvent
// go test -test.run ^\QTestRepoTestSuite\E$/^\QTestGetListEvent\E$ -testify.m ^TestGetListEvent$
