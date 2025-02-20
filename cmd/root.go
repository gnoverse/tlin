package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	cfgFile string
	timeout time.Duration

	logger *zap.Logger
)

var rootCmd = &cobra.Command{
	Use:              "tlin [paths...]",
	Short:            "tlin - a powerful linting tool with multiple subcommands",
	TraverseChildren: true, // Prioritize subcommands
	Run: func(cmd *cobra.Command, args []string) {
		// no subcommand
		if len(args) == 0 {
			// display help when only 'tlin' is entered
			_ = cmd.Help()
			return
		}
		// Format: tlin [path1 path2 ...] => behaves like the lint subcommand
		lintCmd.Run(lintCmd, args)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(fixCmd)
	rootCmd.AddCommand(cfgCmd)
	rootCmd.AddCommand(cycloCmd)
}
