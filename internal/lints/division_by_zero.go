package lints

import (
	"go/ast"
	"go/constant"
	"go/token"

	"github.com/gnolang/tlin/internal/analysis/cfg"
	"github.com/gnolang/tlin/internal/analysis/lattice"
	tt "github.com/gnolang/tlin/internal/types"
)

type divisionIssue struct {
	BlockID int
	Node    ast.Node
	Level   string // ERROR | WARNING
	Reason  string
}

type divConfig struct {
	DivCallArgIndex    int
	DivCallArgIndexSet bool
}

// DetectDivisionByZero runs a CFG-based forward analysis and reports division issues.
func DetectDivisionByZero(filename string, node *ast.File, fset *token.FileSet, _ tt.Severity) ([]tt.Issue, error) {
	var issues []tt.Issue
	cfgConfig := divConfig{DivCallArgIndex: 1}

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		graph := cfg.FromFunc(fn)
		if graph == nil {
			continue
		}

		for _, issue := range analyzeCFG(graph, cfgConfig) {
			if issue.Node == nil {
				continue
			}
			start := fset.Position(issue.Node.Pos())
			end := fset.Position(issue.Node.End())
			issues = append(issues, tt.Issue{
				Rule:     "division-by-zero",
				Filename: filename,
				Start:    start,
				End:      end,
				Message:  "possible division by zero: " + issue.Reason,
				Severity: severityForLevel(issue.Level),
			})
		}
	}

	return issues, nil
}

func severityForLevel(level string) tt.Severity {
	switch level {
	case "ERROR":
		return tt.SeverityError
	case "WARNING":
		return tt.SeverityWarning
	default:
		return tt.SeverityWarning
	}
}

func analyzeCFG(graph *cfg.CFG, config divConfig) []divisionIssue {
	if graph == nil {
		return nil
	}
	if !config.DivCallArgIndexSet {
		config.DivCallArgIndex = 1
	} else if config.DivCallArgIndex < 0 {
		config.DivCallArgIndex = 1
	}

	blocks := graph.Blocks()
	graph.Sort(blocks)
	blockIDs := make(map[ast.Stmt]int, len(blocks))
	nextID := 0
	for _, stmt := range blocks {
		if stmt == nil || stmt == graph.Entry || stmt == graph.Exit {
			continue
		}
		blockIDs[stmt] = nextID
		nextID++
	}

	in := make(map[ast.Stmt]lattice.AbstractState, len(blocks))
	out := make(map[ast.Stmt]lattice.AbstractState, len(blocks))
	worklist := []ast.Stmt{graph.Entry}
	inWorklist := map[ast.Stmt]bool{graph.Entry: true}

	in[graph.Entry] = lattice.AbstractState{}

	for len(worklist) > 0 {
		stmt := worklist[0]
		worklist = worklist[1:]
		inWorklist[stmt] = false

		newIn := computeIn(graph, stmt, out)
		if !lattice.StateEqual(newIn, in[stmt]) {
			in[stmt] = newIn
		}

		newOut := transfer(stmt, newIn)
		if lattice.StateEqual(newOut, out[stmt]) {
			continue
		}
		out[stmt] = newOut

		for _, succ := range graph.Succs(stmt) {
			if succ == nil || inWorklist[succ] {
				continue
			}
			worklist = append(worklist, succ)
			inWorklist[succ] = true
		}
	}

	var issues []divisionIssue
	for _, stmt := range blocks {
		if stmt == nil || stmt == graph.Entry || stmt == graph.Exit {
			continue
		}
		issues = append(issues, findDivisionIssues(stmt, in[stmt], blockIDs[stmt], config)...)
	}

	return issues
}

func computeIn(graph *cfg.CFG, stmt ast.Stmt, out map[ast.Stmt]lattice.AbstractState) lattice.AbstractState {
	if stmt == graph.Entry {
		return lattice.AbstractState{}
	}
	var joined lattice.AbstractState
	for _, pred := range graph.Preds(stmt) {
		edgeState := refineForEdge(pred, stmt, out[pred])
		joined = lattice.JoinStates(joined, edgeState)
	}
	return joined
}

