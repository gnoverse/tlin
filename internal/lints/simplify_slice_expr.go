package lints

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	tt "github.com/gnolang/tlin/internal/types"
)

const (
	baseMessageUnnecessaryLen = "unnecessary use of len() in slice expression, can be simplified"
	ruleName                  = "simplify-slice-range"

	msgTemplateNoIndex = "%s\nin this case, `%s[:len(%s)]` is equivalent to `%s[:]`. " +
		"the full length of the slice is already implied when omitting both start and end indices."

	msgTemplateWithLiteral = "%s\nhere, `%s[%s:len(%s)]` can be simplified to `%s[%s:]`. " +
		"when slicing to the end of a slice, using len() is unnecessary."

	msgTemplateWithIdent = "%s\nin this instance, `%s[%s:len(%s)]` can be written as `%s[%s:]`. " +
		"the len() function is redundant when slicing to the end, regardless of the start index."
)

// DetectUnnecessarySliceLength detects unnecessary len() calls in slice expressions.
// Example: a[:len(a)] -> a[:]
func DetectUnnecessarySliceLength(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	var issues []tt.Issue

	ast.Inspect(node, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			issues = append(issues, processAssignStmt(stmt, filename, fset, severity)...)
		case *ast.ValueSpec:
			// XXX: I'm not sure about we need to process this case.
			issues = append(issues, processValueSpec(stmt, filename, fset, severity)...)
		}
		return true
	})

	return issues, nil
}

// processAssignStmt processes assignment statements to find unnecessary len() calls.
func processAssignStmt(stmt *ast.AssignStmt, filename string, fset *token.FileSet, severity tt.Severity) []tt.Issue {
	var issues []tt.Issue

	for i, rhs := range stmt.Rhs {
		sliceExpr, ok := rhs.(*ast.SliceExpr)
		if !ok {
			continue
		}

		issue, found := checkUnnecessaryLenInSliceExpr(sliceExpr, filename, fset, severity)
		if !found {
			continue
		}

		// Only process if there's a corresponding left-hand side
		if i < len(stmt.Lhs) {
			lhs, ok := stmt.Lhs[i].(*ast.Ident)
			if ok {
				issue.Suggestion = formatSuggestion(lhs.Name, getOperator(lhs.Name, stmt.Tok), issue.Suggestion)
			}
		}

		issues = append(issues, issue)
	}

	return issues
}

// processValueSpec processes variable declarations to find unnecessary len() calls.
func processValueSpec(stmt *ast.ValueSpec, filename string, fset *token.FileSet, severity tt.Severity) []tt.Issue {
	var issues []tt.Issue

	for i, value := range stmt.Values {
		sliceExpr, ok := value.(*ast.SliceExpr)
		if !ok {
			continue
		}

		issue, found := checkUnnecessaryLenInSliceExpr(sliceExpr, filename, fset, severity)
		if !found {
			continue
		}

		// Only process if there's a corresponding name
		if i < len(stmt.Names) {
			name := stmt.Names[i].Name
			issue.Suggestion = formatSuggestion(name, getOperator(name, token.DEFINE), issue.Suggestion)
		}

		issues = append(issues, issue)
	}

	return issues
}

// getOperator returns the appropriate operator based on the identifier and token.
func getOperator(identName string, tok token.Token) string {
	// blank identifier always uses assignment operator
	if identName == blankIdentifier {
		return opAssign
	}

	if tok == token.DEFINE {
		return opDefine
	}

	return opAssign
}

// formatSuggestion formats the suggestion string with the variable name and operator.
func formatSuggestion(name string, operator string, suggestion string) string {
	return fmt.Sprintf("%s %s %s", name, operator, suggestion)
}

// checkUnnecessaryLenInSliceExpr checks a slice expression for unnecessary len() calls.
func checkUnnecessaryLenInSliceExpr(sliceExpr *ast.SliceExpr, filename string, fset *token.FileSet, severity tt.Severity) (tt.Issue, bool) {
	// Skip 3-index slices as they always require the 2nd and 3rd index
	if sliceExpr.Max != nil {
		return tt.Issue{}, false
	}

	if !isSliceWithLenCall(sliceExpr) {
		return tt.Issue{}, false
	}

	sliceIdent := sliceExpr.X.(*ast.Ident)
	suggestion, detailedMessage := createSuggestionAndMessage(sliceExpr, sliceIdent)

	return tt.Issue{
		Rule:       ruleName,
		Filename:   filename,
		Start:      fset.Position(sliceExpr.Pos()),
		End:        fset.Position(sliceExpr.End()),
		Message:    baseMessageUnnecessaryLen,
		Suggestion: suggestion,
		Note:       detailedMessage,
		Severity:   severity,
	}, true
}

