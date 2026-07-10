package lints

import (
	"context"
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

func (r exportedMutablePointerRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return DetectExportedMutablePointer(ctx)
}

// DetectExportedMutablePointer applies exported-mutable-pointer to one file.
func DetectExportedMutablePointer(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return exportedMutablePointerRule{}.CheckPackage(context.Background(), ctx.SinglePackage())
}

// CheckPackage flags exported pointer returns rooted in package-level
// Gno state. Package scope is required because state and accessors are
// commonly split across sibling .gno files.
func (exportedMutablePointerRule) CheckPackage(_ context.Context, pctx *rule.PackageContext) ([]tt.Issue, error) {
	return checkGnoPackage(pctx, scanExportedMutablePointer)
}

func scanExportedMutablePointer(pctx *rule.PackageContext, files []gnoFile) []tt.Issue {
	pkgVars := map[string]bool{}
	for _, f := range files {
		collectPackageLevelVars(f.ast, pkgVars)
	}
	if len(pkgVars) == 0 {
		return nil
	}

	var issues []tt.Issue
	for _, f := range files {
		for _, decl := range f.ast.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() || fn.Body == nil || fn.Type.Results == nil {
				continue
			}
			pointerResults, namedResults := resultPointerInfo(fn.Type.Results)
			if !slices.Contains(pointerResults, true) {
				continue
			}

			// ponytail: alias tracking is monotonic; add reassignment
			// kills if false positives show up in practice.
			var tainted map[string]bool
			isTainted := func(name string) bool { return pkgVars[name] || tainted[name] }

			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.FuncLit:
					return false
				case *ast.AssignStmt:
					if len(n.Lhs) != len(n.Rhs) {
						return true
					}
					for i, lhs := range n.Lhs {
						id, ok := lhs.(*ast.Ident)
						if !ok {
							continue
						}
						if _, leaks := leaksVia(n.Rhs[i], isTainted); leaks {
							if tainted == nil {
								tainted = map[string]bool{}
							}
							tainted[id.Name] = true
						}
					}
				case *ast.ReturnStmt:
					if len(n.Results) == 0 {
						for _, name := range namedResults {
							if isTainted(name) {
								issues = append(issues, packageIssue(pctx, f, "exported-mutable-pointer", n.Pos(), n.End(), leakMessage(name)))
							}
						}
						return true
					}
					for i, expr := range n.Results {
						if i >= len(pointerResults) || !pointerResults[i] {
							continue
						}
						if name, leaks := leaksVia(expr, isTainted); leaks {
							issues = append(issues, packageIssue(pctx, f, "exported-mutable-pointer", expr.Pos(), expr.End(), leakMessage(name)))
						}
					}
				}
				return true
			})
		}
	}
	return issues
}

func leakMessage(name string) string {
	return "exported function returns a pointer rooted in package-level state '" + name + "'; callers can mutate realm state through it — return a copy or read-only accessor instead"
}

// collectPackageLevelVars adds file's package-level vars to vars.
func collectPackageLevelVars(file *ast.File, vars map[string]bool) {
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
}

// resultPointerInfo returns pointer flags and pointer-typed named results.
func resultPointerInfo(results *ast.FieldList) (flags []bool, namedPointers []string) {
	for _, field := range results.List {
		_, isPtr := field.Type.(*ast.StarExpr)
		for range max(len(field.Names), 1) {
			flags = append(flags, isPtr)
		}
		if isPtr {
			for _, name := range field.Names {
				namedPointers = append(namedPointers, name.Name)
			}
		}
	}
	return flags, namedPointers
}

// leaksVia reports whether expr is rooted in a tainted name.
func leaksVia(expr ast.Expr, isTainted func(string) bool) (string, bool) {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}
	for {
		switch e := expr.(type) {
		case *ast.Ident:
			return e.Name, isTainted(e.Name)
		case *ast.SelectorExpr:
			expr = e.X
		default:
			return "", false
		}
	}
}
