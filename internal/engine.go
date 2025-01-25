package internal

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gnolang/tlin/internal/lints"
	"github.com/gnolang/tlin/internal/nolint"
	tt "github.com/gnolang/tlin/internal/types"
)

// Engine manages the linting process.
// TODO: use symbol table
type Engine struct {
	ignoredPaths []string
	ignoredRules map[string]bool
	nolintMgr    *nolint.Manager
	rules        map[string]LintRule
}

// NewEngine creates a new lint engine.
func NewEngine(rootDir string, source []byte, rules map[string]tt.ConfigRule) (*Engine, error) {
	engine := &Engine{}
	engine.applyRules(rules)

	return engine, nil
}

func (e *Engine) applyRules(rules map[string]tt.ConfigRule) {
	e.rules = make(map[string]LintRule)
	e.registerDefaultRules()

	// Iterate over the rules and apply severity
	for key, rule := range rules {
		r, ok := e.findRule(key)
		if !ok {
			newRule, exists := allRules[key]
			if !exists {
				// Unknown rule, continue to the next one
				continue
			}
			newRule.SetSeverity(rule.Severity)
			e.rules[key] = newRule
		} else {
			if rule.Severity == tt.SeverityOff {
				e.IgnoreRule(key)
			}
			r.SetSeverity(rule.Severity)
		}
	}
}

func (e *Engine) registerDefaultRules() {
	// iterate over allRules and add them to the rules map if severity is not off
	for key, newRule := range allRules {
		if newRule.Severity() != tt.SeverityOff {
			newRule.SetName(key)
			e.rules[key] = newRule
		}
	}
}

func (e *Engine) findRule(name string) (LintRule, bool) {
	rule, exists := e.rules[name]
	return rule, exists
}

// Run applies all lint rules to the given file and returns a slice of Issues.
func (e *Engine) Run(filename string) ([]tt.Issue, error) {
	tempFile, err := e.prepareFile(filename)
	if err != nil {
		return nil, err
	}
	defer e.cleanupTemp(tempFile)

	node, fset, err := lints.ParseFile(tempFile, nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing file: %w", err)
	}

	e.nolintMgr = nolint.ParseComments(node, fset)

	var wg sync.WaitGroup
	var mu sync.Mutex

	var allIssues []tt.Issue
	for _, rule := range e.rules {
		wg.Add(1)
		go func(r LintRule) {
			defer wg.Done()
			if e.ignoredRules[r.Name()] {
				return
			}
			issues, err := r.Check(tempFile, node, fset)
			if err != nil {
				return
			}

			nolinted := e.filterNolintIssues(issues)
			noIgnoredPaths := e.filterIgnoredPaths(nolinted)

			mu.Lock()
			allIssues = append(allIssues, noIgnoredPaths...)
			mu.Unlock()
		}(rule)
	}
	wg.Wait()

	// map issues back to .gno file if necessary
	if strings.HasSuffix(filename, ".gno") {
		for i := range allIssues {
			allIssues[i].Filename = filename
		}
	}

	return allIssues, nil
}

// Run applies all lint rules to the given source and returns a slice of Issues.
func (e *Engine) RunSource(source []byte) ([]tt.Issue, error) {
	node, fset, err := lints.ParseFile("", source)
	if err != nil {
		return nil, fmt.Errorf("error parsing content: %w", err)
	}

	e.nolintMgr = nolint.ParseComments(node, fset)

	var wg sync.WaitGroup
	var mu sync.Mutex

	var allIssues []tt.Issue
	for _, rule := range e.rules {
		wg.Add(1)
		go func(r LintRule) {
			defer wg.Done()
			if e.ignoredRules[r.Name()] {
				return
			}
			issues, err := r.Check("", node, fset)
			if err != nil {
				return
			}

			nolinted := e.filterNolintIssues(issues)
			noIgnoredPaths := e.filterIgnoredPaths(nolinted)

			mu.Lock()
			allIssues = append(allIssues, noIgnoredPaths...)
			mu.Unlock()
		}(rule)
	}
	wg.Wait()

	return allIssues, nil
}

func (e *Engine) IgnoreRule(rule string) {
	if e.ignoredRules == nil {
		e.ignoredRules = make(map[string]bool)
	}
	e.ignoredRules[rule] = true
}

func (e *Engine) IgnorePath(path string) {
	e.ignoredPaths = append(e.ignoredPaths, path)
}

func (e *Engine) prepareFile(filename string) (string, error) {
	if strings.HasSuffix(filename, ".gno") {
		return createTempGoFile(filename)
	}
	return filename, nil
}

func (e *Engine) cleanupTemp(temp string) {
	if temp != "" && strings.HasPrefix(filepath.Base(temp), "temp_") {
		_ = os.Remove(temp)
	}
}

func (e *Engine) filterIgnoredPaths(issues []tt.Issue) []tt.Issue {
	filtered := make([]tt.Issue, 0, len(issues))
	for _, issue := range issues {
		if !e.isIgnoredPath(issue.Filename) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func (e *Engine) isIgnoredPath(path string) bool {
	for _, ignored := range e.ignoredPaths {
		res, err := filepath.Match(ignored, path)
		fmt.Println("res:", res, err, ignored, path)
		if err == nil && res {
			fmt.Println("Ignoring path:", path)
			return true
		}
	}
	return false
}

// filterNolintIssues filters issues based on nolint comments.
func (e *Engine) filterNolintIssues(issues []tt.Issue) []tt.Issue {
	if e.nolintMgr == nil {
		return issues
	}
	filtered := make([]tt.Issue, 0, len(issues))
	for _, issue := range issues {
		pos := token.Position{
			Filename: issue.Filename,
			Line:     issue.Start.Line,
		}
		if !e.nolintMgr.IsNolint(pos, issue.Rule) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// createTempGoFile converts a .gno file to a .go file.
// Since golangci-lint does not support .gno file, we need to convert it to .go file.
// gno has a identical syntax to go, so it is possible to convert it to go file.
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

type ModRule interface {
	LintRule
	CheckMod(filename string) ([]tt.Issue, error)
}
