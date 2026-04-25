package internal

import (
	"context"
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
	source       []byte          // Raw bytes the parser read (read once, shared with rules)
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
		r, ok := e.rules[key]
		if !ok {
			continue
		}
		if cfg.Severity == tt.SeverityOff {
			e.IgnoreRule(key)
			continue
		}
		e.severityOverrides[key] = cfg.Severity
		if cfg.Data == nil {
			continue
		}
		// Fail-open on a per-rule config error: the rule keeps its
		// defaults and the engine still runs. Hard-failing here would
		// take down a whole tlin invocation over one bad config block.
		cr, ok := r.(rule.ConfigurableRule)
		if !ok {
			continue
		}
		if err := cr.ParseConfig(cfg.Data); err != nil {
			e.logger.Warn("rule config parse failed",
				zap.String("rule", key),
				zap.Error(err))
		}
	}
}

func (e *Engine) registerDefaultRules() {
	registered := rule.All()
	if e.registry != nil {
		registered = e.registry.All()
	}
	for name, r := range registered {
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

// Run applies all lint rules to the given file without cancellation.
// Prefer RunWithContext when a context is available.
func (e *Engine) Run(filename string) ([]tt.Issue, error) {
	return e.RunWithContext(context.Background(), filename)
}

// RunSource applies all lint rules to the given source without
// cancellation. Prefer RunSourceWithContext when a context is
// available.
func (e *Engine) RunSource(source []byte) ([]tt.Issue, error) {
	return e.RunSourceWithContext(context.Background(), source)
}

// RunWithContext applies all lint rules to the file at filename.
// Cancellation is observed at rule boundaries (~one rule of latency);
// a cancelled run returns ctx.Err() alongside whatever issues were
// collected before the cancellation took effect.
//
// Both file-level Rule.Check and PackageRule.CheckPackage fire here:
// the latter receives a one-file PackageContext via SinglePackage so
// engine.Run callers (single-file paths, tests) keep emitting the
// same issues a directory walk would. Directory walks should call
// RunFile + RunPackage to amortize PackageRule work across siblings.
func (e *Engine) RunWithContext(ctx context.Context, filename string) ([]tt.Issue, error) {
	state, err := e.createRunContext(filename)
	if err != nil {
		return nil, err
	}
	defer e.cleanupContext(state)

	return e.runWithContext(ctx, state, false)
}

func (e *Engine) RunSourceWithContext(ctx context.Context, source []byte) ([]tt.Issue, error) {
	state, err := e.createRunContextFromSource(source)
	if err != nil {
		return nil, err
	}
	return e.runWithContext(ctx, state, false)
}

// RunFile applies only file-level rules to filename, skipping any
// PackageRule. Intended for directory walks that pair RunFile with
// RunPackage so package-level work runs once per directory rather
// than once per file.
func (e *Engine) RunFile(ctx context.Context, filename string) ([]tt.Issue, error) {
	state, err := e.createRunContext(filename)
	if err != nil {
		return nil, err
	}
	defer e.cleanupContext(state)

	return e.runWithContext(ctx, state, true)
}

// RunPackage runs every PackageRule once across the supplied paths,
// which must all live in dir. The engine handles .gno → temp .go
// conversion and per-file nolint manager construction for filtering.
// File-level rules are not dispatched here — pair with RunFile per
// path to cover both layers.
func (e *Engine) RunPackage(ctx context.Context, dir string, paths []string) ([]tt.Issue, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	workingPaths := make([]string, len(paths))
	nolintMgrs := make(map[string]*nolint.Manager, len(paths))
	var cleanups []string
	defer func() {
		for _, p := range cleanups {
			e.cleanupTemp(p)
		}
	}()

	for i, p := range paths {
		wp, err := e.prepareFile(p)
		if err != nil {
			return nil, fmt.Errorf("error preparing %s: %w", p, err)
		}
		workingPaths[i] = wp
		if wp != p {
			cleanups = append(cleanups, wp)
		}
		// Best-effort nolint manager: if parsing fails the rule still
		// runs, the file just won't honor //nolint directives. This
		// matches the per-file path's behavior (createRunContext
		// returns an error there, but here we'd rather keep the rest
		// of the package on the floor).
		source, readErr := os.ReadFile(wp)
		if readErr != nil {
			continue
		}
		node, fset, parseErr := lints.ParseFile(wp, source)
		if parseErr != nil {
			continue
		}
		nolintMgrs[wp] = nolint.ParseComments(node, fset)
	}

	pctx := &rule.PackageContext{
		Dir:           dir,
		OriginalPaths: paths,
		WorkingPaths:  workingPaths,
	}

	var allIssues []tt.Issue
	for _, r := range e.rules {
		if err := ctx.Err(); err != nil {
			return allIssues, err
		}
		pr, ok := r.(rule.PackageRule)
		if !ok {
			continue
		}
		if e.ignoredRules[r.Name()] {
			continue
		}
		sev := e.effectiveSeverity(r)
		if sev == tt.SeverityOff {
			continue
		}
		pctx.Severity = sev
		issues, err := pr.CheckPackage(ctx, pctx)
		if err != nil {
			e.logger.Warn("rule check failed",
				zap.String("rule", r.Name()),
				zap.String("dir", dir),
				zap.Error(err))
			continue
		}
		nolinted := e.filterPackageNolint(issues, pctx, nolintMgrs)
		noIgnoredPaths := e.filterIgnoredPaths(nolinted)
		allIssues = append(allIssues, noIgnoredPaths...)
	}
	return allIssues, nil
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
	state := &runContext{
		originalPath: filename,
	}

	// Prepare temp file if needed
	tempFile, err := e.prepareFile(filename)
	if err != nil {
		return nil, err
	}
	state.tempPath = tempFile

	source, err := os.ReadFile(tempFile)
	if err != nil {
		e.cleanupTemp(tempFile)
		return nil, fmt.Errorf("error reading file: %w", err)
	}
	state.source = source

	node, fset, err := lints.ParseFile(tempFile, source)
	if err != nil {
		e.cleanupTemp(tempFile)
		return nil, fmt.Errorf("error parsing file: %w", err)
	}

	state.node = node
	state.fset = fset
	state.nolintMgr = nolint.ParseComments(node, fset)

	return state, nil
}

// createRunContextFromSource creates a new run context from source bytes.
func (e *Engine) createRunContextFromSource(source []byte) (*runContext, error) {
	node, fset, err := lints.ParseFile("", source)
	if err != nil {
		return nil, fmt.Errorf("error parsing content: %w", err)
	}

	return &runContext{
		source:    source,
		node:      node,
		fset:      fset,
		nolintMgr: nolint.ParseComments(node, fset),
	}, nil
}

// cleanupContext cleans up resources associated with a run context.
func (e *Engine) cleanupContext(state *runContext) {
	if state.tempPath != "" {
		e.cleanupTemp(state.tempPath)
	}
}

// runWithContext executes all lint rules using the provided context.
//
// Rules run sequentially. The previous fan-out spawned len(rules)
// goroutines per file, on top of lint.processDirectory's per-file
// worker pool — multiplying total goroutines by ~12. File-level
// parallelism (the outer worker pool) is preserved; only the inner
// per-rule fan-out is gone.
//
// runWithContext checks ctx.Done() between rules so a cancellation
// propagates within roughly one rule's worth of work. A cancelled
// run returns ctx.Err() alongside whatever issues were already
// collected before the cancellation took effect.
func (e *Engine) runWithContext(ctx context.Context, state *runContext, skipPackageRules bool) ([]tt.Issue, error) {
	// Use temp path for checking (golangci-lint needs .go files).
	workingPath := state.tempPath
	if workingPath == "" {
		workingPath = state.originalPath
	}

	actx := &rule.AnalysisContext{
		OriginalPath: state.originalPath,
		WorkingPath:  workingPath,
		File:         state.node,
		Fset:         state.fset,
		NolintMgr:    state.nolintMgr,
		Source:       state.source,
	}

	var allIssues []tt.Issue
	for _, r := range e.rules {
		if err := ctx.Err(); err != nil {
			return allIssues, err
		}
		if e.ignoredRules[r.Name()] {
			continue
		}
		// A rule whose effective severity resolves to Off is silent
		// for this run — either because the user set severity:off in
		// .tlin.yaml, or because the rule's DefaultSeverity is Off
		// and no config override opted it in. Skip dispatch entirely
		// rather than letting it emit issues that would render as
		// "OFF" severity.
		sev := e.effectiveSeverity(r)
		if sev == tt.SeverityOff {
			continue
		}
		actx.Severity = sev

		var (
			issues []tt.Issue
			err    error
		)
		if pr, isPkg := r.(rule.PackageRule); isPkg {
			if skipPackageRules {
				continue
			}
			// Engine path threads the real ctx into PackageRule;
			// the rule's own Check would fall back to context.Background.
			issues, err = pr.CheckPackage(ctx, actx.SinglePackage())
		} else {
			issues, err = r.Check(actx)
		}
		if err != nil {
			e.logger.Warn("rule check failed",
				zap.String("rule", r.Name()),
				zap.String("file", state.originalPath),
				zap.Error(err))
			continue
		}

		nolinted := e.filterNolintIssues(issues, state)
		noIgnoredPaths := e.filterIgnoredPaths(nolinted)
		allIssues = append(allIssues, noIgnoredPaths...)
	}

	return allIssues, nil
}

// filterPackageNolint applies the per-file nolint managers built in
// RunPackage to issues emitted by a PackageRule. The issue's
// Filename is OriginalPath (the rule emits via RemapFilename), so we
// reverse that map back to WorkingPath to find the matching manager.
func (e *Engine) filterPackageNolint(issues []tt.Issue, pctx *rule.PackageContext, mgrs map[string]*nolint.Manager) []tt.Issue {
	if len(mgrs) == 0 || len(issues) == 0 {
		return issues
	}
	o2w := make(map[string]string, len(pctx.OriginalPaths))
	for i, op := range pctx.OriginalPaths {
		o2w[op] = pctx.WorkingPaths[i]
	}
	filtered := make([]tt.Issue, 0, len(issues))
	for _, issue := range issues {
		wp := o2w[issue.Filename]
		mgr := mgrs[wp]
		if mgr == nil {
			filtered = append(filtered, issue)
			continue
		}
		pos := token.Position{Filename: wp, Line: issue.Start.Line}
		if !mgr.IsNolint(pos, issue.Rule) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
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
//
// Issue.Filename is OriginalPath (the convention every rule emits
// post-EPR-3), but the nolint manager keyed its scopes off the path
// the parser actually saw — that's WorkingPath when the engine
// converted .gno → temp .go. We swap back to WorkingPath when the
// run had a temp file so the manager finds the scope.
func (e *Engine) filterNolintIssues(issues []tt.Issue, state *runContext) []tt.Issue {
	if state.nolintMgr == nil {
		return issues
	}
	lookupFilename := func(emitted string) string {
		if state.tempPath != "" && state.tempPath != state.originalPath && emitted == state.originalPath {
			return state.tempPath
		}
		return emitted
	}
	filtered := make([]tt.Issue, 0, len(issues))
	for _, issue := range issues {
		pos := token.Position{
			Filename: lookupFilename(issue.Filename),
			Line:     issue.Start.Line,
		}
		if !state.nolintMgr.IsNolint(pos, issue.Rule) {
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
