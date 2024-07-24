package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gnoswap-labs/lint/formatter"
	"github.com/gnoswap-labs/lint/internal"
	"github.com/gnoswap-labs/lint/internal/lints"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

func main() {
	// verbose := flag.Bool("verbose", false, "Enable verbose output")
	// formatJSON := flag.Bool("json", false, "Output results in JSON format")
	cyclomaticComplexity := flag.Bool("cyclo", false, "Run cyclomatic complexity analysis")
	// |Cyclomatic Complexity | Risk Evaluation |
	// |----------------------|-----------------|
	// | 1-10                 | Low             |
	// | 11-20                | Moderate        |
	// | 21-50                | High            |
	// | 51+                  | Very High       |
	//
	// [*] MaCabe's article recommends 10 or less, but up to 15 is acceptable (by Microsoft).
	// [*] https://learn.microsoft.com/en-us/visualstudio/code-quality/code-metrics-cyclomatic-complexity?view=vs-2022
	cyclomaticThreshold := flag.Int("threshold", 10, "Cyclomatic complexity threshold")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("error: Please provide file or directory paths")
		os.Exit(1)
	}

	rootDir := "."
	engine, err := internal.NewEngine(rootDir)
	if err != nil {
		fmt.Printf("error initializing lint engine: %v\n", err)
		os.Exit(1)
	}

	if *cyclomaticComplexity {
		runCyclomaticComplexityAnalysis(args, *cyclomaticThreshold)
	} else {
		runNormalLintProcess(engine, args)
	}
}

func runNormalLintProcess(engine *internal.Engine, args []string) {
	var allIssues []tt.Issue
	for _, path := range args {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("error accessing %s: %v\n", path, err)
			continue
		}

		if info.IsDir() {
			err = filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !fileInfo.IsDir() && filepath.Ext(filePath) == ".go" || filepath.Ext(filePath) == ".gno" {
					issues, err := processFile(engine, filePath)
					if err != nil {
						fmt.Printf("error processing %s: %v\n", filePath, err)
					} else {
						allIssues = append(allIssues, issues...)
					}
				}
				return nil
			})
			if err != nil {
				fmt.Printf("error walking directory %s: %v\n", path, err)
			}
		} else {
			if filepath.Ext(path) == ".go" || filepath.Ext(path) == ".gno" {
				issues, err := processFile(engine, path)
				if err != nil {
					fmt.Printf("error processing %s: %v\n", path, err)
				} else {
					allIssues = append(allIssues, issues...)
				}
			} else {
				fmt.Printf("skipping non-.co file: %s\n", path)
			}
		}
	}

	issuesByFile := make(map[string][]tt.Issue)
	for _, issue := range allIssues {
		issuesByFile[issue.Filename] = append(issuesByFile[issue.Filename], issue)
	}

	var sortedFiles []string
	for filename := range issuesByFile {
		sortedFiles = append(sortedFiles, filename)
	}
	sort.Strings(sortedFiles)

	for _, filename := range sortedFiles {
		issues := issuesByFile[filename]
		sourceCode, err := internal.ReadSourceCode(filename)
		if err != nil {
			fmt.Printf("error reading source file %s: %v\n", filename, err)
			continue
		}
		output := formatter.FormatIssuesWithArrows(issues, sourceCode)
		fmt.Println(output)
	}

	if len(allIssues) > 0 {
		os.Exit(1)
	}
}

func runCyclomaticComplexityAnalysis(paths []string, threshold int) {
    var allIssues []tt.Issue
    for _, path := range paths {
        issues, err := processCyclomaticComplexity(path, threshold)
        if err != nil {
            fmt.Printf("error processing %s: %v\n", path, err)
        } else {
            allIssues = append(allIssues, issues...)
        }
    }

    for _, issue := range allIssues {
        fmt.Printf("%s: %s\n", issue.Filename, issue.Message)
    }

    if len(allIssues) > 0 {
        os.Exit(1)
    }
}

func processCyclomaticComplexity(path string, threshold int) ([]tt.Issue, error) {
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
				fileIssue, err := lints.DetectHighCyclomaticComplexity(filePath, threshold)
				if err != nil {
					fmt.Printf("error processing %s: %v\n", filePath, err)
				} else {
					issues = append(issues, fileIssue...)
				}
			}
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", path, err)
		}
	} else if hasDesiredExtension(path) {
		fileIssue, err := lints.DetectHighCyclomaticComplexity(path, threshold)
		if err != nil {
			return nil, err
		}
		issues = append(issues, fileIssue...)
	}

	return issues, nil
}

func processFile(engine *internal.Engine, filePath string) ([]tt.Issue, error) {
	issues, err := engine.Run(filePath)
	if err != nil {
		return nil, fmt.Errorf("error linting %s: %w", filePath, err)
	}
	return issues, nil
}

func hasDesiredExtension(path string) bool {
	return filepath.Ext(path) == ".go" || filepath.Ext(path) == ".gno"
}