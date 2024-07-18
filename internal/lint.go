package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
)

// LintRule defines the interface for all lint rules.
type LintRule interface {
	Check(filename string) ([]Issue, error)
}

type GolangciLintRule struct{}

func (r *GolangciLintRule) Check(filename string) ([]Issue, error) {
	return runGolangciLint(filename)
}

type UnnecessaryElseRule struct{}

func (r *UnnecessaryElseRule) Check(filename string) ([]Issue, error) {
	engine := &Engine{}
	return engine.detectUnnecessaryElse(filename)
}

// detectUnnecessaryElse detects unnecessary else blocks.
// This rule considers an else block unnecessary if the if block ends with a return statement.
// In such cases, the else block can be removed and the code can be flattened to improve readability.
func (e *Engine) detectUnnecessaryElse(filename string) ([]Issue, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	ast.Inspect(node, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		if ifStmt.Else != nil {
			blockStmt := ifStmt.Body
			if len(blockStmt.List) > 0 {
				lastStmt := blockStmt.List[len(blockStmt.List)-1]
				if _, isReturn := lastStmt.(*ast.ReturnStmt); isReturn {
					issue := Issue{
						Rule:     "unnecessary-else",
						Filename: filename,
						Start:    fset.Position(ifStmt.Else.Pos()),
						End:      fset.Position(ifStmt.Else.End()),
						Message:  "unnecessary else block",
					}
					issues = append(issues, issue)
				}
			}
		}

		return true
	})

	return issues, nil
}

type UnusedFunctionRule struct{}

func (r *UnusedFunctionRule) Check(filename string) ([]Issue, error) {
	engine := &Engine{}
	return engine.detectUnusedFunctions(filename)
}

// detectUnusedFunctions detects functions that are declared but never used.
// This rule reports all unused functions except for the following cases:
//  1. The main function: It's considered "used" as it's the entry point of the program.
//  2. The init function: It's used for package initialization and runs without explicit calls.
//  3. Exported functions: Functions starting with a capital letter are excluded as they might be used in other packages.
//
// This rule helps in code cleanup and improves maintainability.
func (e *Engine) detectUnusedFunctions(filename string) ([]Issue, error) {
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

	var issues []Issue
	for funcName, funcDecl := range declaredFuncs {
		if !calledFuncs[funcName] && funcName != "main" && funcName != "init" && !ast.IsExported(funcName) {
			issue := Issue{
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
