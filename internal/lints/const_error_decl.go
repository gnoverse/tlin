package lints

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

func init() {
	rule.Register(constErrorDeclarationRule{})
}

type constErrorDeclarationRule struct{}

func (constErrorDeclarationRule) Name() string                 { return "const-error-declaration" }
func (constErrorDeclarationRule) DefaultSeverity() tt.Severity { return tt.SeverityError }

func (constErrorDeclarationRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return DetectConstErrorDeclaration(ctx)
}

// DetectConstErrorDeclaration flags `const x = errors.New(...)` blocks
// and emits a `var`-form suggestion.
func DetectConstErrorDeclaration(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	var issues []tt.Issue

	ast.Inspect(ctx.File, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			return true
		}

		containsErrorsNew := false
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, value := range valueSpec.Values {
				if isErrorNew(value) {
					containsErrorsNew = true
					break
				}
			}
			if containsErrorsNew {
				break
			}
		}

		if containsErrorsNew {
			startPos := ctx.Fset.Position(genDecl.Pos()).Offset
			endPos := ctx.Fset.Position(genDecl.End()).Offset
			origSnippet := ctx.Source[startPos:endPos]

			suggestion := strings.Replace(string(origSnippet), "const", "var", 1)

			issue := ctx.NewIssue("const-error-declaration", genDecl.Pos(), genDecl.End())
			issue.Message = "avoid declaring constant errors"
			issue.Suggestion = suggestion
			issues = append(issues, issue)
		}

		return true
	})

	return issues, nil
}

func isErrorNew(expr ast.Expr) bool {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	selector, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	return selector.Sel.Name == "New" && ident.Name == "errors"
}