// isSliceWithLenCall checks if the slice expression has unnecessary len() call.
func isSliceWithLenCall(sliceExpr *ast.SliceExpr) bool {
	// check if the array/slice object is a single identifier
	sliceIdent, ok := sliceExpr.X.(*ast.Ident)
	if !ok || sliceIdent.Obj == nil {
		return false
	}

	// check if the high expression is a function call with a single argument
	call, ok := sliceExpr.High.(*ast.CallExpr)
	if !ok || !isValidLenCall(call) {
		return false
	}

	// check if the function is "len" and not locally defined
	lenIdent, ok := call.Fun.(*ast.Ident)
	if !ok || lenIdent.Name != "len" || lenIdent.Obj != nil {
		return false
	}

	// check if the len argument is the same as the array/slice object
	arg, ok := call.Args[0].(*ast.Ident)
	if !ok || arg.Obj != sliceIdent.Obj {
		return false
	}

	return true
}

// isValidLenCall checks if a function call is a valid len() call.
func isValidLenCall(call *ast.CallExpr) bool {
	return len(call.Args) == 1 && !call.Ellipsis.IsValid()
}

// createSuggestionAndMessage creates the suggestion and detailed message for the issue.
func createSuggestionAndMessage(sliceExpr *ast.SliceExpr, sliceIdent *ast.Ident) (string, string) {
	var suggestion, detailedMessage string

	if sliceExpr.Low == nil {
		suggestion = fmt.Sprintf("%s[:]", sliceIdent.Name)
		detailedMessage = fmt.Sprintf(
			msgTemplateNoIndex,
			baseMessageUnnecessaryLen, sliceIdent.Name, sliceIdent.Name, sliceIdent.Name)
	} else if basicLit, ok := sliceExpr.Low.(*ast.BasicLit); ok {
		suggestion = fmt.Sprintf("%s[%s:]", sliceIdent.Name, basicLit.Value)
		detailedMessage = fmt.Sprintf(
			msgTemplateWithLiteral,
			baseMessageUnnecessaryLen, sliceIdent.Name, basicLit.Value, sliceIdent.Name, sliceIdent.Name, basicLit.Value)
	} else if lowIdent, ok := sliceExpr.Low.(*ast.Ident); ok {
		suggestion = fmt.Sprintf("%s[%s:]", sliceIdent.Name, lowIdent.Name)
		detailedMessage = fmt.Sprintf(
			msgTemplateWithIdent,
			baseMessageUnnecessaryLen, sliceIdent.Name, lowIdent.Name, sliceIdent.Name, sliceIdent.Name, lowIdent.Name)
	} else if binaryExpr, ok := sliceExpr.Low.(*ast.BinaryExpr); ok {
		// Handle BinaryExpr type of Low index
		start := fmt.Sprintf("%s[%s:", sliceIdent.Name, formatBinaryExpr(binaryExpr))
		suggestion = start + "]"
		detailedMessage = fmt.Sprintf(
			msgTemplateWithIdent,
			baseMessageUnnecessaryLen, sliceIdent.Name, formatBinaryExpr(binaryExpr), sliceIdent.Name, sliceIdent.Name, formatBinaryExpr(binaryExpr))
	}

	return suggestion, detailedMessage
}

// formatBinaryExpr converts BinaryExpr to a string.
func formatBinaryExpr(expr *ast.BinaryExpr) string {
	left := formatExpr(expr.X)
	right := formatExpr(expr.Y)
	return fmt.Sprintf("%s %s %s", left, expr.Op, right)
}

// formatExpr converts Expr to a string.
func formatExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.BasicLit:
		return e.Value
	case *ast.CallExpr:
		if ident, ok := e.Fun.(*ast.Ident); ok {
			args := make([]string, len(e.Args))
			for i, arg := range e.Args {
				args[i] = formatExpr(arg)
			}
			return fmt.Sprintf("%s(%s)", ident.Name, strings.Join(args, ", "))
		}
	}
	return ""
}
