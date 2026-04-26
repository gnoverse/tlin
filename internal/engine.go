package internal

import (
	"context"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	_ "github.com/gnolang/tlin/internal/lints" // side-effect: rule.Register(...) in init()
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
	src, err := rule.LoadSource(filename)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	return e.runWithContext(ctx, src, false)
}

func (e *Engine) RunSourceWithContext(ctx context.Context, source []byte) ([]tt.Issue, error) {
	src, err := rule.LoadSourceFromBytes(source)
	if err != nil {
		return nil, err
	}
	return e.runWithContext(ctx, src, false)
}

// RunFile applies only file-level rules to filename, skipping any
// PackageRule. Intended for directory walks that pair RunFile with
// RunPackage so package-level work runs once per directory rather
// than once per file.
func (e *Engine) RunFile(ctx context.Context, filename string) ([]tt.Issue, error) {
	src, err := rule.LoadSource(filename)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	return e.runWithContext(ctx, src, true)
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
	sources := make([]*rule.Source, 0, len(paths))
	defer func() {
		for _, s := range sources {
			_ = s.Close()
		}
	}()

	for i, p := range paths {
		// Best-effort load: a parse failure for one file still lets
		// the rest of the package run. The failed file just won't
		// honor //nolint directives and won't appear in WorkingPaths
		// (left empty so RemapFilename / InScope skip it).
		s, err := rule.LoadSource(p)
		if err != nil {
			e.logger.Warn("source load failed",
				zap.String("file", p),
				zap.Error(err))
			continue
		}
		sources = append(sources, s)
		workingPaths[i] = s.WorkingPath
		nolintMgrs[s.WorkingPath] = s.NolintMgr
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
func (e *Engine) runWithContext(ctx context.Context, src *rule.Source, skipPackageRules bool) ([]tt.Issue, error) {
	actx := &rule.AnalysisContext{
		OriginalPath: src.OriginalPath,
		WorkingPath:  src.WorkingPath,
		File:         src.File,
		Fset:         src.Fset,
		NolintMgr:    src.NolintMgr,
		Source:       src.Bytes,
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
				zap.String("file", src.OriginalPath),
				zap.Error(err))
			continue
		}

		nolinted := e.filterNolintIssues(issues, src)
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
func (e *Engine) filterNolintIssues(issues []tt.Issue, src *rule.Source) []tt.Issue {
	if src.NolintMgr == nil {
		return issues
	}
	lookupFilename := func(emitted string) string {
		if src.WorkingPath != "" && src.WorkingPath != src.OriginalPath && emitted == src.OriginalPath {
			return src.WorkingPath
		}
		return emitted
	}
	filtered := make([]tt.Issue, 0, len(issues))
	for _, issue := range issues {
		pos := token.Position{
			Filename: lookupFilename(issue.Filename),
			Line:     issue.Start.Line,
		}
		if !src.NolintMgr.IsNolint(pos, issue.Rule) {
			filtered = append(filtered, issue)
		}
	}
	return filtered
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
