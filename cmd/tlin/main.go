package main

import (
	"context"
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
	"github.com/gnoswap-labs/tlin/internal/lints"
	tt "github.com/gnoswap-labs/tlin/internal/types"
	"go.uber.org/zap"
)

const (
	defaultTimeout  = 5 * time.Minute
	defaultCacheDir = ".tlin-cache"
)

type Config struct {
	Timeout              time.Duration
	CyclomaticComplexity bool
	CyclomaticThreshold  int
	IgnoreRules          string
	Paths                []string
	CFGAnalysis          bool
	FuncName             string
	UseCache             bool
	CacheDir             string
	CacheMaxAge          time.Duration
	InvalidateCache      bool
}

type LintEngine interface {
	Run(filePath string) ([]tt.Issue, error)
	IgnoreRule(rule string)
	SetCacheOptions(useCache bool, cacheDir string, maxAge time.Duration)
	InvalidateCache() error
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	config := parseFlags()

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	engine, err := internal.NewEngine(".", config.UseCache, config.CacheDir)
	if err != nil {
		logger.Fatal("Failed to initialize lint engine", zap.Error(err))
	}

	engine.SetCacheOptions(config.UseCache, config.CacheDir, config.CacheMaxAge)

	if config.InvalidateCache {
		if err := engine.InvalidateCache(); err != nil {
			logger.Error("failed to invalidate cache", zap.Error(err))
		} else {
			logger.Info("cache invalidated successfully")
		}
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
			runCyclomaticComplexityAnalysis(ctx, logger, config.Paths, config.CyclomaticThreshold)
		})
	} else {
		runWithTimeout(ctx, func() {
			runNormalLintProcess(ctx, logger, engine, config.Paths)
		})
	}
}

func parseFlags() Config {
	config := Config{}
	flag.DurationVar(&config.Timeout, "timeout", defaultTimeout, "Set a timeout for the linter. example: 1s, 1m, 1h")
	flag.BoolVar(&config.CyclomaticComplexity, "cyclo", false, "Run cyclomatic complexity analysis")
	flag.IntVar(&config.CyclomaticThreshold, "threshold", 10, "Cyclomatic complexity threshold")
	flag.StringVar(&config.IgnoreRules, "ignore", "", "Comma-separated list of lint rules to ignore")
	flag.BoolVar(&config.CFGAnalysis, "cfg", false, "Run control flow graph analysis")
	flag.StringVar(&config.FuncName, "func", "", "Function name for CFG analysis")
	flag.BoolVar(&config.UseCache, "cache", true, "Use caching for lint results")
	flag.StringVar(&config.CacheDir, "cache-dir", defaultCacheDir, "Directory to store cache files")
	flag.DurationVar(&config.CacheMaxAge, "cache-max-age", 24*time.Hour, "Maximum age of cache entries")
	flag.BoolVar(&config.InvalidateCache, "invalidate-cache", false, "Invalidate the entire cache")

	flag.Parse()

	config.Paths = flag.Args()
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

func runNormalLintProcess(ctx context.Context, logger *zap.Logger, engine LintEngine, paths []string) {
	issues, err := processFiles(ctx, logger, engine, paths, processFile)
	if err != nil {
		logger.Error("Error processing files", zap.Error(err))
		os.Exit(1)
	}

	printIssues(logger, issues)

	if len(issues) > 0 {
		os.Exit(1)
	}
}

func runCyclomaticComplexityAnalysis(ctx context.Context, logger *zap.Logger, paths []string, threshold int) {
	issues, err := processFiles(ctx, logger, nil, paths, func(_ LintEngine, path string) ([]tt.Issue, error) {
		return processCyclomaticComplexity(path, threshold)
	})
	if err != nil {
		logger.Error("Error processing files for cyclomatic complexity", zap.Error(err))
		os.Exit(1)
	}

	printIssues(logger, issues)

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

func processPath(ctx context.Context, logger *zap.Logger, engine LintEngine, path string, processor func(LintEngine, string) ([]tt.Issue, error)) ([]tt.Issue, error) {
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

func printIssues(logger *zap.Logger, issues []tt.Issue) {
	issuesByFile := make(map[string][]tt.Issue)
	for _, issue := range issues {
		issuesByFile[issue.Filename] = append(issuesByFile[issue.Filename], issue)
	}

	var sortedFiles []string
	for filename := range issuesByFile {
		sortedFiles = append(sortedFiles, filename)
	}
	sort.Strings(sortedFiles)

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
}

func hasDesiredExtension(path string) bool {
	return filepath.Ext(path) == ".go" || filepath.Ext(path) == ".gno"
}
