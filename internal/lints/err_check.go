package lints

import (
	"go/ast"
	"go/token"
	"strings"

	tt "github.com/gnolang/tlin/internal/types"
)

func DetectErrCheck(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	v := &errCheckVisitor{
		filename:    filename,
		fset:        fset,
		severity:    severity,
		imports:     make(map[string]bool),
		errorVars:   make(map[string]bool),
		errorChecks: make(map[string]bool),
	}

	// 1st pass: collect error variable declarations and checks
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			v.collectErrorVars(x)
		case *ast.IfStmt:
			v.collectErrorChecks(x)
		}
		return true
	})

	// 2nd pass: check for unchecked errors
	ast.Walk(v, node)
	return v.issues, nil
}

type errCheckVisitor struct {
	filename    string
	fset        *token.FileSet
	severity    tt.Severity
	issues      []tt.Issue
	imports     map[string]bool
	errorVars   map[string]bool // tracks declared error variables
	errorChecks map[string]bool // tracks which error vars have been checked
	currentFunc *ast.FuncDecl   // tracks current function being analyzed
}

func (v *errCheckVisitor) collectErrorVars(assign *ast.AssignStmt) {
	for _, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			if ident.Name == "err" || strings.HasSuffix(ident.Name, "Error") {
				v.errorVars[ident.Name] = true
			}
		}
	}
}

func (v *errCheckVisitor) collectErrorChecks(ifStmt *ast.IfStmt) {
	// check `err != nil` pattern
	if binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr); ok {
		if ident, ok := binExpr.X.(*ast.Ident); ok {
			if v.errorVars[ident.Name] {
				v.errorChecks[ident.Name] = true
			}
		}
	}
}

func (v *errCheckVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return v
	}

	switch n := node.(type) {
	case *ast.FuncDecl:
		v.currentFunc = n
		defer func() { v.currentFunc = nil }()
	case *ast.AssignStmt:
		v.checkAssignment(n)
	case *ast.ExprStmt:
		v.checkExprStmt(n)
	}

	return v
}

func (v *errCheckVisitor) checkAssignment(assign *ast.AssignStmt) {
	// multi-return cases
	if len(assign.Rhs) == 1 && len(assign.Lhs) > 1 {
		if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
			v.checkMultiValueAssign(assign.Lhs, call)
		}
		return
	}

	// single assignments
	for i, rhs := range assign.Rhs {
		if i >= len(assign.Lhs) {
			continue
		}
		if call, ok := rhs.(*ast.CallExpr); ok && v.isLikelyErrorReturn(call) {
			lhs := assign.Lhs[i]
			if isBlankIdentifier(lhs) {
				v.addIssue(
					assign.Pos(), assign.End(),
					"error return value is ignored with blank identifier",
					"consider handling the error value",
				)
			}
		}
	}
}

func (v *errCheckVisitor) checkMultiValueAssign(lhs []ast.Expr, call *ast.CallExpr) {
	if !v.isLikelyErrorReturn(call) {
		return
	}

	// check if the last return value (typically error) is properly handled
	if len(lhs) >= 2 {
		lastExpr := lhs[len(lhs)-1]
		if isBlankIdentifier(lastExpr) {
			v.addIssue(
				call.Pos(), call.End(),
				"error return value is ignored with blank identifier",
				"consider handling the error value",
			)
		}
	}
}

func (v *errCheckVisitor) checkExprStmt(expr *ast.ExprStmt) {
	if call, ok := expr.X.(*ast.CallExpr); ok && v.isLikelyErrorReturn(call) {
		// ignore certain method calls that are commonly used without error checking
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "Close" || sel.Sel.Name == "Print" || sel.Sel.Name == "Println" {
				return
			}
		}

		v.addIssue(
			expr.Pos(), expr.End(),
			"error-returning function call's result is ignored",
			"add error handling or explicitly assign the error",
		)
	}
}

func (v *errCheckVisitor) isLikelyErrorReturn(call *ast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		// common functions that don't return errors
		switch fun.Name {
		case "print", "println", "len", "cap", "append", "recover", "panic":
			return false
		}
	case *ast.SelectorExpr:
		// common error-returning methods
		switch fun.Sel.Name {
		case "Error", "String", "GoString":
			return false
		case "Close", "Write", "Read", "Scan", "Parse", "Open", "Create",
			"Delete", "Update", "Exec", "Query":
			return true
		}
	}
	return true
}

func isBlankIdentifier(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "_"
}

func (v *errCheckVisitor) addIssue(start, end token.Pos, msg, note string) {
	issue := tt.Issue{
		Rule:     "error-check",
		Filename: v.filename,
		Start:    v.fset.Position(start),
		End:      v.fset.Position(end),
		Message:  msg,
		Note:     note,
		Severity: v.severity,
	}
	v.issues = append(v.issues, issue)
}
