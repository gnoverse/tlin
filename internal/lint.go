package internal

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os/exec"

	"github.com/gnoswap-labs/lint/internal/lints"
)

/*
* Implement each lint rule as a separate struct
 */

// LintRule defines the interface for all lint rules.
type LintRule interface {
	// Check runs the lint rule on the given file and returns a slice of Issues.
	Check(filename string) ([]lints.Issue, error)
}

type GolangciLintRule struct{}

func (r *GolangciLintRule) Check(filename string) ([]lints.Issue, error) {
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

func runGolangciLint(filename string) ([]lints.Issue, error) {
	cmd := exec.Command("golangci-lint", "run", "--disable=gosimple", "--out-format=json", filename)
	output, _ := cmd.CombinedOutput()

	var golangciResult golangciOutput
	if err := json.Unmarshal(output, &golangciResult); err != nil {
		return nil, fmt.Errorf("error unmarshaling golangci-lint output: %w", err)
	}

	var issues []lints.Issue
	for _, gi := range golangciResult.Issues {
		issues = append(issues, lints.Issue{
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

func (r *UnnecessaryElseRule) Check(filename string) ([]lints.Issue, error) {
	engine := &Engine{}
	return engine.detectUnnecessaryElse(filename)
}

// detectUnnecessaryElse detects unnecessary else blocks.
// This rule considers an else block unnecessary if the if block ends with a return statement.
// In such cases, the else block can be removed and the code can be flattened to improve readability.
func (e *Engine) detectUnnecessaryElse(filename string) ([]lints.Issue, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var issues []lints.Issue
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
					issue := lints.Issue{
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

func (r *UnusedFunctionRule) Check(filename string) ([]lints.Issue, error) {
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
func (e *Engine) detectUnusedFunctions(filename string) ([]lints.Issue, error) {
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

	var issues []lints.Issue
	for funcName, funcDecl := range declaredFuncs {
		if !calledFuncs[funcName] && funcName != "main" && funcName != "init" && !ast.IsExported(funcName) {
			issue := lints.Issue{
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

func (r *SimplifySliceExprRule) Check(filename string) ([]lints.Issue, error) {
	engine := &Engine{}
	return engine.detectUnnecessarySliceLength(filename)
}

func (e *Engine) detectUnnecessarySliceLength(filename string) ([]lints.Issue, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var issues []lints.Issue
	ast.Inspect(node, func(n ast.Node) bool {
		sliceExpr, ok := n.(*ast.SliceExpr)
		if !ok {
			return true
		}

		if callExpr, ok := sliceExpr.High.(*ast.CallExpr); ok {
			if ident, ok := callExpr.Fun.(*ast.Ident); ok && ident.Name == "len" {
				if arg, ok := callExpr.Args[0].(*ast.Ident); ok {
					if sliceExpr.X.(*ast.Ident).Name == arg.Name {
						var suggestion, detailedMessage string
						baseMessage := "unnecessary use of len() in slice expression, can be simplified"

						if sliceExpr.Low == nil {
							suggestion = fmt.Sprintf("%s[:]", arg.Name)
							detailedMessage = fmt.Sprintf(
								"%s\nIn this case, `%s[:len(%s)]` is equivalent to `%s[:]`. "+
									"The full length of the slice is already implied when omitting both start and end indices.",
								baseMessage, arg.Name, arg.Name, arg.Name)
						} else if basicLit, ok := sliceExpr.Low.(*ast.BasicLit); ok {
							suggestion = fmt.Sprintf("%s[%s:]", arg.Name, basicLit.Value)
							detailedMessage = fmt.Sprintf("%s\nHere, `%s[%s:len(%s)]` can be simplified to `%s[%s:]`. "+
								"When slicing to the end of a slice, using len() is unnecessary.",
								baseMessage, arg.Name, basicLit.Value, arg.Name, arg.Name, basicLit.Value)
						} else if lowIdent, ok := sliceExpr.Low.(*ast.Ident); ok {
							suggestion = fmt.Sprintf("%s[%s:]", arg.Name, lowIdent.Name)
							detailedMessage = fmt.Sprintf("%s\nIn this instance, `%s[%s:len(%s)]` can be written as `%s[%s:]`. "+
								"The len() function is redundant when slicing to the end, regardless of the start index.",
								baseMessage, arg.Name, lowIdent.Name, arg.Name, arg.Name, lowIdent.Name)
						}

						issue := lints.Issue{
							Rule:       "simplify-slice-range",
							Filename:   filename,
							Start:      fset.Position(sliceExpr.Pos()),
							End:        fset.Position(sliceExpr.End()),
							Message:    baseMessage,
							Suggestion: suggestion,
							Note:       detailedMessage,
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

type UnnecessaryConversionRule struct{}

func (r *UnnecessaryConversionRule) Check(filename string) ([]lints.Issue, error) {
	engine := &Engine{}
	return engine.detectUnnecessaryConversions(filename)
}

func (e *Engine) detectUnnecessaryConversions(filename string) ([]lints.Issue, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	conf := types.Config{Importer: importer.Default()}
	//! DO NOT CHECK ERROR HERE.
	//! error check may broke the lint formatting process.
	conf.Check("", fset, []*ast.File{f}, info)

	var issues []lints.Issue
	varDecls := make(map[*types.Var]ast.Node)

	// First pass: collect variable declarations
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ValueSpec:
			for _, name := range node.Names {
				if obj := info.Defs[name]; obj != nil {
					if v, ok := obj.(*types.Var); ok {
						varDecls[v] = node
					}
				}
			}
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				if id, ok := lhs.(*ast.Ident); ok {
					if obj := info.Defs[id]; obj != nil {
						if v, ok := obj.(*types.Var); ok {
							varDecls[v] = node
						}
					}
				}
			}
		}
		return true
	})

	// Second pass: check for unnecessary conversions
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}

		ft, ok := info.Types[call.Fun]
		if !ok || !ft.IsType() {
			return true
		}

		at, ok := info.Types[call.Args[0]]
		if !ok {
			return true
		}

		if types.Identical(ft.Type, at.Type) && !isUntypedValue(call.Args[0], info) {
			var memo, suggestion string

			// find parent node and retrieve the entire assignment statement
			var parent ast.Node
			ast.Inspect(f, func(node ast.Node) bool {
				if node == n {
					return false
				}
				if contains(node, n) {
					parent = node
					return false
				}
				return true
			})

			if assignStmt, ok := parent.(*ast.AssignStmt); ok {
				if len(assignStmt.Lhs) > 0 {
					lhs := types.ExprString(assignStmt.Lhs[0])
					rhs := types.ExprString(call.Args[0])
					suggestion = fmt.Sprintf("%s := %s", lhs, rhs)
				}
			} else {
				// not an assignment statement
				// keep maintaining the original code
				suggestion = types.ExprString(call.Args[0])
			}

			if id, ok := call.Args[0].(*ast.Ident); ok {
				if obj, ok := info.Uses[id].(*types.Var); ok {
					if _, exists := varDecls[obj]; exists {
						declType := obj.Type().String()
						memo = fmt.Sprintf(
							"The variable '%s' is declared as type '%s'. This type conversion appears unnecessary.",
							id.Name, declType,
						)
					}
				}
			}

			issues = append(issues, lints.Issue{
				Rule:       "unnecessary-type-conversion",
				Filename:   filename,
				Start:      fset.Position(call.Pos()),
				End:        fset.Position(call.End()),
				Message:    "unnecessary type conversion",
				Suggestion: suggestion,
				Note:       memo,
			})
		}

		return true
	})

	return issues, nil
}

// ref: https://github.com/mdempsky/unconvert/blob/master/unconvert.go#L570
func isUntypedValue(n ast.Expr, info *types.Info) (res bool) {
	switch n := n.(type) {
	case *ast.BinaryExpr:
		switch n.Op {
		case token.SHL, token.SHR:
			// Shifts yield an untyped value if their LHS is untyped.
			return isUntypedValue(n.X, info)
		case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ:
			// Comparisons yield an untyped boolean value.
			return true
		case token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
			token.AND, token.OR, token.XOR, token.AND_NOT,
			token.LAND, token.LOR:
			return isUntypedValue(n.X, info) && isUntypedValue(n.Y, info)
		}
	case *ast.UnaryExpr:
		switch n.Op {
		case token.ADD, token.SUB, token.NOT, token.XOR:
			return isUntypedValue(n.X, info)
		}
	case *ast.BasicLit:
		// Basic literals are always untyped.
		return true
	case *ast.ParenExpr:
		return isUntypedValue(n.X, info)
	case *ast.SelectorExpr:
		return isUntypedValue(n.Sel, info)
	case *ast.Ident:
		if obj, ok := info.Uses[n]; ok {
			if obj.Pkg() == nil && obj.Name() == "nil" {
				// The universal untyped zero value.
				return true
			}
			if b, ok := obj.Type().(*types.Basic); ok && b.Info()&types.IsUntyped != 0 {
				// Reference to an untyped constant.
				return true
			}
		}
	case *ast.CallExpr:
		if b, ok := asBuiltin(n.Fun, info); ok {
			switch b.Name() {
			case "real", "imag":
				return isUntypedValue(n.Args[0], info)
			case "complex":
				return isUntypedValue(n.Args[0], info) && isUntypedValue(n.Args[1], info)
			}
		}
	}

	return false
}

func asBuiltin(n ast.Expr, info *types.Info) (*types.Builtin, bool) {
	for {
		paren, ok := n.(*ast.ParenExpr)
		if !ok {
			break
		}
		n = paren.X
	}

	ident, ok := n.(*ast.Ident)
	if !ok {
		return nil, false
	}

	obj, ok := info.Uses[ident]
	if !ok {
		return nil, false
	}

	b, ok := obj.(*types.Builtin)
	return b, ok
}

// contains checks if parent contains child node
func contains(parent, child ast.Node) bool {
	found := false
	ast.Inspect(parent, func(n ast.Node) bool {
		if n == child {
			found = true
			return false
		}
		return true
	})
	return found
}
