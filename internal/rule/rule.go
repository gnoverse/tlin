// Package rule defines the contract every lint rule implements and a
// registry rules can register themselves into.
//
// The Rule interface plus the AnalysisContext value passed into Check
// replace the legacy LintRule struct (function-pointer container) used
// by the engine today. During the rule-registration refactor the
// engine first wraps existing rules with the legacyRule adapter
// (legacy.go) and then ports each rule to implement Rule directly.
package rule

import (
	tt "github.com/gnolang/tlin/internal/types"
)

// Rule is the contract every lint rule implements.
//
// Name is the single source of truth for the rule's identity: the
// engine uses it as the registry key, severity overrides in
// .tlin.yaml are keyed off it, nolint comments match it, and Issue.Rule
// must equal it. Embedding the name as a method on the rule (rather
// than as map-key metadata injected from outside) is what removes the
// "two sources of truth" problem the legacy LintRule struct had.
type Rule interface {
	// Name returns the canonical rule key.
	Name() string

	// DefaultSeverity is the severity used when configuration does
	// not override it. With SeverityUnset as the zero value of
	// Severity (a later refactor), an unset config entry will resolve
	// to this value instead of silently becoming SeverityError.
	DefaultSeverity() tt.Severity

	// Check runs the rule against ctx and returns any issues found.
	// ctx.Severity holds the severity the engine has resolved for
	// this rule on this run (config override or DefaultSeverity).
	Check(ctx *AnalysisContext) ([]tt.Issue, error)
}
