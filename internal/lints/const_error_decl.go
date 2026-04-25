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
	return DetectConstErrorDeclaration(ctx.WorkingPath, ctx.Source, ctx.File, ctx.Fset, ctx.Severity)
}

// DetectConstErrorDeclaration flags `const x = errors.New(...)` blocks
// and emits a `var`-form suggestion. src is the raw bytes of the
// file used to slice the original snippet for the suggestion.
func DetectConstErrorDeclaration(
	filename string,
	src []byte,
	node *ast.File,
	fset *token.FileSet,
	severity tt.Severity,
) ([]tt.Issue, error) {
	var issues []tt.Issue

	ast.Inspect(node, func(n ast.Node) bool {
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
			startPos := fset.Position(genDecl.Pos()).Offset
			endPos := fset.Position(genDecl.End()).Offset
			origSnippet := src[startPos:endPos]

			suggestion := strings.Replace(string(origSnippet), "const", "var", 1)

			issue := tt.Issue{
				Rule:       "const-error-declaration",
				Filename:   filename,
				Start:      fset.Position(genDecl.Pos()),
				End:        fset.Position(genDecl.End()),
				Message:    "avoid declaring constant errors",
				Suggestion: suggestion,
				Severity:   severity,
			}
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
