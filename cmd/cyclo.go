package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/gnolang/tlin/lint"
)

// cyclo command flags
var (
	threshold       int
	cycloJsonOutput bool
	outputPath      string
)

var cycloCmd = &cobra.Command{
	Use:   "cyclo [paths...]",
	Short: "Run cyclomatic complexity analysis",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("error: Please provide file or directory paths")
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		runCyclomaticComplexityAnalysis(ctx, logger, args, threshold, cycloJsonOutput, outputPath)
	},
}

func init() {
	cycloCmd.Flags().IntVar(&threshold, "threshold", 10, "Cyclomatic complexity threshold")
	cycloCmd.Flags().BoolVar(&cycloJsonOutput, "json", false, "Output issues in JSON format")
	cycloCmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path (when using JSON)")
}

func runCyclomaticComplexityAnalysis(ctx context.Context, logger *zap.Logger, paths []string, threshold int, isJson bool, jsonOutput string) {
	issues, err := lint.ProcessFiles(ctx, logger, nil, paths, func(_ lint.LintEngine, path string) ([]tt.Issue, error) {
		return lint.ProcessCyclomaticComplexity(path, threshold)
	})
	if err != nil {
		logger.Error("Error processing files for cyclomatic complexity", zap.Error(err))
		os.Exit(1)
	}

	printIssues(logger, issues, isJson, jsonOutput)

	if len(issues) > 0 {
		os.Exit(1)
	}
}
