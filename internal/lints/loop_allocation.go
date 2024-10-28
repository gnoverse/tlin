package lints

import (
	"go/ast"
	"go/token"

	tt "github.com/gnolang/tlin/internal/types"
)

func DetectLoopAllocation(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	var issues []tt.Issue

	ast.Inspect(node, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.RangeStmt, *ast.ForStmt:
			ast.Inspect(node, func(inner ast.Node) bool {
				switch innerNode := inner.(type) {
				case *ast.CallExpr:
					if isAllocationFunction(innerNode) {
						issues = append(issues, tt.Issue{
							Rule:     "loop-allocation",
							Message:  "potential unnecessary allocation inside loop",
							Start:    fset.Position(innerNode.Pos()),
							End:      fset.Position(innerNode.End()),
							Severity: severity,
						})
					}
					// case *ast.AssignStmt:
					// 	if innerNode.Tok == token.DEFINE {
					// 		issues = append(issues, tt.Issue{
					// 			Message: "Variable declaration inside loop may cause unnecessary allocation",
					// 			Start: fset.Position(innerNode.Pos()),
					// 			End: fset.Position(innerNode.End()),
					// 		})
					// 	}
				}
				return true
			})
		}
		return true
	})
	return issues, nil
}

func isAllocationFunction(call *ast.CallExpr) bool {
	if ident, ok := call.Fun.(*ast.Ident); ok {
		return ident.Name == "make" || ident.Name == "new"
	}
	return false
}
