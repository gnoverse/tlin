package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gnoswap-labs/tlin/formatter"
	"github.com/gnoswap-labs/tlin/internal"
	"github.com/gnoswap-labs/tlin/internal/analysis/cfg"
	"github.com/gnoswap-labs/tlin/internal/fixer"
	"github.com/gnoswap-labs/tlin/internal/lints"
	tt "github.com/gnoswap-labs/tlin/internal/types"
	"go.uber.org/zap"
)

const (
	defaultTimeout             = 5 * time.Minute
	defaultConfidenceThreshold = 0.75
)

type Config struct {
	Timeout              time.Duration
	CyclomaticComplexity bool
	CyclomaticThreshold  int
	IgnoreRules          string
	Paths                []string
	CFGAnalysis          bool
	FuncName             string
	AutoFix              bool
	DryRun               bool
	JsonOutput           string
	ConfidenceThreshold  float64
}

type LintEngine interface {
	Run(filePath string) ([]tt.Issue, error)
	IgnoreRule(rule string)
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	config := parseFlags(os.Args[1:])

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	engine, err := internal.NewEngine(".")
	if err != nil {
		logger.Fatal("Failed to initialize lint engine", zap.Error(err))
	}

	if config.IgnoreRules != "" {
		rules := strings.Split(config.IgnoreRules, ",")
		for _, rule := range rules {
			engine.IgnoreRule(strings.TrimSpace(rule))
		}
	}

	if config.CFGAnalysis {
		runWithTimeout(ctx, func() {
			runCFGAnalysis(ctx, logger, config.Paths, config.FuncName)
		})
	} else if config.CyclomaticComplexity {
		runWithTimeout(ctx, func() {
			runCyclomaticComplexityAnalysis(ctx, logger, config.Paths, config.CyclomaticThreshold, config.JsonOutput)
		})
	} else if config.AutoFix {
		runWithTimeout(ctx, func() {
			runAutoFix(ctx, logger, engine, config.Paths, config.DryRun, config.ConfidenceThreshold)
		})
	} else {
		runWithTimeout(ctx, func() {
			runNormalLintProcess(ctx, logger, engine, config.Paths, config.JsonOutput)
		})
	}
}

func parseFlags(args []string) Config {
	flagSet := flag.NewFlagSet("tlin", flag.ExitOnError)
	config := Config{}

	flagSet.DurationVar(&config.Timeout, "timeout", defaultTimeout, "Set a timeout for the linter. example: 1s, 1m, 1h")
	flagSet.BoolVar(&config.CyclomaticComplexity, "cyclo", false, "Run cyclomatic complexity analysis")
	flagSet.IntVar(&config.CyclomaticThreshold, "threshold", 10, "Cyclomatic complexity threshold")
	flagSet.StringVar(&config.IgnoreRules, "ignore", "", "Comma-separated list of lint rules to ignore")
	flagSet.BoolVar(&config.CFGAnalysis, "cfg", false, "Run control flow graph analysis")
	flagSet.StringVar(&config.FuncName, "func", "", "Function name for CFG analysis")
	flagSet.BoolVar(&config.AutoFix, "fix", false, "Automatically fix issues")
	flagSet.BoolVar(&config.DryRun, "dry-run", false, "Run in dry-run mode (show fixes without applying them)")
	flagSet.StringVar(&config.JsonOutput, "json-output", "", "Output issues in JSON format to the specified file")
	flagSet.Float64Var(&config.ConfidenceThreshold, "confidence", defaultConfidenceThreshold, "Confidence threshold for auto-fixing (0.0 to 1.0)")

	err := flagSet.Parse(args)
	if err != nil {
		fmt.Println("Error parsing flags:", err)
		os.Exit(1)
	}

	config.Paths = flagSet.Args()
	if len(config.Paths) == 0 {
		fmt.Println("error: Please provide file or directory paths")
		os.Exit(1)
	}

	return config
}

func runWithTimeout(ctx context.Context, f func()) {
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Linter timed out")
		os.Exit(1)
	case <-done:
		return
	}
}

func runNormalLintProcess(ctx context.Context, logger *zap.Logger, engine LintEngine, paths []string, jsonOutput string) {
	issues, err := processFiles(ctx, logger, engine, paths, processFile)
	if err != nil {
		logger.Error("Error processing files", zap.Error(err))
		os.Exit(1)
	}

	printIssues(logger, issues, jsonOutput)

	if len(issues) > 0 {
		os.Exit(1)
	}
}

func runCyclomaticComplexityAnalysis(ctx context.Context, logger *zap.Logger, paths []string, threshold int, jsonOutput string) {
	issues, err := processFiles(ctx, logger, nil, paths, func(_ LintEngine, path string) ([]tt.Issue, error) {
		return processCyclomaticComplexity(path, threshold)
	})
	if err != nil {
		logger.Error("Error processing files for cyclomatic complexity", zap.Error(err))
		os.Exit(1)
	}

	printIssues(logger, issues, jsonOutput)

	if len(issues) > 0 {
		os.Exit(1)
	}
}

