package lints

import (
	"go/ast"
	"strings"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

func init() {
	rule.Register(unsafePreviousRealmRule{})
}

type unsafePreviousRealmRule struct{}

func (unsafePreviousRealmRule) Name() string                 { return "unsafe-previous-realm" }
func (unsafePreviousRealmRule) DefaultSeverity() tt.Severity { return tt.SeverityError }

func (unsafePreviousRealmRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return DetectUnsafePreviousRealm(ctx)
}

const unsafeRuntimePath = "chain/runtime/unsafe"

// DetectUnsafePreviousRealm flags use of chain/runtime/unsafe in files
// that declare crossing functions (functions with a `realm`-typed
// parameter). unsafe.PreviousRealm() bypasses the cur.IsCurrent()
// frame verification; crossing functions must derive caller identity
// from cur.Previous() instead.
func DetectUnsafePreviousRealm(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	imp := findImport(ctx.File, unsafeRuntimePath)
	if imp == nil || !hasCrossingFunc(ctx.File) {
		return nil, nil
	}
	unsafeName := defaultImportName(unsafeRuntimePath)
	if imp.Name != nil {
		unsafeName = imp.Name.Name
	}

	var issues []tt.Issue
	ast.Inspect(ctx.File, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == unsafeName {
			issue := ctx.NewIssue("unsafe-previous-realm", call.Pos(), call.End())
			issue.Message = unsafeName + "." + sel.Sel.Name + " bypasses cur.IsCurrent() frame verification; guard with cur.IsCurrent() and use cur.Previous() instead"
			issues = append(issues, issue)
		}
		return true
	})

	if len(issues) == 0 {
		// import present alongside crossing functions but no direct
		// call in this file — still a red flag per the review guide
		issue := ctx.NewIssue("unsafe-previous-realm", imp.Pos(), imp.End())
		issue.Message = "import of " + unsafeRuntimePath + " in a realm with crossing functions (cur realm); use cur.IsCurrent() and cur.Previous() instead"
		issues = append(issues, issue)
	}
	return issues, nil
}

// findImport returns the file's ImportSpec for path, or nil.
func findImport(file *ast.File, path string) *ast.ImportSpec {
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) == path {
			return imp
		}
	}
	return nil
}

// hasCrossingFunc reports whether the file declares a function with a
// `realm`-typed parameter (the gno crossing-function signature).
func hasCrossingFunc(file *ast.File) bool {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil {
			continue
		}
		for _, field := range fn.Type.Params.List {
			if isRealmType(field.Type) {
				return true
			}
		}
	}
	return false
}
