package cmd

import (
	"os"
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
	Use:   "tlin",
	Short: "tlin is a linter for Gno code",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			return err
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if logger != nil {
			logger.Sync()
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// global flags for the root command
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", ".tlin.yaml", "Path to the linter configuration file")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 5*time.Minute, "Set a timeout for the linter")

	// register subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(cfgCmd)
	rootCmd.AddCommand(cycloCmd)
	rootCmd.AddCommand(fixCmd)
	rootCmd.AddCommand(lintCmd)
}
