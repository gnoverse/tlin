package lints

import (
	"go/ast"
	"go/token"
	"slices"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

func init() {
	rule.Register(exportedMutablePointerRule{})
}

type exportedMutablePointerRule struct{}

func (exportedMutablePointerRule) Name() string                 { return "exported-mutable-pointer" }
func (exportedMutablePointerRule) DefaultSeverity() tt.Severity { return tt.SeverityWarning }

func (exportedMutablePointerRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return DetectExportedMutablePointer(ctx)
}

// DetectExportedMutablePointer flags exported functions that return a
// pointer to package-level state. A returned pointer lets any caller
// invoke mutation methods (or write fields) on realm-persisted state
// with the realm's own authority — readonly taint does not block
// method dispatch. Return copies or read-only accessors instead.
func DetectExportedMutablePointer(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	// realm-authority leak is a gno concern; exported pointer getters
	// are a legitimate pattern in plain Go
	if !isGnoFile(ctx.OriginalPath) {
		return nil, nil
	}

	var issues []tt.Issue
	var pkgVars map[string]bool // built lazily on the first qualifying function
	for _, decl := range ctx.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() || fn.Body == nil || fn.Type.Results == nil {
			continue
		}
		pointerResults := pointerResultFlags(fn.Type.Results)
		if !slices.Contains(pointerResults, true) {
			continue
		}
		if pkgVars == nil {
			pkgVars = packageLevelVars(ctx.File)
			if len(pkgVars) == 0 {
				return nil, nil
			}
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			// nested function literals have their own result types;
			// only inspect return statements of fn itself
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}
			ret, ok := n.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			for i, expr := range ret.Results {
				if i >= len(pointerResults) || !pointerResults[i] {
					continue
				}
				if name, leaks := leaksPackageVar(expr, pkgVars); leaks {
					issue := ctx.NewIssue("exported-mutable-pointer", expr.Pos(), expr.End())
					issue.Message = "exported function returns a pointer to package-level state '" + name + "'; callers can mutate realm state through it — return a copy or read-only accessor instead"
					issues = append(issues, issue)
				}
			}
			return true
		})
	}
	return issues, nil
}

// packageLevelVars returns the names of all package-level `var`
// declarations in the file.
func packageLevelVars(file *ast.File) map[string]bool {
	vars := map[string]bool{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				if name.Name != blankIdentifier {
					vars[name.Name] = true
				}
			}
		}
	}
	return vars
}

// pointerResultFlags reports, per flattened result position, whether
// the declared type is a pointer.
func pointerResultFlags(results *ast.FieldList) []bool {
	var flags []bool
	for _, field := range results.List {
		_, isPtr := field.Type.(*ast.StarExpr)
		for range max(len(field.Names), 1) {
			flags = append(flags, isPtr)
		}
	}
	return flags
}

// leaksPackageVar reports whether expr hands out a pointer rooted in a
// package-level var: the var itself, its address, or (the address of)
// one of its fields.
func leaksPackageVar(expr ast.Expr, pkgVars map[string]bool) (string, bool) {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}
	for {
		switch e := expr.(type) {
		case *ast.Ident:
			return e.Name, pkgVars[e.Name]
		case *ast.SelectorExpr:
			expr = e.X
		default:
			return "", false
		}
	}
}
