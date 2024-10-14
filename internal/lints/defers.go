package lints

import (
	"go/ast"
	"go/token"

	tt "github.com/gnolang/tlin/internal/types"
)

type DeferChecker struct {
	fset     *token.FileSet
	filename string
	issues   []tt.Issue
	severity tt.Severity
}

func NewDeferChecker(filename string, fset *token.FileSet, severity tt.Severity) *DeferChecker {
	return &DeferChecker{
		filename: filename,
		fset:     fset,
		severity: severity,
	}
}

func (dc *DeferChecker) Check(node *ast.File) []tt.Issue {
	ast.Inspect(node, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.DeferStmt:
			dc.checkDeferPanic(stmt)
			dc.checkDeferNilFunc(stmt)
			dc.checkReturnInDefer(stmt)
		case *ast.ForStmt, *ast.RangeStmt:
			dc.checkDeferInLoop(stmt)
		}
		return true
	})

	return dc.issues
}

func (dc *DeferChecker) checkDeferPanic(stmt *ast.DeferStmt) {
	ast.Inspect(stmt.Call, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
				dc.addIssue("defer-panic", stmt.Pos(), stmt.End(),
					"Avoid calling panic inside a defer statement",
					"Consider removing the panic call from the defer statement. "+
						"If error handling is needed, use a separate error check before the defer.")
			}
		}
		return true
	})
}

func (dc *DeferChecker) checkDeferNilFunc(stmt *ast.DeferStmt) {
	if ident, ok := stmt.Call.Fun.(*ast.Ident); ok {
		if ident.Name == "nil" || (ident.Obj != nil && ident.Obj.Kind == ast.Var) {
			dc.addIssue("defer-nil-func", stmt.Pos(), stmt.End(),
				"Avoid deferring a potentially nil function",
				"Deferring a nil function will cause a panic at runtime. Ensure the function is not nil before deferring.")
		}
	}
}

func (dc *DeferChecker) checkReturnInDefer(stmt *ast.DeferStmt) {
	ast.Inspect(stmt, func(n ast.Node) bool {
		if funcLit, ok := n.(*ast.FuncLit); ok {
			ast.Inspect(funcLit.Body, func(n ast.Node) bool {
				if _, ok := n.(*ast.ReturnStmt); ok {
					dc.addIssue("return-in-defer", n.Pos(), n.End(),
						"Avoid using return statement inside a defer function",
						"The return statement in a deferred function doesn't affect the returned value of the surrounding function. Consider removing it or refactoring your code.")
					return false
				}
				return true
			})
			// stop inspecting once we've checked the function literal
			return false
		}
		return true
	})
}

func (dc *DeferChecker) checkDeferInLoop(n ast.Node) {
	switch n.(type) {
	case *ast.ForStmt, *ast.RangeStmt:
		ast.Inspect(n, func(inner ast.Node) bool {
			if defer_, ok := inner.(*ast.DeferStmt); ok {
				dc.addIssue("defer-in-loop", defer_.Pos(), defer_.End(),
					"Avoid using defer inside a loop",
					"Consider moving the defer statement outside the loop to avoid potential performance issues.")
			}
			return true
		})
	}
}

func (dc *DeferChecker) addIssue(rule string, start, end token.Pos, message, suggestion string) {
	dc.issues = append(dc.issues, tt.Issue{
		Rule:       rule,
		Filename:   dc.filename,
		Start:      dc.fset.Position(start),
		End:        dc.fset.Position(end),
		Message:    message,
		Suggestion: suggestion,
		Severity:   dc.severity,
	})
}

func DetectDeferIssues(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	checker := NewDeferChecker(filename, fset, severity)
	return checker.Check(node), nil
}
