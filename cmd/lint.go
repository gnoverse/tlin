package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/gnolang/tlin/formatter"
	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
	"github.com/gnolang/tlin/lint"
)

var (
	ignoreRules    string
	ignorePaths    string
	lintJsonOutput bool
	outPath        string
)

var lintCmd = &cobra.Command{
	Use:   "lint [paths...]",
	Short: "Run the normal lint process",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("error: Please provide file or directory paths")
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		engine, err := lint.New(".", nil, cfgFile)
		if err != nil {
			logger.Fatal("Failed to initialize lint engine", zap.Error(err))
		}

		if ignoreRules != "" {
			rules := strings.Split(ignoreRules, ",")
			for _, rule := range rules {
				engine.IgnoreRule(strings.TrimSpace(rule))
			}
		}

		if ignorePaths != "" {
			paths := strings.Split(ignorePaths, ",")
			for _, path := range paths {
				engine.IgnorePath(strings.TrimSpace(path))
			}
		}

		runNormalLintProcess(ctx, logger, engine, args, lintJsonOutput, outPath)
	},
}

func init() {
	lintCmd.Flags().StringVar(&ignoreRules, "ignore", "", "Comma-separated list of lint rules to ignore")
	lintCmd.Flags().StringVar(&ignorePaths, "ignore-paths", "", "Comma-separated list of paths to ignore")
	lintCmd.Flags().BoolVar(&lintJsonOutput, "json", false, "Output issues in JSON format")
	lintCmd.Flags().StringVarP(&outPath, "output", "o", "", "Output path (when using JSON)")
}

func runNormalLintProcess(ctx context.Context, logger *zap.Logger, engine lint.LintEngine, paths []string, isJson bool, jsonOutput string) {
	issues, err := lint.ProcessFiles(ctx, logger, engine, paths, lint.ProcessFile)
	if err != nil {
		logger.Error("Error processing files", zap.Error(err))
		os.Exit(1)
	}

	printIssues(logger, issues, isJson, jsonOutput)

	if len(issues) > 0 {
		os.Exit(1)
	}
}

func printIssues(logger *zap.Logger, issues []tt.Issue, isJson bool, jsonOutput string) {
	issuesByFile := make(map[string][]tt.Issue)
	for _, issue := range issues {
		issuesByFile[issue.Filename] = append(issuesByFile[issue.Filename], issue)
	}

	sortedFiles := make([]string, 0, len(issuesByFile))
	for filename := range issuesByFile {
		sortedFiles = append(sortedFiles, filename)
	}
	sort.Strings(sortedFiles)

	if !isJson {
		// text output
		for _, filename := range sortedFiles {
			fileIssues := issuesByFile[filename]
			sourceCode, err := internal.ReadSourceCode(filename)
			if err != nil {
				logger.Error("Error reading source file", zap.String("file", filename), zap.Error(err))
				continue
			}
			output := formatter.GenerateFormattedIssue(fileIssues, sourceCode)
			fmt.Println(output)
		}
	} else {
		// JSON output
		d, err := json.Marshal(issuesByFile)
		if err != nil {
			logger.Error("Error marshalling issues to JSON", zap.Error(err))
			return
		}
		if jsonOutput == "" {
			fmt.Println(string(d))
		} else {
			f, err := os.Create(jsonOutput)
			if err != nil {
				logger.Error("Error creating JSON output file", zap.Error(err))
				return
			}
			defer f.Close()
			_, err = f.Write(d)
			if err != nil {
				logger.Error("Error writing JSON output file", zap.Error(err))
				return
			}
		}
	}
}
