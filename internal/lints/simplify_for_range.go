package lints

import (
	"fmt"
	"go/ast"
	"go/token"

	tt "github.com/gnolang/tlin/internal/types"
)

// DetectSimplifiableForLoops detects simplifiable for loops
//
// Normally, in go, a for loop is of the form:
//
//	for i := 0; i < n; i++ {
//	    ^^^^^^  ^^^^^  ^^^
//	      |      |     |
//	      |      |     +--- increment statement (e.g. i++) (3)
//	      |      +--- condition statement (e.g. i < n) (2)
//	      +--- initialize statement (e.g. i := 0) (1)
//
// We can simplify this to:
//
//	for i := range n {
//	   ^^^^^^  ^^^^^
//	     |      |
//	     |      +--- condition statement (e.g. i < n) (2)
//	     +--- initialize statement (e.g. i := 0) (1)
//
// However, this simplification is not always possible.
//
// For example, the following conditions cannot be simplified:
//
// 1. When the initialization statement doesn't start from 0
// 2. When the condition is not a simple comparison with '<'
// 3. When the increment is not a simple i++
// 4. When the loop variable is modified inside the loop body
//
//	for i := 0; i < n; i++ {
//	    if condition { i++ } // Skip an iteration
//	}
//
// 5. When n is not a constant or variable (it must be iterable):
//
//	for i := 0; i < someFunction(); i++ { ... }
//
// 6. When the loop counter is used after the loop:
//
//	for i := 0; i < n; i++ { ... }
//	fmt.Println("Final i:", i) // i is used after the loop
//
// 7. When the intent is to iterate over indices, but range iteration needs the values:
//
//	for i := 0; i < len(slice); i++ {
//	    if i > 0 && slice[i] == slice[i-1] { ... } // Needs index access
//	}
//
// 8. When using multiple variables in the loop:
//
// 9. When the loop needs to break early with a specific index value:
//
//	for i := 0; i < n; i++ {
//	    if condition { break }
//	}
//	// Using the final value of i for something
func DetectSimplifiableForLoops(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	var issues []tt.Issue

	ast.Inspect(node, func(n ast.Node) bool {
		forStmt, ok := n.(*ast.ForStmt)
		if !ok {
			return true
		}

		init, ok := forStmt.Init.(*ast.AssignStmt)
		if !ok || init.Tok != token.DEFINE || len(init.Lhs) != 1 || len(init.Rhs) != 1 {
			return true
		}

		// (2)
		cond, ok := forStmt.Cond.(*ast.BinaryExpr)
		if !ok || cond.Op != token.LSS {
			return true
		}

		// (3)
		post, ok := forStmt.Post.(*ast.IncDecStmt)
		if !ok || post.Tok != token.INC {
			return true
		}

		// check variable name is consistent
		initIdent, ok := init.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}

		condLeftIdent, ok := cond.X.(*ast.Ident)
		if !ok || condLeftIdent.Name != initIdent.Name {
			return true
		}

		postIdent, ok := post.X.(*ast.Ident)
		if !ok || postIdent.Name != initIdent.Name {
			return true
		}

		// check initial value is 0
		initValue, ok := init.Rhs[0].(*ast.BasicLit)
		if !ok || initValue.Value != "0" {
			return true
		}

		// check variable is not modified inside the loop
		var modifiesLoopVar bool
		ast.Inspect(forStmt.Body, func(node ast.Node) bool {
			if assign, ok := node.(*ast.AssignStmt); ok {
				for _, lhs := range assign.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok && ident.Name == initIdent.Name {
						modifiesLoopVar = true
						return false
					}
				}
			}
			if incDec, ok := node.(*ast.IncDecStmt); ok {
				if ident, ok := incDec.X.(*ast.Ident); ok && ident.Name == initIdent.Name {
					modifiesLoopVar = true
					return false
				}
			}
			return true
		})
		if modifiesLoopVar {
			return true
		}

		// check loop range is not a function call
		if _, ok := cond.Y.(*ast.CallExpr); ok {
			return true
		}

		// check loop variable is not used after the loop
		var usedAfterLoop bool
		ast.Inspect(node, func(n ast.Node) bool {
			if n == forStmt {
				return false // skip loop body
			}
			if ident, ok := n.(*ast.Ident); ok && ident.Name == initIdent.Name {
				usedAfterLoop = true
				return false
			}
			return true
		})
		if usedAfterLoop {
			return true
		}

		condRightIdent := cond.Y
		bodyStr := tt.Node2String(forStmt.Body)

		suggestion := fmt.Sprintf("for %s := range %s %s",
			initIdent.Name,
			tt.Node2String(condRightIdent),
			bodyStr,
		)

		pos := fset.Position(forStmt.Pos())
		end := fset.Position(forStmt.End())

		issues = append(issues, tt.Issue{
			Rule:       "simplify_for_range",
			Message:    "counter-based for loop can be simplified to range-based loop",
			Category:   "style",
			Start:      pos,
			End:        end,
			Severity:   severity,
			Suggestion: suggestion,
			Confidence: 0.9,
		})

		return true
	})

	return issues, nil
}
