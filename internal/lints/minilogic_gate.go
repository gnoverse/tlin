package lints

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"

	"github.com/gnolang/tlin/internal/minilogic"
)

func verifyWithMiniLogic(original, suggestion string) bool {
	origStmt, ok := minilogicStmtFromSnippet(original)
	if !ok {
		return false
	}
	transStmt, ok := minilogicStmtFromSnippet(suggestion)
	if !ok {
		return false
	}

	ml := minilogic.New()
	report := ml.Verify(origStmt, transStmt)
	if report.Result == minilogic.Equivalent {
		return true
	}
	if report.Result == minilogic.NotEquivalent {
		return false
	}

	normOrig := ml.Normalize(origStmt)
	normTrans := ml.Normalize(transStmt)
	flatOrig := flattenAllIfElseChains(normOrig)
	flatTrans := flattenAllIfElseChains(normTrans)
	return reflect.DeepEqual(flatOrig, flatTrans)
}

func minilogicStmtFromSnippet(snippet string) (minilogic.Stmt, bool) {
	fset := token.NewFileSet()
	src := "package p\nfunc _() {\n" + snippet + "\n}"
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, false
	}

	var block *ast.BlockStmt
	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			block = fd.Body
			return false
		}
		return true
	})
	if block == nil {
		return nil, false
	}

	return astBlockToMini(block)
}

func astBlockToMini(block *ast.BlockStmt) (minilogic.Stmt, bool) {
	stmts := make([]minilogic.Stmt, 0, len(block.List))
	for _, stmt := range block.List {
		mStmt, ok := astStmtToMini(stmt)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, mStmt)
	}
	return minilogic.Seq(stmts...), true
}

func astStmtToMini(stmt ast.Stmt) (minilogic.Stmt, bool) {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		return astBlockToMini(s)
	case *ast.IfStmt:
		return astIfToMini(s)
	case *ast.ReturnStmt:
		return astReturnToMini(s)
	case *ast.AssignStmt:
		return astAssignToMini(s)
	case *ast.DeclStmt:
		return astDeclStmtToMini(s)
	case *ast.ExprStmt:
		return astExprStmtToMini(s)
	case *ast.EmptyStmt:
		return minilogic.NoopStmt{}, true
	default:
		return nil, false
	}
}

func astIfToMini(stmt *ast.IfStmt) (minilogic.Stmt, bool) {
	var initStmt minilogic.Stmt
	if stmt.Init != nil {
		var ok bool
		initStmt, ok = astIfInitToMini(stmt.Init)
		if !ok {
			return nil, false
		}
	}
	cond, ok := astExprToMini(stmt.Cond)
	if !ok {
		return nil, false
	}
	thenStmt, ok := astBlockToMini(stmt.Body)
	if !ok {
		return nil, false
	}

	var elseStmt minilogic.Stmt
	if stmt.Else != nil {
		var ok bool
		elseStmt, ok = astStmtToMini(stmt.Else)
		if !ok {
			return nil, false
		}
	}

	if initStmt != nil {
		return minilogic.IfInit(initStmt, cond, thenStmt, elseStmt), true
	}
	return minilogic.If(cond, thenStmt, elseStmt), true
}

func astIfInitToMini(stmt ast.Stmt) (minilogic.Stmt, bool) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if s.Tok != token.DEFINE {
			return nil, false
		}
		return astAssignToMini(s)
	case *ast.DeclStmt:
		return astDeclStmtToMini(s)
	default:
		return nil, false
	}
}

func astReturnToMini(stmt *ast.ReturnStmt) (minilogic.Stmt, bool) {
	if len(stmt.Results) == 0 {
		return nil, false
	}
	if len(stmt.Results) > 1 {
		return nil, false
	}
	val, ok := astExprToMini(stmt.Results[0])
	if !ok {
		return nil, false
	}
	return minilogic.Return(val), true
}

