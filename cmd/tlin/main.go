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
)

func main() {
	// verbose := flag.Bool("verbose", false, "Enable verbose output")
	// formatJSON := flag.Bool("json", false, "Output results in JSON format")
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

	var allIssues []lints.Issue
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

	issuesByFile := make(map[string][]lints.Issue)
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

func processFile(engine *internal.Engine, filePath string) ([]lints.Issue, error) {
	issues, err := engine.Run(filePath)
	if err != nil {
		return nil, fmt.Errorf("error linting %s: %w", filePath, err)
	}
	return issues, nil
}
