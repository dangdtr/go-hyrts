package executor

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/dangdtr/go-hyrts/internal/core/util"
)

func ExecShell(include map[string]bool) {

	if len(include) == 0 && util.OldDir == "" {
		cmd := exec.Command(
			"go",
			"test",
			fmt.Sprint(util.ProgramPath, "/..."),
		)
		fmt.Println(cmd.String())
		output, _ := cmd.CombinedOutput()
		fmt.Println(string(output))
		return
	}
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
			"-run",
			//"-testify.m",
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
