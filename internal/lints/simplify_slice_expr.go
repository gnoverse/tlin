package lints

import (
	"fmt"
	"go/ast"
	"go/token"

	tt "github.com/gnoswap-labs/tlin/internal/types"
)

func DetectUnnecessarySliceLength(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	var issues []tt.Issue
	ast.Inspect(node, func(n ast.Node) bool {
		sliceExpr, ok := n.(*ast.SliceExpr)
		if !ok {
			return true
		}

		if callExpr, ok := sliceExpr.High.(*ast.CallExpr); ok {
			if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "len" {
				if arg, ok := callExpr.Args[0].(*ast.Ident); ok {
					if sliceExpr.X.(*ast.Ident).Name == arg.Name {
						var suggestion, detailedMessage string
						baseMessage := "unnecessary use of len() in slice expression, can be simplified"

						if sliceExpr.Low == nil {
							suggestion = fmt.Sprintf("%s[:]", arg.Name)
							detailedMessage = fmt.Sprintf(
								"%s\nIn this case, `%s[:len(%s)]` is equivalent to `%s[:]`. "+
									"The full length of the slice is already implied when omitting both start and end indices.",
								baseMessage, arg.Name, arg.Name, arg.Name)
						} else if basicLit, ok := sliceExpr.Low.(*ast.BasicLit); ok {
							suggestion = fmt.Sprintf("%s[%s:]", arg.Name, basicLit.Value)
							detailedMessage = fmt.Sprintf("%s\nHere, `%s[%s:len(%s)]` can be simplified to `%s[%s:]`. "+
								"When slicing to the end of a slice, using len() is unnecessary.",
								baseMessage, arg.Name, basicLit.Value, arg.Name, arg.Name, basicLit.Value)
						} else if lowIdent, ok := sliceExpr.Low.(*ast.Ident); ok {
							suggestion = fmt.Sprintf("%s[%s:]", arg.Name, lowIdent.Name)
							detailedMessage = fmt.Sprintf("%s\nIn this instance, `%s[%s:len(%s)]` can be written as `%s[%s:]`. "+
								"The len() function is redundant when slicing to the end, regardless of the start index.",
								baseMessage, arg.Name, lowIdent.Name, arg.Name, arg.Name, lowIdent.Name)
						}

						issue := tt.Issue{
							Rule:       "simplify-slice-range",
							Filename:   filename,
							Start:      fset.Position(sliceExpr.Pos()),
							End:        fset.Position(sliceExpr.End()),
							Message:    baseMessage,
							Suggestion: suggestion,
							Note:       detailedMessage,
						}
						issues = append(issues, issue)
					}
				}
			}
		}

		return true
	})

	return issues, nil
}