func astAssignToMini(stmt *ast.AssignStmt) (minilogic.Stmt, bool) {
	switch stmt.Tok {
	case token.ASSIGN:
		if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
			return nil, false
		}
		ident, ok := stmt.Lhs[0].(*ast.Ident)
		if !ok {
			return nil, false
		}
		expr, ok := astExprToMini(stmt.Rhs[0])
		if !ok {
			return nil, false
		}
		return minilogic.Assign(ident.Name, expr), true
	case token.DEFINE:
		if len(stmt.Rhs) != 1 || len(stmt.Lhs) == 0 {
			return nil, false
		}
		expr, ok := astExprToMini(stmt.Rhs[0])
		if !ok {
			return nil, false
		}
		vars := make([]string, len(stmt.Lhs))
		for i, lhs := range stmt.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				return nil, false
			}
			vars[i] = ident.Name
		}
		if len(vars) == 1 {
			return minilogic.DeclAssign(vars[0], expr), true
		}
		return minilogic.DeclAssignMulti(vars, expr), true
	default:
		return nil, false
	}
}

func astDeclStmtToMini(stmt *ast.DeclStmt) (minilogic.Stmt, bool) {
	gen, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.VAR {
		return nil, false
	}
	if len(gen.Specs) != 1 {
		return nil, false
	}
	spec, ok := gen.Specs[0].(*ast.ValueSpec)
	if !ok {
		return nil, false
	}
	if len(spec.Names) != 1 {
		return nil, false
	}
	if len(spec.Values) > 1 {
		return nil, false
	}

	name := spec.Names[0].Name
	if len(spec.Values) == 0 {
		return minilogic.VarDecl(name, nil), true
	}
	expr, ok := astExprToMini(spec.Values[0])
	if !ok {
		return nil, false
	}
	return minilogic.VarDecl(name, expr), true
}

func astExprStmtToMini(stmt *ast.ExprStmt) (minilogic.Stmt, bool) {
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return nil, false
	}

	funcName, ok := callFuncName(call.Fun)
	if !ok {
		return nil, false
	}

	args := make([]minilogic.Expr, len(call.Args))
	for i, arg := range call.Args {
		converted, ok := astExprToMini(arg)
		if !ok {
			return nil, false
		}
		args[i] = converted
	}

	return minilogic.Call(funcName, args...), true
}

func astExprToMini(expr ast.Expr) (minilogic.Expr, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			val, err := strconv.ParseInt(e.Value, 0, 64)
			if err != nil {
				return nil, false
			}
			return minilogic.IntLit(val), true
		case token.STRING:
			val, err := strconv.Unquote(e.Value)
			if err != nil {
				return nil, false
			}
			return minilogic.StrLit(val), true
		default:
			return nil, false
		}
	case *ast.Ident:
		switch e.Name {
		case "true":
			return minilogic.BoolLit(true), true
		case "false":
			return minilogic.BoolLit(false), true
		case "nil":
			return minilogic.NilLit(), true
		default:
			return minilogic.Var(e.Name), true
		}
	case *ast.BinaryExpr:
		left, ok := astExprToMini(e.X)
		if !ok {
			return nil, false
		}
		right, ok := astExprToMini(e.Y)
		if !ok {
			return nil, false
		}
		op, ok := tokenToBinaryOp(e.Op)
		if !ok {
			return nil, false
		}
		return minilogic.Binary(op, left, right), true
	case *ast.UnaryExpr:
		operand, ok := astExprToMini(e.X)
		if !ok {
			return nil, false
		}
		op, ok := tokenToUnaryOp(e.Op)
		if !ok {
			return nil, false
		}
		return minilogic.Unary(op, operand), true
	case *ast.ParenExpr:
		return astExprToMini(e.X)
	case *ast.CallExpr:
		funcName, ok := callFuncName(e.Fun)
		if !ok {
			return nil, false
		}
		args := make([]minilogic.Expr, len(e.Args))
		for i, arg := range e.Args {
			converted, ok := astExprToMini(arg)
			if !ok {
				return nil, false
			}
			args[i] = converted
		}
		return minilogic.CallExpr{Func: funcName, Args: args}, true
	default:
		return nil, false
	}
}

