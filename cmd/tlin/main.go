package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gnoswap-labs/lint/formatter"
	"github.com/gnoswap-labs/lint/internal"
	"github.com/gnoswap-labs/lint/internal/lints"
	tt "github.com/gnoswap-labs/lint/internal/types"
	"go.uber.org/zap"
)

const defaultTimeout = 5 * time.Minute

type Config struct {
	Timeout              time.Duration
	CyclomaticComplexity bool
	CyclomaticThreshold  int
	IgnoreRules          string
	Paths                []string
}

type LintEngine interface {
	Run(filePath string) ([]tt.Issue, error)
	IgnoreRule(rule string)
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	config := parseFlags()

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

	if config.CyclomaticComplexity {
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

func processFiles(ctx context.Context, logger *zap.Logger, engine LintEngine, paths []string, processor func(LintEngine, string) ([]tt.Issue, error)) ([]tt.Issue, error) {
	var allIssues []tt.Issue
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			issues, err := processPath(ctx, logger, engine, p, processor)
			if err != nil {
				logger.Error("Error processing path", zap.String("path", p), zap.Error(err))
				return
			}
			mutex.Lock()
			allIssues = append(allIssues, issues...)
			mutex.Unlock()
		}(path)
	}

	wg.Wait()

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
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					fileIssues, err := processor(engine, filePath)
					if err != nil {
						logger.Error("Error processing file", zap.String("file", filePath), zap.Error(err))
					} else {
						issues = append(issues, fileIssues...)
					}
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
		output := formatter.FormatIssuesWithArrows(fileIssues, sourceCode)
		fmt.Println(output)
	}
}

func hasDesiredExtension(path string) bool {
	return filepath.Ext(path) == ".go" || filepath.Ext(path) == ".gno"
}
