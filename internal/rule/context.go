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
	// Fset is the FileSet that produced File. Always non-nil during
	// engine dispatch; Position keeps a defensive nil branch so
	// hand-built test contexts (RunSource fixtures, unit-test
	// constructions) don't panic.
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

// NewIssue builds a tt.Issue with Filename, Start, and End all
// remapped from WorkingPath to OriginalPath, so user-visible paths
// stay consistent even when the engine ran the parser against a
// temporary .go file copied from a .gno source.
func (ctx *AnalysisContext) NewIssue(ruleName string, start, end token.Pos) tt.Issue {
	return tt.Issue{
		Rule:     ruleName,
		Filename: ctx.OriginalPath,
		Start:    ctx.Position(start),
		End:      ctx.Position(end),
		Severity: ctx.Severity,
	}
}

// Position returns Fset.Position(p) with Filename remapped from
// WorkingPath to OriginalPath.
func (ctx *AnalysisContext) Position(p token.Pos) token.Position {
	if ctx.Fset == nil {
		return token.Position{Filename: ctx.OriginalPath}
	}
	pos := ctx.Fset.Position(p)
	pos.Filename = ctx.RemapFilename(pos.Filename)
	return pos
}

// RemapFilename returns OriginalPath when name equals WorkingPath,
// otherwise name unchanged. Empty WorkingPath disables the remap so
// source-only runs (RunSource) pass through.
func (ctx *AnalysisContext) RemapFilename(name string) string {
	if ctx.WorkingPath == "" || name != ctx.WorkingPath {
		return name
	}
	return ctx.OriginalPath
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
