package lints

import (
	"fmt"
	"go/ast"
	"go/token"

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
		sliceExpr, ok := n.(*ast.SliceExpr)
		if !ok {
			return true
		}

		if issue, found := checkUnnecessaryLenInSliceExpr(sliceExpr, filename, fset, severity); found {
			issues = append(issues, issue)
		}

		return true
	})

	return issues, nil
}

// checkUnnecessaryLenInSliceExpr checks a slice expression for unnecessary len() calls.
func checkUnnecessaryLenInSliceExpr(sliceExpr *ast.SliceExpr, filename string, fset *token.FileSet, severity tt.Severity) (tt.Issue, bool) {
	// skip 3-index slices as they always require the 2nd and 3rd index
	if sliceExpr.Max != nil {
		return tt.Issue{}, false
	}

	// check if the array/slice object is a single identifier
	sliceIdent, ok := sliceExpr.X.(*ast.Ident)
	if !ok || sliceIdent.Obj == nil {
		return tt.Issue{}, false
	}

	// check if the high expression is a function call with a single argument
	call, ok := sliceExpr.High.(*ast.CallExpr)
	if !ok || !isValidLenCall(call) {
		return tt.Issue{}, false
	}

	// check if the function is "len" and not locally defined
	lenIdent, ok := call.Fun.(*ast.Ident)
	if !ok || lenIdent.Name != "len" || lenIdent.Obj != nil {
		return tt.Issue{}, false
	}

	// check if the len argument is the same as the array/slice object
	arg, ok := call.Args[0].(*ast.Ident)
	if !ok || arg.Obj != sliceIdent.Obj {
		return tt.Issue{}, false
	}

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
	}

	return suggestion, detailedMessage
}
