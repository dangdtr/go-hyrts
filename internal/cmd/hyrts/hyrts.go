package hyrts

import (
	"fmt"

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

	cmd.Flags().String("run", "", "Run Go HyRTS")

	return &cmd
}

func initialize(cmd *cobra.Command, _ []string) {
	fmt.Println("Running go-hyrts")
}
