package interrealm

import (
	"go/ast"
	"go/token"

	tt "github.com/gnolang/tlin/internal/types"
)

// DetectCrossingPosition checks for correct usage of the `crossing()` statement in Gno code
func DetectCrossingPosition(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	// 1. Determine package type (p or r)
	pkgCtx, err := GetPackageContextForFile(filename)
	if err != nil {
		// If we can't determine package type, we'll assume it's not a Gno package
		// and skip the check without error
		return nil, nil
	}

	var issues []tt.Issue

	// 2. Inspect all function declarations
	ast.Inspect(node, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true // Not a function declaration, continue traversal
		}

		// Skip functions without a body
		if funcDecl.Body == nil || len(funcDecl.Body.List) == 0 {
			return true
		}

		// Check if `crossing()` is used in the function
		crossingPos := findCrossingCallPosition(funcDecl.Body)

		// 3. Check for issues based on crossing position and package type
		if crossingPos.exists {
			// Issue 1: `crossing()` is used in p package (forbidden)
			if pkgCtx.Type == TypePackage {
				issue := tt.Issue{
					Rule:     "crossing-in-p-package",
					Filename: filename,
					Start:    fset.Position(crossingPos.pos),
					End:      fset.Position(crossingPos.end),
					Message:  "crossing() statement cannot be used in p packages",
					Severity: tt.SeverityError,
				}
				issues = append(issues, issue)
			}

			// Issue 2: `crossing()` is not the first statement in function
			// TODO: create suggestion to move `crossing()` to the first statement
			if crossingPos.exists && !crossingPos.isFirst {
				issue := tt.Issue{
					Rule:     "crossing-position",
					Filename: filename,
					Start:    fset.Position(crossingPos.pos),
					End:      fset.Position(crossingPos.end),
					Message:  "crossing() must be the first statement in a function body",
					Severity: tt.SeverityError,
				}
				issues = append(issues, issue)
			}

			// Issue 3: `crossing()` is called with arguments (invalid)
			if crossingPos.hasArgs {
				issue := tt.Issue{
					Rule:     "crossing-with-args",
					Filename: filename,
					Start:    fset.Position(crossingPos.pos),
					End:      fset.Position(crossingPos.end),
					Message:  "crossing() should not have any arguments",
					Severity: severity,
				}
				issues = append(issues, issue)
			}
		}

		// Check for public functions in r packages without `crossing()`
		// This is a common mistake: public functions in realm packages should use `crossing()`
		// so they can be called by MsgCall
		if pkgCtx.Type == TypeRealm && isPublicFunction(funcDecl) && !crossingPos.exists {
			// This is a warning rather than an error, as it's a common pattern but not always necessary
			// Public utility functions might not need `crossing()`
			issue := tt.Issue{
				Rule:     "public-function-without-crossing",
				Filename: filename,
				Start:    fset.Position(funcDecl.Pos()),
				End:      fset.Position(funcDecl.Pos()), // Just mark the beginning of the function
				Message:  "public function '" + funcDecl.Name.Name + "' in realm package has no crossing() statement, it cannot be called via MsgCall",
				Severity: tt.SeverityInfo,
				Category: "style", // This is more of a style/design issue than a strict error
			}
			issues = append(issues, issue)
		}

		return true
	})

	return issues, nil
}

// crossingCallInfo contains information about a crossing() call in a function
type crossingCallInfo struct {
	exists  bool      // Whether crossing() exists in the function
	isFirst bool      // Whether crossing() is the first statement
	hasArgs bool      // Whether crossing() has arguments (invalid)
	pos     token.Pos // Start position of the crossing call
	end     token.Pos // End position of the crossing call
}

// findCrossingCallPosition searches for crossing() calls in a function body
// and determines if they are positioned correctly
func findCrossingCallPosition(body *ast.BlockStmt) crossingCallInfo {
	result := crossingCallInfo{
		exists:  false,
		isFirst: false,
		hasArgs: false,
	}

	if len(body.List) == 0 {
		return result
	}

	// Check if the first statement is a crossing() call
	firstStmt := body.List[0]
	if exprStmt, ok := firstStmt.(*ast.ExprStmt); ok {
		if call, ok := exprStmt.X.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "crossing" {
				result.exists = true
				result.isFirst = true
				result.hasArgs = len(call.Args) > 0
				result.pos = call.Pos()
				result.end = call.End()
			}
		}
	}

	// If not found as first statement, look for crossing() elsewhere in the body
	if !result.exists {
		for _, stmt := range body.List {
			if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
				if call, ok := exprStmt.X.(*ast.CallExpr); ok {
					if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "crossing" {
						result.exists = true
						result.isFirst = false // Since we already checked the first statement
						result.hasArgs = len(call.Args) > 0
						result.pos = call.Pos()
						result.end = call.End()
						break
					}
				}
			}
		}
	}

	return result
}

// isPublicFunction checks if a function is public (exported)
// In Go, this means the function name starts with an uppercase letter
func isPublicFunction(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Name == nil {
		return false
	}

	name := funcDecl.Name.Name
	if len(name) == 0 {
		return false
	}

	// Check if first rune is uppercase
	return name[0] >= 'A' && name[0] <= 'Z'
}
