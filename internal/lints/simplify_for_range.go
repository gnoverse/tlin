package lints

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/importer"
	"go/token"
	"go/types"

	tt "github.com/gnolang/tlin/internal/types"
)

// DetectSimplifiableForLoops detects counter-based for-loops that can be safely
// rewritten to Go/Gno-style range loops:
//
//	for i := 0; i < len(xs); i++ { ... }
//
// ->   for i := range xs { ... }
//
// This simplification is allowed only when the loop follows the *canonical*
// index-loop pattern used for iterating over slices, arrays, or strings.
// The checker performs full syntactic analysis plus type information from
// go/types (best-effort in Gno).
//
// A loop is simplifiable if and only if all of the following conditions hold:
//
//  1. **Initialization is canonical**
//     i := 0
//
//     - Must be a short variable declaration (`:=`).
//     - Must declare exactly one identifier.
//     - Initial value must be an integer literal equal to zero.
//
//  2. **Condition is canonical**
//     i < len(xs)
//
//     - Must be a binary expression with `<`.
//     - The left-hand side must be the *same declared variable* `i`.
//     - The right-hand side must be a call to the built-in `len` with exactly one
//     argument: `len(xs)`.
//
//  3. **Increment is canonical**
//     i++
//
//     - Must be `i++` (post-increment). Forms like `i += 1` or `i = i + 1` are rejected.
//
//  4. **Target of len(...) is iterable**
//     xs is a slice, array, or string
//
//     - Determined via go/types (`t.Underlying()`).
//     - Maps, structs, numeric expressions, constants, arithmetic expressions, and
//     function calls are not iterable.
//     - This ensures that `for i := range xs` is valid and preserves semantics.
//
// 5. **Loop counter is not modified inside the loop body**
//
//	All of the following constitute modification and prevent simplification:
//	- `i = ...`
//	- `i++` or `i--`
//	- `i += ...`, `i -= ...`, etc.
//	- Inclusion in multi-assignments (e.g. `x, i = ...`)
//
//	Importantly, **declarations inside the loop body like `var i int` shadow the
//	loop variable**, but are *not* considered a modification. Shadowing is
//	detected via object identity (`Ident.Obj`) and is allowed.
//
// 6. **Loop counter is not used after the loop**
//
//	Using the final value of `i` after the loop implies that the exact iteration
//	count matters, which range loops do not replicate. Example:
//
//	    for i := 0; i < len(xs); i++ { ... }
//	    println(i)   // disallow
//
// 7. **No non-canonical loop shape**
//
//	The following patterns disqualify simplification:
//	- condition is not `<`
//	- condition uses anything other than `len(xs)`
//	- non-canonical init or post statements (`i = 0`, `i += 2`, etc.)
//	- multiple variables in init/cond/post
//	- complex expressions for the collection (e.g., `len(xs[foo()])`)
//
// ## Shadowing vs Modification
//
// Gno code frequently shadows identifiers inside nested blocks. Shadowing does
// *not* modify the loop counter and does *not* prevent simplification:
//
//	for i := 0; i < len(xs); i++ {
//	    var i int    // shadowing: allowed
//	    _ = i
//	}
//
// Modification detection covers assignments, inc/dec, and multi-assign targets,
// and it traverses DeclStmt -> GenDecl -> ValueSpec to detect shadows correctly.
//
// ## Summary
//
// A loop is simplified *only* when it is a pure, canonical “iterate over indices
// of xs” loop and no surrounding semantics depend on the exact final index value
// or mutation of the loop variable.
//
// All other counter-based loops remain untouched.
func DetectSimplifiableForLoops(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Uses:  make(map[*ast.Ident]types.Object),
		Defs:  make(map[*ast.Ident]types.Object),
	}

	conf := types.Config{Importer: importer.Default()}
	// ignore type-checking errors to keep linting best-effort.
	_, _ = conf.Check(node.Name.Name, fset, []*ast.File{node}, info)

	var issues []tt.Issue

	ast.Inspect(node, func(n ast.Node) bool {
		forStmt, ok := n.(*ast.ForStmt)
		if !ok {
			return true
		}

		idxIdent, collectionExpr, ok := extractLoopComponents(forStmt, info)
		if !ok {
			return true
		}

		if loopVarModified(forStmt.Body, idxIdent) {
			return true
		}

		if loopVarUsedAfter(node, forStmt, idxIdent) {
			return true
		}

		bodyStr := tt.Node2String(forStmt.Body)
		suggestion := fmt.Sprintf(
			"for %s := range %s %s",
			idxIdent.Name,
			tt.Node2String(collectionExpr),
			bodyStr,
		)

		issues = append(issues, tt.Issue{
			Rule:       "simplify_for_range",
			Filename:   filename,
			Message:    "counter-based for loop can be simplified to range-based loop",
			Category:   "style",
			Start:      fset.Position(forStmt.Pos()),
			End:        fset.Position(forStmt.End()),
			Severity:   severity,
			Suggestion: suggestion,
		})

		return true
	})

	return issues, nil
}

