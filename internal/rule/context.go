package rule

import (
	"go/ast"
	"go/token"

	"github.com/gnolang/tlin/internal/nolint"
	tt "github.com/gnolang/tlin/internal/types"
)

// AnalysisContext bundles per-file data passed into Rule.Check. The
// engine builds one context per file per run.
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
	// NolintMgr provides nolint comment resolution. May be nil for
	// source-only runs.
	NolintMgr *nolint.Manager
	// Severity is the resolved severity for this rule on this run
	// (config override or DefaultSeverity). Rules embed this in
	// Issue.Severity.
	Severity tt.Severity
}
