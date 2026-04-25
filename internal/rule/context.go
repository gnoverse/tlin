package rule

import (
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"sync"

	"github.com/gnolang/tlin/internal/nolint"
	tt "github.com/gnolang/tlin/internal/types"
)

// AnalysisContext bundles per-file data passed into Rule.Check.
//
// One context is shared across every rule's Check call for a given
// file run, which is what makes TypesInfo()'s sync.Once cache useful
// (the type checker runs at most once per file). Severity is
// mutated between calls and rules MUST NOT retain the pointer
// beyond Check — rules run sequentially per file, which makes the
// mutation safe.
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
	// Source is the raw bytes the parser read. May be nil.
	Source []byte

	typesInfoOnce sync.Once
	typesInfo     *types.Info
}

// TypesInfo returns a *types.Info populated by a single types.Check
// pass on File, computed lazily and cached for the lifetime of the
// context. Type-check errors are swallowed — the result is
// best-effort and rules must tolerate missing entries.
func (ctx *AnalysisContext) TypesInfo() *types.Info {
	ctx.typesInfoOnce.Do(func() {
		ctx.typesInfo = &types.Info{
			Types: map[ast.Expr]types.TypeAndValue{},
			Uses:  map[*ast.Ident]types.Object{},
			Defs:  map[*ast.Ident]types.Object{},
		}
		if ctx.File == nil || ctx.Fset == nil {
			return
		}
		cfg := &types.Config{Importer: importer.Default()}
		_, _ = cfg.Check(ctx.File.Name.Name, ctx.Fset, []*ast.File{ctx.File}, ctx.typesInfo)
	})
	return ctx.typesInfo
}
