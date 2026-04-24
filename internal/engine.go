package internal

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/gnolang/tlin/internal/lints"
	"github.com/gnolang/tlin/internal/nolint"
	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
	"go.uber.org/zap"
)

// Engine manages the linting process.
type Engine struct {
	ignoredPaths      []string               // Readonly
	ignoredRules      map[string]bool        // Readonly
	rules             map[string]rule.Rule   // Readonly
	severityOverrides map[string]tt.Severity // Readonly
	logger            *zap.Logger            // Readonly
	registry          *rule.Registry         // nil = rule package's default registry
}

// Option configures an Engine at construction time. Pass via NewEngine.
type Option func(*Engine)

// WithLogger overrides the engine's logger. Default is zap.NewNop().
// The engine emits Warn entries when a rule's Check returns an error
// instead of silently dropping the failure.
func WithLogger(logger *zap.Logger) Option {
	return func(e *Engine) {
		if logger != nil {
			e.logger = logger
		}
	}
}

// WithRules replaces the engine's rule set entirely, skipping the
// default registration (allRules + registry). Intended for tests that
// need to inject a fake rule (e.g. one that returns an error to
// exercise the logging path). Production code should leave rule
// registration to allRules and the rule package's default registry.
func WithRules(rules map[string]rule.Rule) Option {
	return func(e *Engine) {
		e.rules = rules
	}
}

// WithRegistry swaps the rule registry the engine reads during default
// registration. Without it the engine uses rule.All(); tests pass an
// isolated rule.NewRegistry() so init()-registered production rules
// don't leak in.
func WithRegistry(reg *rule.Registry) Option {
	return func(e *Engine) {
		e.registry = reg
	}
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

// NewEngine creates a new lint engine. Options run before rule
// registration so WithRules and WithRegistry can influence which rules
// the engine ends up holding.
func NewEngine(rules map[string]tt.ConfigRule, opts ...Option) (*Engine, error) {
	engine := &Engine{
		logger: zap.NewNop(),
	}
	for _, opt := range opts {
		opt(engine)
	}
	engine.applyRules(rules)

	return engine, nil
}

func (e *Engine) applyRules(rules map[string]tt.ConfigRule) {
	e.severityOverrides = make(map[string]tt.Severity)
	if e.rules == nil {
		e.rules = make(map[string]rule.Rule)
		e.registerDefaultRules()
	}

	for key, cfg := range rules {
		if _, ok := e.rules[key]; !ok {
			continue
		}
		if cfg.Severity == tt.SeverityOff {
			e.IgnoreRule(key)
			continue
		}
		e.severityOverrides[key] = cfg.Severity
	}
}

// registerDefaultRules merges allRules legacy adapters with the rules
// in the configured registry. A name in both is a hard error — every
// rule's canonical name must have exactly one source of truth.
func (e *Engine) registerDefaultRules() {
	for key, lr := range allRules {
		e.rules[key] = rule.NewLegacy(key, lr.severity, lr.check)
	}
	registered := rule.All()
	if e.registry != nil {
		registered = e.registry.All()
	}
	for name, r := range registered {
		if _, dup := e.rules[name]; dup {
			panic(fmt.Sprintf("rule name conflict: %q is registered both via allRules and via init()", name))
		}
		e.rules[name] = r
	}
}

// effectiveSeverity returns the severity to use for r on this run:
// the config override if one was supplied, otherwise the rule's
// DefaultSeverity.
func (e *Engine) effectiveSeverity(r rule.Rule) tt.Severity {
	if sev, ok := e.severityOverrides[r.Name()]; ok {
		return sev
	}
	return r.DefaultSeverity()
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

func (e *Engine) IgnoreRule(name string) {
	if e.ignoredRules == nil {
		e.ignoredRules = make(map[string]bool)
	}
	e.ignoredRules[name] = true
}

// IgnorePath registers a glob pattern that matches file paths to skip.
// Patterns use doublestar syntax — `**` matches across directory
// separators (e.g. `testdata/**/*.gno`), `*` matches a single path
// segment.
//
// The pattern is normalized to an absolute, cleaned form so callers
// can pass either absolute or relative paths and still match issues
// produced from absolute working paths inside the engine.
func (e *Engine) IgnorePath(pattern string) {
	e.ignoredPaths = append(e.ignoredPaths, normalizeIgnorePattern(pattern))
}

// normalizeIgnorePattern converts a user-supplied path/glob into the
// canonical form used by isIgnoredPath. Glob meta-characters (* ? [)
// are preserved; only directory separators and absolute-vs-relative
// resolution are normalized.
func normalizeIgnorePattern(pattern string) string {
	abs, err := filepath.Abs(pattern)
	if err != nil {
		// Fall back to the cleaned literal if Abs fails (extremely rare —
		// only when os.Getwd fails). Better to keep the user's pattern
		// than to drop it entirely.
		return filepath.Clean(pattern)
	}
	return abs
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
//
// Rules run sequentially. The previous fan-out spawned len(rules)
// goroutines per file, on top of lint.processDirectory's per-file
// worker pool — multiplying total goroutines by ~12. File-level
// parallelism (the outer worker pool) is preserved; only the inner
// per-rule fan-out is gone.
func (e *Engine) runWithContext(ctx *runContext) ([]tt.Issue, error) {
	// Use temp path for checking (golangci-lint needs .go files).
	workingPath := ctx.tempPath
	if workingPath == "" {
		workingPath = ctx.originalPath
	}

	var allIssues []tt.Issue
	for _, r := range e.rules {
		if e.ignoredRules[r.Name()] {
			continue
		}
		actx := &rule.AnalysisContext{
			OriginalPath: ctx.originalPath,
			WorkingPath:  workingPath,
			File:         ctx.node,
			Fset:         ctx.fset,
			NolintMgr:    ctx.nolintMgr,
			Severity:     e.effectiveSeverity(r),
		}
		issues, err := r.Check(actx)
		if err != nil {
			e.logger.Warn("rule check failed",
				zap.String("rule", r.Name()),
				zap.String("file", ctx.originalPath),
				zap.Error(err))
			continue
		}

		nolinted := e.filterNolintIssues(issues, ctx)
		noIgnoredPaths := e.filterIgnoredPaths(nolinted, ctx)
		allIssues = append(allIssues, noIgnoredPaths...)
	}

	// Map issues back to original filename only if we used a temp file that differs from the original.
	// This ensures we only remap for .gno -> .go conversions, not for regular .go files.
	// EPR-3 will replace this post-hoc loop with a structural ctx.NewIssue helper.
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
	if len(e.ignoredPaths) == 0 {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	for _, pattern := range e.ignoredPaths {
		match, err := doublestar.Match(pattern, abs)
		if err == nil && match {
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
