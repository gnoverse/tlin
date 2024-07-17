package internal

import (
	"encoding/json"
	"fmt"
	"go/token"
	"os"
	"os/exec"
	"strings"
)

// SourceCode stores the content of a source code file.
type SourceCode struct {
	Lines []string
}

// ReadSourceFile reads the content of a file and returns it as a `SourceCode` struct.
func ReadSourceCode(filename string) (*SourceCode, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	return &SourceCode{Lines: lines}, nil
}

// Issue represents a lint issue found in the code base.
type Issue struct {
	Rule     string
	Filename string
	Start    token.Position
	End      token.Position
	Message  string
}

// Engine manages the linting process.
type Engine struct {
	SymbolTable *SymbolTable
}

// NewEngine creates a new lint engine.
func NewEngine(rootDir string) (*Engine, error) {
	st, err := BuildSymbolTable(rootDir)
	if err != nil {
		return nil, fmt.Errorf("error building symbol table: %w", err)
	}
	return &Engine{SymbolTable: st}, nil
}

// Run applies golangci-lint to the given file and returns a slice of issues.
func (e *Engine) Run(filename string) ([]Issue, error) {
	issues, err := runGolangciLint(filename)
	if err != nil {
		return nil, fmt.Errorf("error running golangci-lint: %w", err)
	}
	filtered := e.filterUndefinedIssues(issues)

	unnecessaryElseIssues, err := e.detectUnnecessaryElse(filename)
	if err != nil {
		return nil, fmt.Errorf("error detecting unnecessary else: %w", err)
	}
	filtered = append(filtered, unnecessaryElseIssues...)

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
			Filename: gi.Pos.Filename,
			Start:    token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1}, // Set End to Start + 1 column
			Message:  gi.Text,
		})
	}

	return issues, nil
}
