package lint

import (
	"go/ast"
	"go/token"
)

// Define linting rules

// NoEmptyIfRule checks for empty if statements.
type NoEmptyIfRule struct{}

func (r NoEmptyIfRule) Check(fset *token.FileSet, node ast.Node) (bool, string) {
	if ifStmt, ok := node.(*ast.IfStmt); ok {
		if len(ifStmt.Body.List) == 0 {
			return true, "empty if statement"
		}
	}
	return false, ""
}