// transfer updates the abstract state for a single statement.
func transfer(stmt ast.Stmt, in lattice.AbstractState) lattice.AbstractState {
	if in == nil {
		return nil
	}
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		return transferAssign(s, in)
	case *ast.DeclStmt:
		return transferDecl(s, in)
	case *ast.IncDecStmt:
		return transferIncDec(s, in)
	default:
		return in
	}
}

func transferAssign(stmt *ast.AssignStmt, in lattice.AbstractState) lattice.AbstractState {
	if in == nil {
		return nil
	}
	out := lattice.CloneState(in)

	isSimpleAssign := stmt.Tok == token.ASSIGN || stmt.Tok == token.DEFINE
	for i, lhs := range stmt.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}

		value := lattice.Top
		if isSimpleAssign {
			if len(stmt.Rhs) == len(stmt.Lhs) && i < len(stmt.Rhs) {
				value = evalExpr(stmt.Rhs[i], in)
			}
		} else {
			var rhs lattice.ValueKind = lattice.Top
			if len(stmt.Rhs) == len(stmt.Lhs) && i < len(stmt.Rhs) {
				rhs = evalExpr(stmt.Rhs[i], in)
			}
			lhsVal := lattice.GetValue(in, ident.Name)
			value = combineBinary(opFromAssign(stmt.Tok), lhsVal, rhs)
		}

		lattice.SetValue(out, ident.Name, value)
	}

	return out
}

func transferDecl(stmt *ast.DeclStmt, in lattice.AbstractState) lattice.AbstractState {
	if in == nil {
		return nil
	}
	gen, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.VAR {
		return in
	}
	out := lattice.CloneState(in)
	for _, spec := range gen.Specs {
		valSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range valSpec.Names {
			if name.Name == "_" {
				continue
			}
			value := lattice.Top
			if i < len(valSpec.Values) {
				value = evalExpr(valSpec.Values[i], in)
			}
			lattice.SetValue(out, name.Name, value)
		}
	}
	return out
}

func transferIncDec(stmt *ast.IncDecStmt, in lattice.AbstractState) lattice.AbstractState {
	if in == nil {
		return nil
	}
	ident, ok := stmt.X.(*ast.Ident)
	if !ok || ident.Name == "_" {
		return in
	}
	out := lattice.CloneState(in)
	cur := lattice.GetValue(in, ident.Name)
	lattice.SetValue(out, ident.Name, incDecValue(cur))
	return out
}

func evalExpr(expr ast.Expr, state lattice.AbstractState) lattice.ValueKind {
	if expr == nil {
		return lattice.Top
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return lattice.GetValue(state, e.Name)
	case *ast.BasicLit:
		if divZeroIsZeroLiteral(e) {
			return lattice.Zero
		}
		if divZeroIsNumericLiteral(e) {
			return lattice.NonZero
		}
		return lattice.Top
	case *ast.UnaryExpr:
		if e.Op == token.ADD || e.Op == token.SUB {
			return evalExpr(e.X, state)
		}
		return lattice.Top
	case *ast.BinaryExpr:
		left := evalExpr(e.X, state)
		right := evalExpr(e.Y, state)
		return combineBinary(e.Op, left, right)
	case *ast.ParenExpr:
		return evalExpr(e.X, state)
	default:
		return lattice.Top
	}
}

func opFromAssign(tok token.Token) token.Token {
	switch tok {
	case token.ADD_ASSIGN:
		return token.ADD
	case token.SUB_ASSIGN:
		return token.SUB
	case token.MUL_ASSIGN:
		return token.MUL
	case token.QUO_ASSIGN:
		return token.QUO
	case token.REM_ASSIGN:
		return token.REM
	default:
		return token.ADD
	}
}

