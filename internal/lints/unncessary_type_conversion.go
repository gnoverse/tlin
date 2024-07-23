package lints

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"

	tt "github.com/gnoswap-labs/lint/internal/types"
)

func DetectUnnecessaryConversions(filename string) ([]tt.Issue, error) {
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

	var issues []tt.Issue
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
				if containsNode(node, n) {
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

			issues = append(issues, tt.Issue{
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

// containsNode checks if parent containsNode child node
func containsNode(parent, child ast.Node) bool {
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
