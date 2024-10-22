package lints

import (
	"go/ast"
	"go/token"
	"os"
	"strings"

	tt "github.com/gnolang/tlin/internal/types"
)

// This 

func DetectConstErrorDeclaration(
	filename string,
	node *ast.File,
	fset *token.FileSet,
	severity tt.Severity,
) ([]tt.Issue, error) {
	var issues []tt.Issue

	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

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
				Message:    "Avoid declaring constant errors",
				Suggestion: suggestion,
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
