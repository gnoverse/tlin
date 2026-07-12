package lints

import (
	"go/ast"
	"go/token"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

func init() {
	rule.Register(storedRealmRule{})
}

type storedRealmRule struct{}

func (storedRealmRule) Name() string                 { return "stored-realm" }
func (storedRealmRule) DefaultSeverity() tt.Severity { return tt.SeverityError }

func (storedRealmRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return DetectStoredRealm(ctx)
}

// DetectStoredRealm flags `realm`-typed package-level variables and
// struct fields. realm values are ephemeral call-frame data: storing
// one panics at persistence time. Store Previous().Address() or
// PkgPath() strings instead.
func DetectStoredRealm(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	if !isGnoFile(ctx.OriginalPath) {
		return nil, nil
	}

	var issues []tt.Issue

	flag := func(node ast.Node, msg string) {
		issue := ctx.NewIssue("stored-realm", node.Pos(), node.End())
		issue.Message = msg
		issues = append(issues, issue)
	}

	for _, decl := range ctx.File.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			switch s := spec.(type) {
			case *ast.ValueSpec:
				if gen.Tok == token.VAR && containsRealmType(s.Type) {
					flag(s, "realm values are ephemeral and must not be stored in package-level variables; store Previous().Address() or PkgPath() instead")
				}
			case *ast.TypeSpec:
				st, ok := s.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range st.Fields.List {
					if containsRealmType(field.Type) {
						flag(field, "realm values are ephemeral and must not be stored in struct fields; store Address() or PkgPath() strings instead")
					}
				}
			}
		}
	}

	return issues, nil
}

func isRealmType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "realm"
}

// containsRealmType reports whether expr contains a persisted realm value.
func containsRealmType(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return isRealmType(e)
	case *ast.StarExpr:
		return containsRealmType(e.X)
	case *ast.ArrayType:
		return containsRealmType(e.Elt)
	case *ast.MapType:
		return containsRealmType(e.Key) || containsRealmType(e.Value)
	case *ast.ChanType:
		return containsRealmType(e.Value)
	}
	return false
}
