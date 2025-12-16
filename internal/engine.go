package internal

import (
	"fmt"
	"go/ast"
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
//
// TODO: use symbol table
type Engine struct {
	ignoredPaths []string            // Readonly
	ignoredRules map[string]bool     // Readonly
	rules        map[string]LintRule // Readonly
}

// runContext holds per-run state to ensure thread safety during concurrent linting.
// Each call to Run or RunSource creates its own context.
type runContext struct {
	originalPath string          // Original filename (.gno or .go)
	tempPath     string          // Temporary file path if converted from .gno
	node         *ast.File       // Parsed AST
	fset         *token.FileSet  // Token file set
	nolintMgr    *nolint.Manager // Nolint manager for this specific run
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
			newRule.severity = rule.Severity
			e.rules[key] = newRule
		} else {
			if rule.Severity == tt.SeverityOff {
				e.IgnoreRule(key)
			}
			r.severity = rule.Severity
			e.rules[key] = r
		}
	}
}

func (e *Engine) registerDefaultRules() {
	// iterate over allRules and add them to the rules map if severity is not off
	for key, newRule := range allRules {
		if newRule.Severity() != tt.SeverityOff {
			newRule.name = key
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
	// Create run context for this specific run
	ctx, err := e.createRunContext(filename)
	if err != nil {
		return nil, err
	}
	defer e.cleanupContext(ctx)

	return e.runWithContext(ctx)
}

// RunSource applies all lint rules to the given source and returns a slice of Issues.
func (e *Engine) RunSource(source []byte) ([]tt.Issue, error) {
	// Create run context for source
	ctx, err := e.createRunContextFromSource(source)
	if err != nil {
		return nil, err
	}
	// No cleanup needed for source-based runs

	return e.runWithContext(ctx)
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

// createRunContext creates a new run context for a file.
func (e *Engine) createRunContext(filename string) (*runContext, error) {
	ctx := &runContext{
		originalPath: filename,
	}

	// Prepare temp file if needed
	tempFile, err := e.prepareFile(filename)
	if err != nil {
		return nil, err
	}
	ctx.tempPath = tempFile

	// Parse file
	node, fset, err := lints.ParseFile(tempFile, nil)
	if err != nil {
		e.cleanupTemp(tempFile)
		return nil, fmt.Errorf("error parsing file: %w", err)
	}

	ctx.node = node
	ctx.fset = fset
	ctx.nolintMgr = nolint.ParseComments(node, fset)

	return ctx, nil
}

// createRunContextFromSource creates a new run context from source bytes.
func (e *Engine) createRunContextFromSource(source []byte) (*runContext, error) {
	node, fset, err := lints.ParseFile("", source)
	if err != nil {
		return nil, fmt.Errorf("error parsing content: %w", err)
	}

	ctx := &runContext{
		originalPath: "",
		tempPath:     "",
		node:         node,
		fset:         fset,
		nolintMgr:    nolint.ParseComments(node, fset),
	}

	return ctx, nil
}

// cleanupContext cleans up resources associated with a run context.
func (e *Engine) cleanupContext(ctx *runContext) {
	if ctx.tempPath != "" {
		e.cleanupTemp(ctx.tempPath)
	}
}

// runWithContext executes all lint rules using the provided context.
func (e *Engine) runWithContext(ctx *runContext) ([]tt.Issue, error) {
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
			// Use temp path for checking (golangci-lint needs .go files)
			checkPath := ctx.tempPath
			if checkPath == "" {
				checkPath = ctx.originalPath
			}
			issues, err := r.Check(checkPath, ctx.node, ctx.fset)
			if err != nil {
				return
			}

			nolinted := e.filterNolintIssues(issues, ctx)
			noIgnoredPaths := e.filterIgnoredPaths(nolinted, ctx)

			mu.Lock()
			allIssues = append(allIssues, noIgnoredPaths...)
			mu.Unlock()
		}(rule)
	}
	wg.Wait()

	// Map issues back to original filename only if we used a temp file that differs from the original
	// This ensures we only remap for .gno -> .go conversions, not for regular .go files
	if ctx.tempPath != "" && ctx.originalPath != "" && ctx.tempPath != ctx.originalPath {
		for i := range allIssues {
			// Only remap if the issue is pointing to the temp file
			if allIssues[i].Filename == ctx.tempPath {
				allIssues[i].Filename = ctx.originalPath
			}
		}
	}

	return allIssues, nil
}

func (e *Engine) filterIgnoredPaths(issues []tt.Issue, ctx *runContext) []tt.Issue {
	filtered := make([]tt.Issue, 0, len(issues))
	for _, issue := range issues {
		// Use original path for ignored path checking
		checkPath := issue.Filename
		if ctx.originalPath != "" && ctx.tempPath != "" && issue.Filename == ctx.tempPath {
			checkPath = ctx.originalPath
		}
		if !e.isIgnoredPath(checkPath) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

func (e *Engine) isIgnoredPath(path string) bool {
	for _, ignored := range e.ignoredPaths {
		res, err := filepath.Match(ignored, path)
		if err == nil && res {
			return true
		}
	}
	return false
}

// filterNolintIssues filters issues based on nolint comments.
func (e *Engine) filterNolintIssues(issues []tt.Issue, ctx *runContext) []tt.Issue {
	if ctx.nolintMgr == nil {
		return issues
	}
	filtered := make([]tt.Issue, 0, len(issues))
	for _, issue := range issues {
		pos := token.Position{
			Filename: issue.Filename,
			Line:     issue.Start.Line,
		}
		if !ctx.nolintMgr.IsNolint(pos, issue.Rule) {
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
