package internal

import (
	"fmt"
	"os"
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

// Run applies all lint rules to the given file and returns a slice of Issues.
func (e *Engine) Run(filename string) ([]Issue, error) {
	tempFile, err := e.prepareFile(filename)
	if err != nil {
		return nil, err
	}
	defer e.cleanupTemp(tempFile)

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

func (e *Engine) prepareFile(filename string) (string, error) {
	if strings.HasSuffix(filename, "gno") {
		return createTempGoFile(filename)
	}
	return filename, nil
}

func (e *Engine) cleanupTemp(temp string) {
	if temp != "" && strings.HasPrefix(filepath.Base(temp), "temp_") {
		_ = os.Remove(temp)
	}
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

// ReadSourceFile reads the content of a file and returns it as a `SourceCode` struct.
func ReadSourceCode(filename string) (*SourceCode, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	return &SourceCode{Lines: lines}, nil
}