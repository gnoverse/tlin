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
		case *ast.IndexExpr, *ast.SliceExpr:
			ident, ok := getIdentForSliceOrArr(x)
			if !ok {
				return true
			}

			// Check if this access is within an if statement that checks bounds
			if !isWithinBoundsCheck(node, x, ident) {
				issues = append(issues, tt.Issue{
					Rule:     "slice-bounds-check",
					Filename: filename,
					Start:    fset.Position(x.Pos()),
					End:      fset.Position(x.End()),
					Message:  "Potential slice bounds check failure. Consider checking length before access or using append().",
				})
			}
		}
		return true
	})

	return issues, nil
}

// getIdentForSliceOrArr checks if the node is within an if statement
// that performs a bounds check.
func getIdentForSliceOrArr(node ast.Node) (*ast.Ident, bool) {
	switch n := node.(type) {
	case *ast.IndexExpr:
		if ident, ok := n.X.(*ast.Ident); ok {
			return ident, true
		}
	case *ast.SliceExpr:
		if ident, ok := n.X.(*ast.Ident); ok {
			return ident, true
		}
	}
	return nil, false
}

// isWithinBoundsCheck checks if the node is within an if statement that performs a bounds check.
func isWithinBoundsCheck(file *ast.File, node ast.Node, ident *ast.Ident) bool {
	var ifStmt *ast.IfStmt
	var found bool

	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		if x, ok := n.(*ast.IfStmt); ok && containsNode(x, node) {
			ifStmt = x
			found = true
		}
		return true
	})

	if ifStmt == nil {
		return false
	}
	return isIfCapCheck(ifStmt, ident)
}

// isIfCapCheck checks if the if statement condition is a capacity or length check.
func isIfCapCheck(ifStmt *ast.IfStmt, ident *ast.Ident) bool {
	if binaryExpr, ok := ifStmt.Cond.(*ast.BinaryExpr); ok {
		return binExprHasLenCapCall(binaryExpr, ident)
	}
	return false
}

// binExprHasLenCapCall checks if a binary expression contains a length or capacity call.
func binExprHasLenCapCall(bin *ast.BinaryExpr, ident *ast.Ident) bool {
	if call, ok := bin.X.(*ast.CallExpr); ok {
		return isCapOrLenCallWithIdent(call, ident)
	}
	if call, ok := bin.Y.(*ast.CallExpr); ok {
		return isCapOrLenCallWithIdent(call, ident)
	}
	return false
}

// isCapOrLenCallWithIdent checks if a call expression is a len or cap call with the identifier.
func isCapOrLenCallWithIdent(call *ast.CallExpr, ident *ast.Ident) bool {
	if fun, ok := call.Fun.(*ast.Ident); ok {
		if (fun.Name == "len" || fun.Name == "cap") && len(call.Args) == 1 {
			arg, ok := call.Args[0].(*ast.Ident)
			return ok && arg.Name == ident.Name
		}
	}
	return false
}
