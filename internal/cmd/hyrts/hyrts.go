package hyrts

import (
	"os"

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
	util.OldDir = "./diff_old"
	util.NewDir = "./diff_new"

	util.ProgramPath, _ = os.Getwd()
	//util.ProgramPath = "/Users/dangdt/Documents/coding/go-hyrts/go-hyrts/example"
	include := hybrid_rts.Run()

	//include := make(map[string]bool)
	//include["example/package1/file1_test.go"] = true
	//include["TestJoinStrings"] = true

	executor.ExecShell(include)
}