func callFuncName(expr ast.Expr) (string, bool) {
	switch fn := expr.(type) {
	case *ast.Ident:
		return fn.Name, true
	case *ast.SelectorExpr:
		if ident, ok := fn.X.(*ast.Ident); ok {
			return ident.Name + "." + fn.Sel.Name, true
		}
	}
	return "", false
}

func tokenToBinaryOp(tok token.Token) (minilogic.BinaryOp, bool) {
	switch tok {
	case token.ADD:
		return minilogic.OpAdd, true
	case token.SUB:
		return minilogic.OpSub, true
	case token.MUL:
		return minilogic.OpMul, true
	case token.QUO:
		return minilogic.OpDiv, true
	case token.REM:
		return minilogic.OpMod, true
	case token.EQL:
		return minilogic.OpEq, true
	case token.NEQ:
		return minilogic.OpNeq, true
	case token.LSS:
		return minilogic.OpLt, true
	case token.LEQ:
		return minilogic.OpLte, true
	case token.GTR:
		return minilogic.OpGt, true
	case token.GEQ:
		return minilogic.OpGte, true
	case token.LAND:
		return minilogic.OpAnd, true
	case token.LOR:
		return minilogic.OpOr, true
	default:
		return 0, false
	}
}

func tokenToUnaryOp(tok token.Token) (minilogic.UnaryOp, bool) {
	switch tok {
	case token.NOT:
		return minilogic.OpNot, true
	case token.SUB:
		return minilogic.OpNeg, true
	default:
		return 0, false
	}
}

func flattenAllIfElseChains(stmt minilogic.Stmt) minilogic.Stmt {
	switch s := stmt.(type) {
	case minilogic.SeqStmt:
		first := flattenAllIfElseChains(s.First)
		second := flattenAllIfElseChains(s.Second)
		return minilogic.Seq(first, second)
	case minilogic.BlockStmt:
		stmts := make([]minilogic.Stmt, 0, len(s.Stmts))
		for _, st := range s.Stmts {
			stmts = append(stmts, flattenAllIfElseChains(st))
		}
		return minilogic.Block(stmts...)
	case minilogic.IfStmt:
		thenStmt := flattenAllIfElseChains(s.Then)
		var elseStmt minilogic.Stmt
		if s.Else != nil {
			elseStmt = flattenAllIfElseChains(s.Else)
		}
		s.Then = thenStmt
		s.Else = elseStmt
		return flattenMiniLogicIfChain(s)
	default:
		return stmt
	}
}

func flattenMiniLogicIfChain(ifStmt minilogic.IfStmt) minilogic.Stmt {
	if !branchTerminates(ifStmt.Then) {
		return ifStmt
	}

	result := []minilogic.Stmt{minilogic.IfStmt{
		Init: ifStmt.Init,
		Cond: ifStmt.Cond,
		Then: ifStmt.Then,
		Else: nil,
	}}

	current := ifStmt.Else
	for current != nil {
		if nested, ok := current.(minilogic.IfStmt); ok {
			if branchTerminates(nested.Then) {
				result = append(result, minilogic.IfStmt{
					Init: nested.Init,
					Cond: nested.Cond,
					Then: nested.Then,
					Else: nil,
				})
				current = nested.Else
				continue
			}
		}
		result = append(result, current)
		break
	}

	return minilogic.Seq(result...)
}

func branchTerminates(stmt minilogic.Stmt) bool {
	switch s := stmt.(type) {
	case minilogic.ReturnStmt, minilogic.BreakStmt, minilogic.ContinueStmt:
		return true
	case minilogic.BlockStmt:
		if len(s.Stmts) == 0 {
			return false
		}
		return branchTerminates(s.Stmts[len(s.Stmts)-1])
	case minilogic.SeqStmt:
		if branchTerminates(s.First) {
			return true
		}
		return branchTerminates(s.Second)
	case minilogic.IfStmt:
		thenTerms := branchTerminates(s.Then)
		elseTerms := s.Else != nil && branchTerminates(s.Else)
		return thenTerms && elseTerms
	default:
		return false
	}
}