func extractLoopComponents(forStmt *ast.ForStmt, info *types.Info) (*ast.Ident, ast.Expr, bool) {
	init, ok := forStmt.Init.(*ast.AssignStmt)
	if !ok || init.Tok != token.DEFINE || len(init.Lhs) != 1 || len(init.Rhs) != 1 {
		return nil, nil, false
	}

	initIdent, ok := init.Lhs[0].(*ast.Ident)
	if !ok || initIdent.Name == "" {
		return nil, nil, false
	}

	if !isZeroLiteral(init.Rhs[0]) {
		return nil, nil, false
	}

	cond, ok := forStmt.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.LSS {
		return nil, nil, false
	}

	post, ok := forStmt.Post.(*ast.IncDecStmt)
	if !ok || post.Tok != token.INC {
		return nil, nil, false
	}

	postIdent, ok := post.X.(*ast.Ident)
	if !ok || !sameIdent(postIdent, initIdent) {
		return nil, nil, false
	}

	condLeftIdent, ok := unwrapIdent(cond.X)
	if !ok || !sameIdent(condLeftIdent, initIdent) {
		return nil, nil, false
	}

	call, ok := unwrapCall(cond.Y)
	if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() {
		return nil, nil, false
	}

	lenIdent, ok := call.Fun.(*ast.Ident)
	if !ok || lenIdent.Name != "len" || lenIdent.Obj != nil {
		return nil, nil, false
	}

	arg := unwrapParen(call.Args[0])
	if !isPlainSelectorOrIdent(arg) {
		return nil, nil, false
	}

	if !isIterable(info, arg) {
		return nil, nil, false
	}

	return initIdent, arg, true
}

func isZeroLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return false
	}
	if lit.Value == "0" {
		return true
	}

	// Handle other zero representations like 0x0
	val := constant.MakeFromLiteral(lit.Value, token.INT, 0)
	if val.Kind() == constant.Unknown {
		return false
	}
	return constant.Sign(val) == 0
}

func unwrapIdent(expr ast.Expr) (*ast.Ident, bool) {
	expr = unwrapParen(expr)
	ident, ok := expr.(*ast.Ident)
	return ident, ok
}

func unwrapCall(expr ast.Expr) (*ast.CallExpr, bool) {
	expr = unwrapParen(expr)
	call, ok := expr.(*ast.CallExpr)
	return call, ok
}

func unwrapParen(expr ast.Expr) ast.Expr {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = paren.X
	}
	return expr
}

func sameIdent(a, b *ast.Ident) bool {
	if a == nil || b == nil {
		return false
	}

	// A variable is uniquely identified by its Obj.
	// If either Obj is nil, we CANNOT reliably compare.
	if a.Obj == nil || b.Obj == nil {
		return false
	}

	return a.Obj == b.Obj
}

func isPlainSelectorOrIdent(expr ast.Expr) bool {
	switch node := expr.(type) {
	case *ast.Ident:
		return node.Name != ""
	case *ast.SelectorExpr:
		return isPlainSelectorOrIdent(node.X)
	default:
		return false
	}
}

func isIterable(info *types.Info, expr ast.Expr) bool {
	if info == nil {
		return false
	}

	tv, ok := info.Types[expr]
	if !ok || tv.Type == nil {
		return false
	}

	switch t := tv.Type.Underlying().(type) {
	case *types.Slice, *types.Array:
		return true
	case *types.Basic:
		return t.Kind() == types.String
	default:
		return false
	}
}

func loopVarModified(body *ast.BlockStmt, loopVar *ast.Ident) bool {
	if body == nil {
		return false
	}

	var modified bool
	ast.Inspect(body, func(n ast.Node) bool {
		if modified {
			return false
		}

		switch stmt := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range stmt.Lhs {
				ident, ok := lhs.(*ast.Ident)
				if !ok || ident.Name == "" {
					continue
				}
				if sameIdent(ident, loopVar) || (stmt.Tok == token.DEFINE && ident.Name == loopVar.Name) {
					modified = true
					return false
				}
			}
		case *ast.IncDecStmt:
			if ident, ok := stmt.X.(*ast.Ident); ok && sameIdent(ident, loopVar) {
				modified = true
				return false
			}
		case *ast.DeclStmt:
			gen, ok := stmt.Decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				return true
			}
			for _, spec := range gen.Specs {
				valSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range valSpec.Names {
					if name == nil || name.Name == "" {
						continue
					}
					if sameIdent(name, loopVar) || name.Name == loopVar.Name {
						modified = true
						return false
					}
				}
			}
		}
		return true
	})

	return modified
}

func loopVarUsedAfter(root ast.Node, forStmt *ast.ForStmt, loopVar *ast.Ident) bool {
	var used bool
	forEnd := forStmt.End()

	ast.Inspect(root, func(n ast.Node) bool {
		if used {
			return false
		}
		ident, ok := n.(*ast.Ident)
		if !ok || !sameIdent(ident, loopVar) {
			return true
		}
		if ident.Pos() > forEnd {
			used = true
			return false
		}
		return true
	})

	return used
}
