package rule

import (
	"go/ast"
	"go/token"

	tt "github.com/gnolang/tlin/internal/types"
)

// LegacyCheck is the per-rule signature used by the existing
// internal.LintRule struct: a function taking filename, parsed AST,
// fileset, and severity, returning issues. Every built-in rule
// currently has this shape.
type LegacyCheck func(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error)

// NewLegacy wraps an old-style check function as a Rule. The engine
// uses this to migrate the existing allRules entries to the new
// interface without rewriting each rule body.
//
// This adapter is temporary and is removed in a later PR once every
// rule implements Rule directly.
func NewLegacy(name string, defaultSeverity tt.Severity, check LegacyCheck) Rule {
	return &legacyRule{name: name, defaultSeverity: defaultSeverity, check: check}
}

type legacyRule struct {
	name            string
	defaultSeverity tt.Severity
	check           LegacyCheck
}

func (l *legacyRule) Name() string                 { return l.name }
func (l *legacyRule) DefaultSeverity() tt.Severity { return l.defaultSeverity }

// Check forwards to the wrapped function. The engine sets
// ctx.Severity to the resolved severity for this run (config override
// or DefaultSeverity), so legacy checks see exactly what they would
// have seen under the old r.severity field.
func (l *legacyRule) Check(ctx *AnalysisContext) ([]tt.Issue, error) {
	return l.check(ctx.WorkingPath, ctx.File, ctx.Fset, ctx.Severity)
}