func runCFGAnalysis(_ context.Context, logger *zap.Logger, paths []string, funcName string) {
	functionFound := false
	for _, path := range paths {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			logger.Error("Failed to parse file", zap.String("path", path), zap.Error(err))
			continue
		}

		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				if fn.Name.Name == funcName {
					cfgGraph := cfg.FromFunc(fn)
					var buf strings.Builder
					cfgGraph.PrintDot(&buf, fset, func(n ast.Stmt) string { return "" })
					fmt.Printf("CFG for function %s in file %s:\n%s\n", funcName, path, buf.String())
					functionFound = true
					return
				}
			}
		}
	}

	if !functionFound {
		fmt.Printf("Function not found: %s\n", funcName)
	}
}

func runAutoFix(ctx context.Context, logger *zap.Logger, engine LintEngine, paths []string, dryRun bool, confidenceThreshold float64) {
	fix := fixer.New(dryRun, confidenceThreshold)

	for _, path := range paths {
		issues, err := processPath(ctx, logger, engine, path, processFile)
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

func processFiles(ctx context.Context, logger *zap.Logger, engine LintEngine, paths []string, processor func(LintEngine, string) ([]tt.Issue, error)) ([]tt.Issue, error) {
	var allIssues []tt.Issue
	for _, path := range paths {
		issues, err := processPath(ctx, logger, engine, path, processor)
		if err != nil {
			logger.Error("Error processing path", zap.String("path", path), zap.Error(err))
			return nil, err
		}
		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

func processPath(_ context.Context, logger *zap.Logger, engine LintEngine, path string, processor func(LintEngine, string) ([]tt.Issue, error)) ([]tt.Issue, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("error accessing %s: %w", path, err)
	}

	var issues []tt.Issue
	if info.IsDir() {
		err = filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fileInfo.IsDir() && hasDesiredExtension(filePath) {
				fileIssues, err := processor(engine, filePath)
				if err != nil {
					logger.Error("Error processing file", zap.String("file", filePath), zap.Error(err))
				} else {
					issues = append(issues, fileIssues...)
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", path, err)
		}
	} else if hasDesiredExtension(path) {
		fileIssues, err := processor(engine, path)
		if err != nil {
			return nil, err
		}
		issues = append(issues, fileIssues...)
	}

	return issues, nil
}

func processFile(engine LintEngine, filePath string) ([]tt.Issue, error) {
	return engine.Run(filePath)
}

func processCyclomaticComplexity(path string, threshold int) ([]tt.Issue, error) {
	return lints.DetectHighCyclomaticComplexity(path, threshold)
}

func printIssues(logger *zap.Logger, issues []tt.Issue, jsonOutput string) {
	issuesByFile := make(map[string][]tt.Issue)
	for _, issue := range issues {
		issuesByFile[issue.Filename] = append(issuesByFile[issue.Filename], issue)
	}

	var sortedFiles []string
	for filename := range issuesByFile {
		sortedFiles = append(sortedFiles, filename)
	}
	sort.Strings(sortedFiles)

	if jsonOutput == "" {
		for _, filename := range sortedFiles {
			fileIssues := issuesByFile[filename]
			sourceCode, err := internal.ReadSourceCode(filename)
			if err != nil {
				logger.Error("Error reading source file", zap.String("file", filename), zap.Error(err))
				continue
			}
			output := formatter.GenetateFormattedIssue(fileIssues, sourceCode)
			fmt.Println(output)
		}
	} else {
		issuesEssentialsByFile := make(map[string][]tt.IssueEssential)
		for _, filename := range sortedFiles {
			fileIssues := issuesByFile[filename]
			sourceCode, err := internal.ReadSourceCode(filename)
			if err != nil {
				logger.Error("Error reading source file", zap.String("file", filename), zap.Error(err))
				continue
			}

			for _, issue := range fileIssues {
				codeSnippet := formatter.GetCodeSnippet(issue, sourceCode)
				issueEssential := tt.IssueEssential{
					Rule:       issue.Rule,
					Category:   issue.Category,
					Message:    issue.Message,
					Suggestion: issue.Suggestion,
					Note:       issue.Note,
					Snippet:    codeSnippet,
					Confidence: issue.Confidence,
				}
				issuesEssentialsByFile[filename] = append(issuesEssentialsByFile[filename], issueEssential)
			}
		}

		d, err := json.Marshal(issuesEssentialsByFile)
		if err != nil {
			logger.Error("Error marshalling issues to JSON", zap.Error(err))
			return
		}
		f, _ := os.Create(jsonOutput)
		defer f.Close()
		f.Write(d)
	}
}

func hasDesiredExtension(path string) bool {
	return filepath.Ext(path) == ".go" || filepath.Ext(path) == ".gno"
}
