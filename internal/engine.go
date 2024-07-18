package internal

import (
	"encoding/json"
	"fmt"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Engine manages the linting process.
type Engine struct {
	SymbolTable *SymbolTable
	rules       []LintRule
}

// NewEngine creates a new lint engine.
func NewEngine(rootDir string) (*Engine, error) {
	st, err := BuildSymbolTable(rootDir)
	if err != nil {
		return nil, fmt.Errorf("error building symbol table: %w", err)
	}

	engine := &Engine{SymbolTable: st}
	engine.registerDefaultRules()

	return engine, nil
}

// registerDefaultRules adds the default set of lint rules to the engine.
func (e *Engine) registerDefaultRules() {
	e.rules = append(e.rules,
		&GolangciLintRule{},
		&UnnecessaryElseRule{},
		&UnusedFunctionRule{},
	)
}

// AddRule allows adding custom lint rules to the engine.
func (e *Engine) AddRule(rule LintRule) {
	e.rules = append(e.rules, rule)
}

// Run applies golangci-lint to the given file and returns a slice of issues.
func (e *Engine) Run(filename string) ([]Issue, error) {
	var tempFile string
	var err error

	if strings.HasSuffix(filename, ".gno") {
		tempFile, err = createTempGoFile(filename)
		if err != nil {
			return nil, fmt.Errorf("error creating temp file: %w", err)
		}
		// deferring the removal of the temporary file to the end of the function
		// to ensure that the golangci-lint analyze the file before it's removed.
		defer func() {
			if tempFile != "" {
				_ = os.Remove(tempFile)
			}
		}()
	} else {
		tempFile = filename
	}

	// issues, err := runGolangciLint(tempFile)
	// if err != nil {
	// 	return nil, fmt.Errorf("error running golangci-lint: %w", err)
	// }
	// filtered := e.filterUndefinedIssues(issues)

	// unnecessaryElseIssues, err := e.detectUnnecessaryElse(tempFile)
	// if err != nil {
	// 	return nil, fmt.Errorf("error detecting unnecessary else: %w", err)
	// }
	// filtered = append(filtered, unnecessaryElseIssues...)

	// unusedFunc, err := e.detectUnusedFunctions(tempFile)
	// if err != nil {
	// 	return nil, fmt.Errorf("error detecting unused functions: %w", err)
	// }
	// filtered = append(filtered, unusedFunc...)

	// // map issues back to .gno file if necessary
	// if strings.HasSuffix(filename, ".gno") {
	// 	for i := range filtered {
	// 		filtered[i].Filename = filename
	// 	}
	// }

	// return filtered, nil

	var allIssues []Issue
	for _, rule := range e.rules {
		issues, err := rule.Check(tempFile)
		if err != nil {
			return nil, fmt.Errorf("error running lint rule: %w", err)
		}
		allIssues = append(allIssues, issues...)
	}

	filtered := e.filterUndefinedIssues(allIssues)

	// map issues back to .gno file if necessary
	if strings.HasSuffix(filename, ".gno") {
		for i := range filtered {
			filtered[i].Filename = filename
		}
	}

	return filtered, nil
}

func (e *Engine) filterUndefinedIssues(issues []Issue) []Issue {
	var filtered []Issue
	for _, issue := range issues {
		if issue.Rule == "typecheck" && strings.Contains(issue.Message, "undefined:") {
			symbol := strings.TrimSpace(strings.TrimPrefix(issue.Message, "undefined:"))
			if e.SymbolTable.IsDefined(symbol) {
				// ignore issues if the symbol is defined in the symbol table
				continue
			}
		}
		filtered = append(filtered, issue)
	}
	return filtered
}

type golangciOutput struct {
	Issues []struct {
		FromLinter string `json:"FromLinter"`
		Text       string `json:"Text"`
		Pos        struct {
			Filename string `json:"Filename"`
			Line     int    `json:"Line"`
			Column   int    `json:"Column"`
		} `json:"Pos"`
	} `json:"Issues"`
}

func runGolangciLint(filename string) ([]Issue, error) {
	cmd := exec.Command("golangci-lint", "run", "--out-format=json", filename)
	output, _ := cmd.CombinedOutput()

	var golangciResult golangciOutput
	if err := json.Unmarshal(output, &golangciResult); err != nil {
		return nil, fmt.Errorf("error unmarshaling golangci-lint output: %w", err)
	}

	var issues []Issue
	for _, gi := range golangciResult.Issues {
		issues = append(issues, Issue{
			Rule:     gi.FromLinter,
			Filename: gi.Pos.Filename, // Use the filename from golangci-lint output
			Start:    token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1},
			Message:  gi.Text,
		})
	}

	return issues, nil
}

func createTempGoFile(gnoFile string) (string, error) {
	content, err := os.ReadFile(gnoFile)
	if err != nil {
		return "", fmt.Errorf("error reading .gno file: %w", err)
	}

	dir := filepath.Dir(gnoFile)
	tempFile, err := os.CreateTemp(dir, "temp_*.go")
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %w", err)
	}

	_, err = tempFile.Write(content)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("error writing to temp file: %w", err)
	}

	err = tempFile.Close()
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("error closing temp file: %w", err)
	}

	return tempFile.Name(), nil
}
