package lints

import (
	"fmt"
	"go/ast"
	"go/token"

	tt "github.com/gnolang/tlin/internal/types"
)

// TODO: Make more precisely.
func DetectSliceBoundCheck(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	var issues []tt.Issue
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.IndexExpr, *ast.SliceExpr:
			ident, ok := getIdentForSliceOrArr(x)
			if !ok {
				return true
			}

			if assignStmt, ok := findAssignmentForIdent(node, ident); ok {
				if callExpr, ok := assignStmt.Rhs[0].(*ast.CallExpr); ok {
					if fun, ok := callExpr.Fun.(*ast.Ident); ok && fun.Name == "make" {
						// Slice created with make, it can be dangerous if the initial length is 0
						// because the slice will be nil and accessing an index will panic
						if len(callExpr.Args) >= 2 {
							if lit, ok := callExpr.Args[1].(*ast.BasicLit); ok && lit.Value == "0" {
								issue := createIssue(x, ident, filename, fset, severity)
								issues = append(issues, issue)
								return true
							}
						}
					}
				}
			}

			if !isWithinSafeContext(node, x) && !isWithinBoundsCheck(node, x, ident) {
				issue := createIssue(x, ident, filename, fset, severity)
				issues = append(issues, issue)
			}
		}
		return true
	})

	return issues, nil
}

func createIssue(node ast.Node, ident *ast.Ident, filename string, fset *token.FileSet, severity tt.Severity) tt.Issue {
	var category, message, suggestion, note string

	switch x := node.(type) {
	case *ast.IndexExpr:
		if isConstantIndex(x.Index) {
			return tt.Issue{}
		}
		category = "index-access"
		message = "potential out of bounds array/slice index access"
		suggestion = fmt.Sprintf("if i < len(%s) { value := %s[i] }", ident.Name, ident.Name)
		note = "always check the length of the array/slice before accessing an index to prevent runtime panics."
	case *ast.SliceExpr:
		category = "slice-expression"
		message = "potential out of bounds slice expression"
		suggestion = fmt.Sprintf("%s = append(%s, newElement)", ident.Name, ident.Name)
		note = "consider using append() for slices to automatically handle capacity and prevent out of bounds errors."
	}

	return tt.Issue{
		Rule:       "slice-bounds-check",
		Category:   category,
		Filename:   filename,
		Start:      fset.Position(node.Pos()),
		End:        fset.Position(node.End()),
		Message:    message,
		Suggestion: suggestion,
		Note:       note,
		Confidence: 0.8,
		Severity:   severity,
	}
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

// isWithinSafeContext checks if the given node is within a "safe" context.
// This function considers the following cases as "safe":
//  1. Inside a for loop with appropriate bounds checking
//  2. Inside a range loop where the loop variable is used as an index
//
// Behavior:
//  1. Traverses the entire AST to find the parent statements containing the given node.
//  2. If a range statement is found, it verifies if the node correctly uses the range variable.
//  3. If a for statement is found, it checks for appropriate length checks.
//  4. If a safe context is found, it immediately stops traversal and returns true.
//
// Notes:
//   - This function may not cover all possible safe usage scenarios.
//   - Complex nested structures or indirect access through function calls may be difficult to analyze accurately.
func isWithinSafeContext(file *ast.File, node ast.Node) bool {
	var safeContext bool
	var rangeIndex *ast.Ident

	ast.Inspect(file, func(n ast.Node) bool {
		if n == node {
			return false
		}
		switch x := n.(type) {
		case *ast.RangeStmt:
			if containsNode(x.Body, node) {
				// Store the range index variable
				if ident, ok := x.Key.(*ast.Ident); ok {
					rangeIndex = ident
				}
				// Check if the node is a safe slice operation using the range index
				if sliceExpr, ok := node.(*ast.SliceExpr); ok {
					if isRangeIndexSlice(sliceExpr, rangeIndex) {
						safeContext = true
						return false
					}
				}
				// inside a range statement, but check if the index expression is the range variable
				if indexExpr, ok := node.(*ast.IndexExpr); ok {
					if ident, ok := indexExpr.X.(*ast.Ident); ok {
						// accessing a different slice/array than the range variable is not safe
						safeContext = (ident.Name == rangeIndex.Name)
					}
				}
				return false
			}
		case *ast.ForStmt:
			if isForWithLenCheck(x) && containsNode(x.Body, node) {
				safeContext = true
				return false
			}
		}
		return true
	})
	return safeContext
}

// isForWithLenCheck checks if a for statement has a length check condition.
func isForWithLenCheck(forStmt *ast.ForStmt) bool {
	if cond, ok := forStmt.Cond.(*ast.BinaryExpr); ok {
		if isBinaryExprLenCheck(cond) {
			return true
		}
	}
	return false
}

// isConstantIndex determines if an expression is a constant index.
func isConstantIndex(expr ast.Expr) bool {
	switch x := expr.(type) {
	case *ast.BasicLit:
		return x.Kind == token.INT
	case *ast.Ident:
		return x.Obj != nil && x.Obj.Kind == ast.Con
	}
	return false
}

// isBinaryExprLenCheck checks if a binary expression is a length check.
func isBinaryExprLenCheck(expr *ast.BinaryExpr) bool {
	if expr.Op == token.LSS || expr.Op == token.LEQ {
		if call, ok := expr.Y.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok {
				return ident.Name == "len"
			}
		}
	}
	return false
}

// findAssignmentForIdent finds the assignment statement for a given identifier.
func findAssignmentForIdent(file *ast.File, ident *ast.Ident) (*ast.AssignStmt, bool) {
	var assignStmt *ast.AssignStmt
	var found bool

	ast.Inspect(file, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok {
			for _, lhs := range assign.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && id.Name == ident.Name {
					assignStmt = assign
					found = true
					return false
				}
			}
		}
		return true
	})

	return assignStmt, found
}

// isRangeIndexSlice checks if a node is a safe slice operation within a range loop.
func isRangeIndexSlice(node ast.Node, rangeIndex *ast.Ident) bool {
	switch expr := node.(type) {
	case *ast.SliceExpr:
		return isRangeIndexInSlice(expr, rangeIndex)
	case *ast.CallExpr:
		// Check for append(arr[:i], arr[i+1:]...) pattern
		if fun, ok := expr.Fun.(*ast.Ident); ok && fun.Name == "append" {
			for _, arg := range expr.Args {
				if slice, ok := arg.(*ast.SliceExpr); ok {
					if isRangeIndexInSlice(slice, rangeIndex) {
						return true
					}
				}
			}
		}
	}
	return false
}

// isRangeIndexInSlice checks if a slice expression uses the range loop index.
func isRangeIndexInSlice(sliceExpr *ast.SliceExpr, rangeIndex *ast.Ident) bool {
	// Check low bound
	if isRangeIndexExpr(sliceExpr.Low, rangeIndex) {
		return true
	}

	// Check high bound
	if isRangeIndexExpr(sliceExpr.High, rangeIndex) {
		return true
	}

	// Check for Ellipsis
	if sliceExpr.Slice3 {
		return isRangeIndexExpr(sliceExpr.Max, rangeIndex)
	}

	return false
}

// isRangeIndexExpr recursively checks if an expression uses the range loop index.
func isRangeIndexExpr(expr ast.Expr, rangeIndex *ast.Ident) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name == rangeIndex.Name
	case *ast.BinaryExpr:
		return isRangeIndexExpr(e.X, rangeIndex) || isRangeIndexExpr(e.Y, rangeIndex)
	}
	return false
}
