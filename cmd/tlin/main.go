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
	"sort"
	"strings"
	"time"

	"github.com/gnolang/tlin/formatter"
	"github.com/gnolang/tlin/internal"
	"github.com/gnolang/tlin/internal/analysis/cfg"
	"github.com/gnolang/tlin/internal/fixer"
	tt "github.com/gnolang/tlin/internal/types"
	"github.com/gnolang/tlin/lint"
	"github.com/go-yaml/yaml"
	"go.uber.org/zap"
)

const (
	defaultTimeout             = 5 * time.Minute
	defaultConfidenceThreshold = 0.75
)

type Config struct {
	IgnoreRules          string
	FuncName             string
	Output               string
	Paths                []string
	Timeout              time.Duration
	CyclomaticThreshold  int
	ConfidenceThreshold  float64
	CyclomaticComplexity bool
	CFGAnalysis          bool
	AutoFix              bool
	DryRun               bool
	JsonOutput           bool
	Init                 bool
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	config := parseFlags(os.Args[1:])

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	if config.Init {
		err := initConfigurationFile(config.Output)
		if err != nil {
			logger.Error("Error initializing config file", zap.Error(err))
			os.Exit(1)
		}
		return
	}

	engine, err := lint.New(".", nil)
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
			runCFGAnalysis(ctx, logger, config.Paths, config.FuncName, config.Output)
		})
	} else if config.CyclomaticComplexity {
		runWithTimeout(ctx, func() {
			runCyclomaticComplexityAnalysis(ctx, logger, config.Paths, config.CyclomaticThreshold, config.JsonOutput, config.Output)
		})
	} else if config.AutoFix {
		runWithTimeout(ctx, func() {
			runAutoFix(ctx, logger, engine, config.Paths, config.DryRun, config.ConfidenceThreshold)
		})
	} else {
		runWithTimeout(ctx, func() {
			runNormalLintProcess(ctx, logger, engine, config.Paths, config.JsonOutput, config.Output)
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
	flagSet.StringVar(&config.Output, "o", "", "Output path")
	flagSet.BoolVar(&config.DryRun, "dry-run", false, "Run in dry-run mode (show fixes without applying them)")
	flagSet.BoolVar(&config.JsonOutput, "json", false, "Output issues in JSON format")
	flagSet.Float64Var(&config.ConfidenceThreshold, "confidence", defaultConfidenceThreshold, "Confidence threshold for auto-fixing (0.0 to 1.0)")
	flagSet.BoolVar(&config.Init, "init", false, "Initialize a new linter configuration file")

	err := flagSet.Parse(args)
	if err != nil {
		fmt.Println("Error parsing flags:", err)
		os.Exit(1)
	}

	config.Paths = flagSet.Args()
	if !config.Init && len(config.Paths) == 0 {
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

func runCFGAnalysis(_ context.Context, logger *zap.Logger, paths []string, funcName string, output string) {
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
					if output != "" {
						err := cfg.RenderToGraphVizFile([]byte(buf.String()), output)
						if err != nil {
							logger.Error("Failed to render CFG to GraphViz file", zap.Error(err))
						}
					} else {
						fmt.Printf("CFG for function %s in file %s:\n%s\n", funcName, path, buf.String())
					}
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

func initConfigurationFile(configurationPath string) error {
	if configurationPath == "" {
		configurationPath = ".tlin.yaml"
	}

	// Create a yaml file with rules
	config := lint.Config{
		Name:  "tlin",
		Rules: map[string]lint.Rule{},
	}
	d, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	f, err := os.Create(configurationPath)
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Write(d)
	if err != nil {
		return err
	}

	return nil
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
