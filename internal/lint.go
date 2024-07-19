package internal

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
)

/*
* Implement each lint rule as a separate struct
 */

// LintRule defines the interface for all lint rules.
type LintRule interface {
	// Check runs the lint rule on the given file and returns a slice of Issues.
	Check(filename string) ([]Issue, error)
}

type GolangciLintRule struct{}

func (r *GolangciLintRule) Check(filename string) ([]Issue, error) {
	return runGolangciLint(filename)
}

type golangciOutput struct {
	Issues []struct {
		FromLinter string `json:"FromLinter"`
		Text       string `json:"Text"`
		Pos        struct {
			Filename string `json:"Filename"`
			Line     int    `json:"Line"`
			Column   int    `json:"Column"`
		} `json:"Pos"`
	} `json:"Issues"`
}

func runGolangciLint(filename string) ([]Issue, error) {
	cmd := exec.Command("golangci-lint", "run", "--disable=gosimple", "--out-format=json", filename)
	output, _ := cmd.CombinedOutput()

	var golangciResult golangciOutput
	if err := json.Unmarshal(output, &golangciResult); err != nil {
		return nil, fmt.Errorf("error unmarshaling golangci-lint output: %w", err)
	}

	var issues []Issue
	for _, gi := range golangciResult.Issues {
		issues = append(issues, Issue{
			Rule:     gi.FromLinter,
			Filename: gi.Pos.Filename, // Use the filename from golangci-lint output
			Start:    token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1},
			Message:  gi.Text,
		})
	}

	return issues, nil
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

type SimplifySliceExprRule struct{}

func (r *SimplifySliceExprRule) Check(filename string) ([]Issue, error) {
	engine := &Engine{}
	return engine.detectUnnecessarySliceLength(filename)
}

func (e *Engine) detectUnnecessarySliceLength(filename string) ([]Issue, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	ast.Inspect(node, func(n ast.Node) bool {
		sliceExpr, ok := n.(*ast.SliceExpr)
		if !ok {
			return true
		}

		if callExpr, ok := sliceExpr.High.(*ast.CallExpr); ok {
			if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "len" {
				if arg, ok := callExpr.Args[0].(*ast.Ident); ok {
					if sliceExpr.X.(*ast.Ident).Name == arg.Name {
						var suggestion string
						baseMessage := "unnecessary use of len() in slice expression, can be simplified"

						if sliceExpr.Low == nil {
							suggestion = fmt.Sprintf("%s[:]", arg.Name)
						} else if basicLit, ok := sliceExpr.Low.(*ast.BasicLit); ok {
							suggestion = fmt.Sprintf("%s[%s:]", arg.Name, basicLit.Value)
						} else if lowIdent, ok := sliceExpr.Low.(*ast.Ident); ok {
							suggestion = fmt.Sprintf("%s[%s:]", arg.Name, lowIdent.Name)
						}

						issue := Issue{
							Rule: "simplify-slice-range",
							Filename: filename,
							Start: fset.Position(sliceExpr.Pos()),
							End: fset.Position(sliceExpr.End()),
							Message: baseMessage,
							Suggestion: suggestion,
						}
						issues = append(issues, issue)
					}
				}
			}
		}

		return true
	})

	return issues, nil
}
