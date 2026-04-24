package rule

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/gnolang/tlin/internal/nolint"
	tt "github.com/gnolang/tlin/internal/types"
)

// AnalysisContext bundles everything a rule needs for one file.
//
// The engine builds one context per file per run and passes it to
// every rule's Check. Most fields are populated today by the engine
// (OriginalPath, WorkingPath, File, Fset, NolintMgr, Severity).
//
// Source, TypesInfo, and Imports are placeholders for the
// execution-layer refactor: they are reserved here so the Rule.Check
// signature does not change again when that work lands. Rules MAY
// reference these today, with the understanding that:
//
//   - Source is nil unless populated; rules needing the original bytes
//     must fall back to reading WorkingPath themselves until then.
//   - TypesInfo is nil unless the engine wires shared type-checking;
//     callers must nil-check.
//   - Imports is empty unless populated.
type AnalysisContext struct {
	// OriginalPath is the user-supplied path. May end in .gno.
	OriginalPath string
	// WorkingPath is the path the parser actually read. Equals
	// OriginalPath for .go files; differs for .gno files that the
	// engine converts to a temp .go file.
	WorkingPath string
	// File is the parsed AST. Always non-nil inside Check.
	File *ast.File
	// Fset is the FileSet that produced File. Always non-nil.
	Fset *token.FileSet
	// NolintMgr provides nolint comment resolution. May be nil if
	// the engine constructed the context for a source-only run.
	NolintMgr *nolint.Manager
	// Severity is the resolved severity for this rule on this run
	// (config override or DefaultSeverity). Rules embed this in
	// Issue.Severity.
	Severity tt.Severity
	// RuleData carries the rule-specific configuration value from
	// .tlin.yaml's `data:` field, when ConfigurableRule is added in
	// a later PR. Nil for rules that don't implement ConfigurableRule.
	RuleData any

	// Source is the original file bytes. Reserved for the
	// execution-layer refactor (lets rules avoid re-reading the
	// file). Currently nil unless populated.
	Source []byte
	// TypesInfo returns shared type-check information for File.
	// Reserved for the execution-layer refactor (lets rules share
	// one types.Config.Check pass per file). Currently nil unless
	// populated.
	TypesInfo func() *types.Info
	// Imports maps import alias -> import path for File. Reserved
	// for the execution-layer refactor. Currently nil unless
	// populated.
	Imports map[string]string
}
