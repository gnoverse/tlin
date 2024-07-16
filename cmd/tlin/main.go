package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	lint "github.com/gnoswap-labs/lint/internal"
)

func main() {
	// verbose := flag.Bool("verbose", false, "Enable verbose output")
	// formatJSON := flag.Bool("json", false, "Output results in JSON format")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Error: Please provide file or directory paths")
		os.Exit(1)
	}

	engine := lint.NewEngine()
	var allIssues []lint.Issue

	for _, path := range args {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("Error accessing %s: %v\n", path, err)
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
						fmt.Printf("Error processing %s: %v\n", filePath, err)
					} else {
						allIssues = append(allIssues, issues...)
					}
				}
				return nil
			})
			if err != nil {
				fmt.Printf("Error walking directory %s: %v\n", path, err)
			}
		} else {
			if filepath.Ext(path) == ".go" || filepath.Ext(path) == ".gno" {
				issues, err := processFile(engine, path)
				if err != nil {
					fmt.Printf("Error processing %s: %v\n", path, err)
				} else {
					allIssues = append(allIssues, issues...)
				}
			} else {
				fmt.Printf("Skipping non-.co file: %s\n", path)
			}
		}
	}

	issuesByFile := make(map[string][]lint.Issue)
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
		sourceCode, err := lint.ReadSourceCode(filename)
		if err != nil {
			fmt.Printf("error reading source file %s: %v\n", filename, err)
			continue
		}
		output := lint.FormatIssuesWithArrows(issues, sourceCode)
		fmt.Println(output)
	}

	if len(allIssues) > 0 {
		os.Exit(1)
	}
}

func processFile(engine *lint.Engine, filePath string) ([]lint.Issue, error) {
	issues, err := engine.Run(filePath)
	if err != nil {
		return nil, fmt.Errorf("error linting %s: %w", filePath, err)
	}
	return issues, nil
}
