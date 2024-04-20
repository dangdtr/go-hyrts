package hyrts

import (
	"fmt"
	"go/build"
	"os"
	"strings"

	"github.com/dangdtr/go-hyrts/internal/core/executor"
	"github.com/dangdtr/go-hyrts/internal/core/rts/hybrid_rts"
	"github.com/dangdtr/go-hyrts/internal/core/util"
	"github.com/spf13/cobra"
)

func NewCmdHyRTS() *cobra.Command {
	cmd := cobra.Command{
		Use:   "hyrts",
		Short: "HyRTS in go",
		Long:  "Hybrid Regression Test Selection go application",
		//Aliases: []string{"initialize", "configure", "config", "setup"},
		Run: initialize,
	}

	cmd.Flags().SortFlags = false

	return &cmd
}

func initialize(cmd *cobra.Command, _ []string) {
	fmt.Println("[go-hyrts] is running...")
	util.OldDir = "./diff_old"
	util.NewDir = "./diff_new"

	util.ProgramPath, _ = os.Getwd()
	include := hybrid_rts.Run()

	executor.ExecShell(include)

}

func isStandardPackage(path string) bool {
	p, _ := build.Import(path, "", build.FindOnly)
	return p.Goroot && p.ImportPath != "" && !strings.Contains(p.ImportPath, ".")
}
