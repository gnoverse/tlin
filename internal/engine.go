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
	ignoredRules map[string]bool
	nolintMgr    *nolint.Manager
	rules        []LintRule
	defaultRules []LintRule
	allRules     []LintRule
}

// NewEngine creates a new lint engine.
func NewEngine(rootDir string, source []byte, rules map[string]tt.ConfigRule) (*Engine, error) {
	engine := &Engine{}
	engine.applyRules(rules)

	return engine, nil
}

func (e *Engine) applyRules(rules map[string]tt.ConfigRule) {
	e.registerDefaultRules()
	e.registerAllRules()

	// Iterate over the rules and apply severity
	for _, rule := range e.allRules {
		if _, ok := rules[rule.Name()]; ok {
			severity := rules[rule.Name()].Severity
			appliedRule := e.findRule(rule.Name())
			if appliedRule != nil {
				// if severity is OFF, ignore the rule
				if severity == tt.SeverityOff {
					e.ignoreRule(rule.Name())
					continue
				}
				// set the severity of the rule
				(*appliedRule).SetSeverity(severity)
			} else {
				// Add a new rule with the given severity
				rule.SetSeverity(severity)
				e.rules = append(e.rules, rule)
			}
		}
	}
}

func (e *Engine) registerAllRules() {
	allRules := []LintRule{
		&GolangciLintRule{},
		&DeprecateFuncRule{},
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
	e.allRules = append(e.allRules, allRules...)
}

func (e *Engine) registerDefaultRules() {
	e.defaultRules = []LintRule{
		NewGolangciLintRule(),
		NewDeprecateFuncRule(),
		NewEarlyReturnOpportunityRule(),
		NewSimplifySliceExprRule(),
		NewUnnecessaryConversionRule(),
		NewLoopAllocationRule(),
		NewEmitFormatRule(),
		NewDetectCycleRule(),
		NewGnoSpecificRule(),
		NewRepeatedRegexCompilationRule(),
		NewUselessBreakRule(),
		NewDeferRule(),
		NewMissingModPackageRule(),
	}
	e.rules = append(e.rules, e.defaultRules...)
}

func (e *Engine) findRule(name string) *LintRule {
	for _, rule := range e.rules {
		if rule.Name() == name {
			return &rule
		}
	}
	return nil
}

func (e *Engine) ignoreRule(name string) {
	if e.ignoredRules == nil {
		e.ignoredRules = make(map[string]bool)
	}
	e.ignoredRules[name] = true
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

			severityIssues := e.applySeverity(issues, r)
			nolinted := e.filterNolintIssues(severityIssues)

			mu.Lock()
			allIssues = append(allIssues, nolinted...)
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

			severityIssues := e.applySeverity(issues, r)
			nolinted := e.filterNolintIssues(severityIssues)

			mu.Lock()
			allIssues = append(allIssues, nolinted...)
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

func (e *Engine) applySeverity(issues []tt.Issue, rule LintRule) []tt.Issue {
	for i := range issues {
		issues[i].Severity = rule.Severity()
	}
	return issues
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
