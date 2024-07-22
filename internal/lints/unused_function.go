package lints

import (
	"go/ast"
	"go/parser"
	"go/token"

	tt "github.com/gnoswap-labs/lint/internal/types"
)

// DetectUnusedFunctions detects functions that are declared but never used.
// This rule reports all unused functions except for the following cases:
//  1. The main function: It's considered "used" as it's the entry point of the program.
//  2. The init function: It's used for package initialization and runs without explicit calls.
//  3. Exported functions: Functions starting with a capital letter are excluded as they might be used in other packages.
//
// This rule helps in code cleanup and improves maintainability.
func DetectUnusedFunctions(filename string) ([]tt.Issue, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	declaredFuncs := make(map[string]*ast.FuncDecl)
	calledFuncs := make(map[string]bool)

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			declaredFuncs[x.Name.Name] = x
		case *ast.CallExpr:
			if ident, ok := x.Fun.(*ast.Ident); ok {
				calledFuncs[ident.Name] = true
			}
		}
		return true
	})

	var issues []tt.Issue
	for funcName, funcDecl := range declaredFuncs {
		if !calledFuncs[funcName] && funcName != "main" && funcName != "init" && !ast.IsExported(funcName) {
			issue := tt.Issue{
				Rule:     "unused-function",
				Filename: filename,
				Start:    fset.Position(funcDecl.Pos()),
				End:      fset.Position(funcDecl.End()),
				Message:  "function " + funcName + " is declared but not used",
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}
