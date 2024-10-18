package lints

import (
	"go/ast"
	"go/token"

	tt "github.com/gnolang/tlin/internal/types"
)

func DetectConstErrorDeclaration(
	filename string,
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
			issue := tt.Issue{
				Rule:       "const-error-declaration",
				Filename:   filename,
				Start:      fset.Position(genDecl.Pos()),
				End:        fset.Position(genDecl.End()),
				Message:    "Constant declaration of errors.New() is not allowed",
				Suggestion: "Use var instead of const for error declarations",
				Confidence: 1.0,
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