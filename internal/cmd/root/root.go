package root

import (
	"fmt"

	"github.com/dangdtr/go-hyrts/internal/cmd/hyrts"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	debug bool
)

func init() {
	cobra.OnInitialize(func() {
		if err := viper.ReadInConfig(); err == nil && debug {
			fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
		}
	})
}

// NewCmdRoot is a root command.
func NewCmdRoot() *cobra.Command {
	cmd := cobra.Command{
		Use:     "go-hyrts <command> <subcommand>",
		Version: "go-hyrts 1.2.8",
		Short:   "Interactive go-hyrts CLI",
		Long:    "Interactive go-hyrts command line.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Turn on debug output")

	addChildCommands(&cmd)

	return &cmd
}

func addChildCommands(cmd *cobra.Command) {
	cmd.AddCommand(
		hyrts.NewCmdHyRTS(),
	)
}
