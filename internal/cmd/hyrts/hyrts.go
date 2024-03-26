package hyrts

import (
	"github.com/dangdtr/go-hyrts/internal/core/rts/hybrid_rts"
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
	hybrid_rts.Run()
}