func combineBinary(op token.Token, lhs, rhs lattice.ValueKind) lattice.ValueKind {
	if lhs == lattice.Bottom || rhs == lattice.Bottom {
		return lattice.Bottom
	}
	if lhs == lattice.Top || rhs == lattice.Top {
		return lattice.Top
	}
	if lhs == lattice.MaybeZero || rhs == lattice.MaybeZero {
		switch op {
		case token.MUL:
			if lhs == lattice.Zero || rhs == lattice.Zero {
				return lattice.Zero
			}
		}
		return lattice.MaybeZero
	}

	switch op {
	case token.MUL:
		if lhs == lattice.Zero || rhs == lattice.Zero {
			return lattice.Zero
		}
		if lhs == lattice.NonZero && rhs == lattice.NonZero {
			return lattice.NonZero
		}
		return lattice.MaybeZero
	case token.ADD, token.SUB:
		if lhs == lattice.Zero {
			return rhs
		}
		if rhs == lattice.Zero {
			return lhs
		}
		if lhs == lattice.NonZero && rhs == lattice.NonZero {
			return lattice.MaybeZero
		}
		return lattice.MaybeZero
	case token.QUO, token.REM:
		if rhs == lattice.Zero {
			return lattice.Top
		}
		return lattice.MaybeZero
	default:
		return lattice.MaybeZero
	}
}

func incDecValue(v lattice.ValueKind) lattice.ValueKind {
	switch v {
	case lattice.Zero:
		return lattice.NonZero
	case lattice.NonZero:
		return lattice.MaybeZero
	case lattice.MaybeZero:
		return lattice.MaybeZero
	case lattice.Top:
		return lattice.Top
	default:
		return v
	}
}

func findDivisionIssues(stmt ast.Stmt, in lattice.AbstractState, blockID int, config divConfig) []divisionIssue {
	if in == nil {
		return nil
	}
	var issues []divisionIssue

	if assign, ok := stmt.(*ast.AssignStmt); ok && assign.Tok == token.QUO_ASSIGN {
		if len(assign.Rhs) > 0 {
			if issue, ok := issueForDivisor(assign, evalExpr(assign.Rhs[0], in), blockID); ok {
				issues = append(issues, issue)
			}
		}
	}

	ast.Inspect(stmt, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if n != stmt {
			if _, ok := n.(ast.Stmt); ok {
				return false
			}
		}
		switch expr := n.(type) {
		case *ast.BinaryExpr:
			if expr.Op != token.QUO {
				return true
			}
			if issue, ok := issueForDivisor(expr, evalExpr(expr.Y, in), blockID); ok {
				issues = append(issues, issue)
			}
		case *ast.CallExpr:
			sel, ok := expr.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "Div" {
				return true
			}
			idx := config.DivCallArgIndex
			if idx < 0 || idx >= len(expr.Args) {
				return true
			}
			if issue, ok := issueForDivisor(expr, evalExpr(expr.Args[idx], in), blockID); ok {
				issues = append(issues, issue)
			}
		}
		return true
	})

	return issues
}

func issueForDivisor(node ast.Node, divisor lattice.ValueKind, blockID int) (divisionIssue, bool) {
	switch divisor {
	case lattice.Zero:
		return divisionIssue{
			BlockID: blockID,
			Node:    node,
			Level:   "ERROR",
			Reason:  "divisor is definitely zero",
		}, true
	case lattice.MaybeZero, lattice.Top:
		return divisionIssue{
			BlockID: blockID,
			Node:    node,
			Level:   "WARNING",
			Reason:  "divisor may be zero (" + divisor.String() + ")",
		}, true
	default:
		return divisionIssue{}, false
	}
}

