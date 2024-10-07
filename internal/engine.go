package internal

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gnolang/tlin/internal/lints"
	tt "github.com/gnolang/tlin/internal/types"
)

// Engine manages the linting process.
type Engine struct {
	SymbolTable  *SymbolTable
	rules        []LintRule
	ignoredRules map[string]bool
	defaultRules []LintRule
	nolintMgr    *nolintManager
}

// NewEngine creates a new lint engine.
func NewEngine(rootDir string, source []byte) (*Engine, error) {
	st, err := BuildSymbolTable(rootDir, source)
	if err != nil {
		return nil, fmt.Errorf("error building symbol table: %w", err)
	}

	engine := &Engine{SymbolTable: st}
	engine.initDefaultRules()

	return engine, nil
}

// registerDefaultRules adds the default set of lint rules to the engine.
func (e *Engine) registerDefaultRules() {
	e.rules = append(e.rules, e.defaultRules...)
}

func (e *Engine) initDefaultRules() {
	e.defaultRules = []LintRule{
		&GolangciLintRule{},
		&EarlyReturnOpportunityRule{},
		&SimplifySliceExprRule{},
		&UnnecessaryConversionRule{},
		&LoopAllocationRule{},
		&EmitFormatRule{},
		&DetectCycleRule{},
		&GnoSpecificRule{},
		&RepeatedRegexCompilationRule{},
		&UselessBreakRule{},
		&DeferRule{},
		&MissingModPackageRule{},
	}
	e.registerDefaultRules()
}

// Run applies all lint rules to the given file and returns a slice of Issues.
func (e *Engine) Run(filename string) ([]tt.Issue, error) {
	if strings.HasSuffix(filename, ".mod") {
		return e.runModCheck(filename)
	}

	tempFile, err := e.prepareFile(filename)
	if err != nil {
		return nil, err
	}
	defer e.cleanupTemp(tempFile)

	node, fset, err := lints.ParseFile(tempFile, nil)
	if err != nil {
		return nil, fmt.Errorf("error parsing file: %w", err)
	}

	e.nolintMgr = ParseNolintComments(node, fset)

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

			mu.Lock()
			allIssues = append(allIssues, nolinted...)
			mu.Unlock()
		}(rule)
	}
	wg.Wait()

	filtered := e.filterUndefinedIssues(allIssues)

	// map issues back to .gno file if necessary
	if strings.HasSuffix(filename, ".gno") {
		for i := range filtered {
			filtered[i].Filename = filename
		}
	}

	return filtered, nil
}

// Run applies all lint rules to the given source and returns a slice of Issues.
func (e *Engine) RunSource(source []byte) ([]tt.Issue, error) {
	node, fset, err := lints.ParseFile("", source)
	if err != nil {
		return nil, fmt.Errorf("error parsing content: %w", err)
	}

	e.nolintMgr = ParseNolintComments(node, fset)

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

			mu.Lock()
			allIssues = append(allIssues, nolinted...)
			mu.Unlock()
		}(rule)
	}
	wg.Wait()

	filtered := e.filterUndefinedIssues(allIssues)

	return filtered, nil
}

func (e *Engine) IgnoreRule(rule string) {
	if e.ignoredRules == nil {
		e.ignoredRules = make(map[string]bool)
	}
	e.ignoredRules[rule] = true
}

func (e *Engine) prepareFile(filename string) (string, error) {
	if strings.HasSuffix(filename, ".gno") {
		return createTempGoFile(filename)
	}
	return filename, nil
}

func (e *Engine) runModCheck(filename string) ([]tt.Issue, error) {
	var allIssues []tt.Issue
	for _, rule := range e.rules {
		if e.ignoredRules[rule.Name()] {
			continue
		}
		if modRule, ok := rule.(ModRule); ok {
			issues, err := modRule.CheckMod(filename)
			if err != nil {
				return nil, fmt.Errorf("error checking .mod file: %w", err)
			}
			allIssues = append(allIssues, issues...)
		}
	}
	return allIssues, nil
}

func (e *Engine) cleanupTemp(temp string) {
	if temp != "" && strings.HasPrefix(filepath.Base(temp), "temp_") {
		_ = os.Remove(temp)
	}
}

// filterUndefinedIssue filters out golangci-lint's undefined symbol issues.
// TODO: This is a temporary fix. need to find a better way to handle this.
func (e *Engine) filterUndefinedIssues(issues []tt.Issue) []tt.Issue {
	filtered := make([]tt.Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.Rule == "typecheck" && strings.HasPrefix(issue.Message, "undefined:") {
			symbol := strings.TrimSpace(issue.Message[10:])
			if e.SymbolTable.IsDefined(symbol) {
				continue
			}
		}
		filtered = append(filtered, issue)
	}
	return filtered
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
			Column:   issue.Start.Column,
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
