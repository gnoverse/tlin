package rule

import (
	"context"
	"path/filepath"
	"sync"

	tt "github.com/gnolang/tlin/internal/types"
)

// PackageRule is implemented by rules that benefit from per-package
// amortization — typically rules that load packages or invoke external
// tools, where doing the work once per directory is dramatically
// cheaper than once per file.
//
// Implementations must still satisfy Rule.Check so that engine.Run
// (single-file dispatch) keeps working; the conventional wiring is to
// have Check delegate to CheckPackage with a one-file PackageContext
// built from AnalysisContext.SinglePackage().
type PackageRule interface {
	Rule
	CheckPackage(ctx context.Context, pctx *PackageContext) ([]tt.Issue, error)
}

// PackageContext bundles per-package data passed into PackageRule.CheckPackage.
//
// OriginalPaths and WorkingPaths are index-aligned: WorkingPaths[i] is
// the path the parser/loader actually reads (post .gno → temp .go
// conversion), OriginalPaths[i] is the user-visible path the issue
// must reference. Use RemapFilename to translate from working to
// original.
type PackageContext struct {
	// Dir is the package directory the rule should load or invoke
	// tools against. All OriginalPaths share this directory.
	Dir string
	// OriginalPaths is the list of user-supplied file paths for this
	// package, in the order the engine intends to surface them.
	OriginalPaths []string
	// WorkingPaths is the parser-visible path for each entry in
	// OriginalPaths (same .go path for plain .go inputs; a temp .go
	// path for converted .gno inputs).
	WorkingPaths []string
	// Severity is the resolved severity for this rule on this run
	// (config override or DefaultSeverity). Rules embed this in
	// Issue.Severity.
	Severity tt.Severity

	indexOnce sync.Once
	w2o       map[string]string
}

// w2oIndex returns the working→original map, building it once per
// PackageContext. Lookups in InScope and RemapFilename then become
// O(1) instead of O(N) over WorkingPaths. When OriginalPaths is
// shorter than WorkingPaths the missing slots map back to the
// working path itself, which keeps InScope true and makes
// RemapFilename a no-op for that entry — same effective behavior
// as falling off the original linear scan.
func (ctx *PackageContext) w2oIndex() map[string]string {
	ctx.indexOnce.Do(func() {
		ctx.w2o = make(map[string]string, len(ctx.WorkingPaths))
		for i, wp := range ctx.WorkingPaths {
			op := wp
			if i < len(ctx.OriginalPaths) {
				op = ctx.OriginalPaths[i]
			}
			ctx.w2o[wp] = op
		}
	})
	return ctx.w2o
}

// RemapFilename returns the user-visible OriginalPath when name
// matches one of the WorkingPaths, otherwise name unchanged.
func (ctx *PackageContext) RemapFilename(name string) string {
	if name == "" {
		return name
	}
	if op, ok := ctx.w2oIndex()[name]; ok {
		return op
	}
	return name
}

// InScope reports whether name corresponds to one of the working
// files the package context owns. Used to filter loader-supplied
// diagnostics down to the files tlin was asked to inspect.
func (ctx *PackageContext) InScope(name string) bool {
	_, ok := ctx.w2oIndex()[name]
	return ok
}

// SinglePackage returns a PackageContext that holds just this file,
// suitable for PackageRule.Check fallbacks dispatched from the
// single-file engine path. Dir is derived from WorkingPath so
// loaders that scan the directory pick up the temp .go (not the
// untouched .gno sibling).
func (ctx *AnalysisContext) SinglePackage() *PackageContext {
	dir := ""
	if ctx.WorkingPath != "" {
		dir = filepath.Dir(ctx.WorkingPath)
	} else if ctx.OriginalPath != "" {
		dir = filepath.Dir(ctx.OriginalPath)
	}
	return &PackageContext{
		Dir:           dir,
		OriginalPaths: []string{ctx.OriginalPath},
		WorkingPaths:  []string{ctx.WorkingPath},
		Severity:      ctx.Severity,
	}
}