func refineForEdge(pred ast.Stmt, succ ast.Stmt, out lattice.AbstractState) lattice.AbstractState {
	if out == nil {
		return nil
	}
	if ifs, ok := pred.(*ast.IfStmt); ok {
		kind, ok := detectBranchKind(ifs, succ)
		if ok {
			return refineStateForCond(ifs.Cond, out, kind)
		}
	}
	return out
}

type branchKind int

const (
	branchUnknown branchKind = iota
	branchTrue
	branchFalse
)

func detectBranchKind(ifs *ast.IfStmt, succ ast.Stmt) (branchKind, bool) {
	thenEntry := firstStmt(ifs.Body)
	elseEntry := elseEntryStmt(ifs.Else)

	switch {
	case thenEntry != nil && succ == thenEntry:
		return branchTrue, true
	case elseEntry != nil && succ == elseEntry:
		return branchFalse, true
	default:
		return branchUnknown, false
	}
}

func firstStmt(block *ast.BlockStmt) ast.Stmt {
	if block == nil || len(block.List) == 0 {
		return nil
	}
	return block.List[0]
}

func elseEntryStmt(node ast.Stmt) ast.Stmt {
	switch s := node.(type) {
	case *ast.BlockStmt:
		return firstStmt(s)
	case *ast.IfStmt:
		return s
	default:
		return nil
	}
}

func refineStateForCond(cond ast.Expr, out lattice.AbstractState, kind branchKind) lattice.AbstractState {
	name, op, ok := divZeroZeroComparison(cond)
	if !ok {
		return out
	}

	switch op {
	case token.NEQ:
		if kind == branchTrue {
			return refineVar(out, name, lattice.NonZero)
		}
		return refineVar(out, name, lattice.Zero)
	case token.EQL:
		if kind == branchTrue {
			return refineVar(out, name, lattice.Zero)
		}
		return refineVar(out, name, lattice.NonZero)
	case token.GTR, token.LSS:
		if kind == branchTrue {
			return refineVar(out, name, lattice.NonZero)
		}
		return refineVar(out, name, lattice.MaybeZero)
	default:
		return out
	}
}

func divZeroZeroComparison(expr ast.Expr) (string, token.Token, bool) {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return divZeroZeroComparison(e.X)
	case *ast.BinaryExpr:
		if ident, ok := e.X.(*ast.Ident); ok && divZeroIsZeroExpr(e.Y) {
			return ident.Name, e.Op, true
		}
		if ident, ok := e.Y.(*ast.Ident); ok && divZeroIsZeroExpr(e.X) {
			return ident.Name, divZeroFlipOp(e.Op), true
		}
	}
	return "", token.ILLEGAL, false
}

func divZeroFlipOp(op token.Token) token.Token {
	switch op {
	case token.LSS:
		return token.GTR
	case token.GTR:
		return token.LSS
	default:
		return op
	}
}

func divZeroIsZeroExpr(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return divZeroIsZeroLiteral(e)
	case *ast.UnaryExpr:
		if e.Op == token.ADD || e.Op == token.SUB {
			return divZeroIsZeroExpr(e.X)
		}
	}
	return false
}

func divZeroIsZeroLiteral(lit *ast.BasicLit) bool {
	if !divZeroIsNumericLiteral(lit) {
		return false
	}
	val := constant.MakeFromLiteral(lit.Value, lit.Kind, 0)
	return val.Kind() != constant.Unknown && constant.Sign(val) == 0
}

func divZeroIsNumericLiteral(lit *ast.BasicLit) bool {
	switch lit.Kind {
	case token.INT, token.FLOAT:
		return true
	default:
		return false
	}
}

func refineVar(state lattice.AbstractState, name string, constraint lattice.ValueKind) lattice.AbstractState {
	if state == nil {
		return nil
	}
	current := lattice.GetValue(state, name)
	refined := lattice.Meet(current, constraint)
	if refined == lattice.Bottom {
		return nil
	}
	if refined == current {
		return state
	}
	out := lattice.CloneState(state)
	lattice.SetValue(out, name, refined)
	return out
}
