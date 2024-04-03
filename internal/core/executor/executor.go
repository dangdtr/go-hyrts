package executor

import (
	"fmt"
	"os/exec"

	"github.com/dangdtr/go-hyrts/internal/core/util"
)

func ExecShell(include map[string]bool) {

	for testFile := range include {
		//args := fmt.Sprintf("%s/%s", util.ProgramPath, testFile)
		//
		//lastIndex := strings.LastIndex(args, "/")
		//if lastIndex != -1 {
		//	args = args[:lastIndex]
		//}
		//cmd := exec.Command(
		//	"go",
		//	"test",
		//	"-v",
		//	"-run",
		//	"Test.*",
		//	args,
		//)

		cmd := exec.Command(
			"go",
			"test",
			"-v",
			"-run",
			testFile,
			//"^(TestJoinStrings|TestGetUserInfo)$",
			fmt.Sprint(util.ProgramPath, "/..."),
		)
		fmt.Println(cmd.String())
		output, _ := cmd.CombinedOutput()
		fmt.Println(string(output))

	}

}
