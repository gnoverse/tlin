package lints

import (
	"go/ast"
	"go/token"

	tt "github.com/gnoswap-labs/lint/internal/types"
)

func DetectSliceBoundCheck(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	var issues []tt.Issue

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if len(x.Lhs) == 1 && len(x.Rhs) == 1 {
				if indexExpr, ok := x.Lhs[0].(*ast.IndexExpr); ok {
					if _, ok := indexExpr.X.(*ast.Ident); ok {
						issues = append(issues, tt.Issue{
							Rule:     "slice-bounds-check",
							Filename: filename,
							Start:    fset.Position(x.Pos()),
							End:      fset.Position(x.End()),
							Message:  "Potential slice bounds check failure. Consider using append() instead.",
						})
					}
				}
			}
		}
		return true
	})

	return issues, nil
}