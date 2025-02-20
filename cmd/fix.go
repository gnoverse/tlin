package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/gnolang/tlin/internal/fixer"
	"github.com/gnolang/tlin/lint"
)

var (
	dryRun              bool
	confidenceThreshold float64
)

var fixCmd = &cobra.Command{
	Use:   "fix [paths...]",
	Short: "Automatically fix issues",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("error: Please provide file or directory paths")
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// initialize the lint engine
		engine, err := lint.New(".", nil, cfgFile)
		if err != nil {
			logger.Fatal("Failed to initialize lint engine", zap.Error(err))
		}

		runAutoFix(ctx, logger, engine, args, dryRun, confidenceThreshold)
	},
}

func init() {
	fixCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run in dry-run mode (show fixes without applying them)")
	fixCmd.Flags().Float64Var(&confidenceThreshold, "confidence", 0.75, "Confidence threshold for auto-fixing (0.0 to 1.0)")
}

func runAutoFix(ctx context.Context, logger *zap.Logger, engine lint.LintEngine, paths []string, dryRun bool, confidenceThreshold float64) {
	fix := fixer.New(dryRun, confidenceThreshold)

	for _, path := range paths {
		issues, err := lint.ProcessPath(ctx, logger, engine, path, lint.ProcessFile)
		if err != nil {
			logger.Error("error processing path", zap.String("path", path), zap.Error(err))
			continue
		}

		err = fix.Fix(path, issues)
		if err != nil {
			logger.Error("error fixing issues", zap.String("path", path), zap.Error(err))
		}
	}
}
