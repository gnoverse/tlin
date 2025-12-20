package minilogic

// Normalizer transforms statements into canonical forms for equivalence checking.
type Normalizer struct {
	config EvalConfig
}

// NewNormalizer creates a new normalizer.
func NewNormalizer(config EvalConfig) *Normalizer {
	return &Normalizer{config: config}
}

// Normalize transforms a statement into its canonical form.
// This helps detect structural equivalence between different syntactic forms.
func (n *Normalizer) Normalize(stmt Stmt) Stmt {
	return n.normalizeStmt(stmt)
}

func (n *Normalizer) normalizeStmt(stmt Stmt) Stmt {
	switch s := stmt.(type) {
	case AssignStmt:
		return AssignStmt{Var: s.Var, Expr: n.normalizeExpr(s.Expr)}

	case DeclAssignStmt:
		return DeclAssignStmt{Var: s.Var, Expr: n.normalizeExpr(s.Expr)}

	case SeqStmt:
		first := n.normalizeStmt(s.First)
		second := n.normalizeStmt(s.Second)

		// If first is a no-op, return second
		if _, ok := first.(NoopStmt); ok {
			return second
		}

		// If second is a no-op, return first
		if _, ok := second.(NoopStmt); ok {
			return first
		}

		return SeqStmt{First: first, Second: second}

	case BlockStmt:
		normalized := make([]Stmt, 0, len(s.Stmts))
		for _, st := range s.Stmts {
			norm := n.normalizeStmt(st)
			if _, ok := norm.(NoopStmt); !ok {
				normalized = append(normalized, norm)
			}
		}
		if len(normalized) == 0 {
			return NoopStmt{}
		}
		if len(normalized) == 1 {
			return normalized[0]
		}
		return BlockStmt{Stmts: normalized}

	case IfStmt:
		return n.normalizeIf(s)

	case ReturnStmt:
		if s.Value != nil {
			return ReturnStmt{Value: n.normalizeExpr(s.Value)}
		}
		return s

	case BreakStmt, ContinueStmt, NoopStmt:
		return s

	case CallStmt:
		return CallStmt{Call: n.normalizeCallExpr(s.Call)}

	default:
		return stmt
	}
}

func (n *Normalizer) normalizeIf(s IfStmt) Stmt {
	var init Stmt
	if s.Init != nil {
		init = n.normalizeStmt(s.Init)
	}

	cond := n.normalizeExpr(s.Cond)
	then := n.normalizeStmt(s.Then)

	var els Stmt
	if s.Else != nil {
		els = n.normalizeStmt(s.Else)
	}

	// Constant folding for conditions
	if lit, ok := cond.(LiteralExpr); ok {
		if b, ok := lit.Val.(BoolValue); ok {
			if b.Val {
				// if true { S1 } else { S2 } => S1
				if init != nil {
					return SeqStmt{First: init, Second: then}
				}
				return then
			}
			// if false { S1 } else { S2 } => S2
			if els != nil {
				if init != nil {
					return SeqStmt{First: init, Second: els}
				}
				return els
			}
			// if false { S1 } => noop
			if init != nil {
				return init
			}
			return NoopStmt{}
		}
	}

	return IfStmt{Init: init, Cond: cond, Then: then, Else: els}
}

func (n *Normalizer) normalizeExpr(expr Expr) Expr {
	switch e := expr.(type) {
	case LiteralExpr:
		return e

	case VarExpr:
		return e

	case BinaryExpr:
		left := n.normalizeExpr(e.Left)
		right := n.normalizeExpr(e.Right)

		// Constant folding
		if llit, lok := left.(LiteralExpr); lok {
			if rlit, rok := right.(LiteralExpr); rok {
				if result := n.evalConstBinary(e.Op, llit.Val, rlit.Val); result != nil {
					return LiteralExpr{Val: result}
				}
			}
		}

		// Boolean simplifications
		if e.Op == OpAnd {
			// false && x => false
			if llit, ok := left.(LiteralExpr); ok {
				if b, ok := llit.Val.(BoolValue); ok && !b.Val {
					return LiteralExpr{Val: BoolValue{Val: false}}
				}
			}
			// x && false => false
			if rlit, ok := right.(LiteralExpr); ok {
				if b, ok := rlit.Val.(BoolValue); ok && !b.Val {
					return LiteralExpr{Val: BoolValue{Val: false}}
				}
			}
			// true && x => x
			if llit, ok := left.(LiteralExpr); ok {
				if b, ok := llit.Val.(BoolValue); ok && b.Val {
					return right
				}
			}
			// x && true => x
			if rlit, ok := right.(LiteralExpr); ok {
				if b, ok := rlit.Val.(BoolValue); ok && b.Val {
					return left
				}
			}
		}

		if e.Op == OpOr {
			// true || x => true
			if llit, ok := left.(LiteralExpr); ok {
				if b, ok := llit.Val.(BoolValue); ok && b.Val {
					return LiteralExpr{Val: BoolValue{Val: true}}
				}
			}
			// x || true => true
			if rlit, ok := right.(LiteralExpr); ok {
				if b, ok := rlit.Val.(BoolValue); ok && b.Val {
					return LiteralExpr{Val: BoolValue{Val: true}}
				}
			}
			// false || x => x
			if llit, ok := left.(LiteralExpr); ok {
				if b, ok := llit.Val.(BoolValue); ok && !b.Val {
					return right
				}
			}
			// x || false => x
			if rlit, ok := right.(LiteralExpr); ok {
				if b, ok := rlit.Val.(BoolValue); ok && !b.Val {
					return left
				}
			}
		}

		return BinaryExpr{Op: e.Op, Left: left, Right: right}

	case UnaryExpr:
		operand := n.normalizeExpr(e.Operand)

		// Constant folding
		if lit, ok := operand.(LiteralExpr); ok {
			if result := n.evalConstUnary(e.Op, lit.Val); result != nil {
				return LiteralExpr{Val: result}
			}
		}

		// Double negation elimination
		if e.Op == OpNot {
			if inner, ok := operand.(UnaryExpr); ok && inner.Op == OpNot {
				return inner.Operand
			}
		}

		return UnaryExpr{Op: e.Op, Operand: operand}

	case CallExpr:
		return n.normalizeCallExpr(&e)

	default:
		return expr
	}
}

func (n *Normalizer) normalizeCallExpr(c *CallExpr) *CallExpr {
	args := make([]Expr, len(c.Args))
	for i, arg := range c.Args {
		args[i] = n.normalizeExpr(arg)
	}
	return &CallExpr{Func: c.Func, Args: args}
}

func (n *Normalizer) evalConstBinary(op BinaryOp, left, right Value) Value {
	switch op {
	case OpAdd:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				return IntValue{Val: l.Val + r.Val}
			}
		}
	case OpSub:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				return IntValue{Val: l.Val - r.Val}
			}
		}
	case OpMul:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				return IntValue{Val: l.Val * r.Val}
			}
		}
	case OpDiv:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				if r.Val != 0 {
					return IntValue{Val: l.Val / r.Val}
				}
			}
		}
	case OpMod:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				if r.Val != 0 {
					return IntValue{Val: l.Val % r.Val}
				}
			}
		}
	case OpEq:
		return BoolValue{Val: left.Equal(right)}
	case OpNeq:
		return BoolValue{Val: !left.Equal(right)}
	case OpLt:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				return BoolValue{Val: l.Val < r.Val}
			}
		}
	case OpLte:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				return BoolValue{Val: l.Val <= r.Val}
			}
		}
	case OpGt:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				return BoolValue{Val: l.Val > r.Val}
			}
		}
	case OpGte:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				return BoolValue{Val: l.Val >= r.Val}
			}
		}
	case OpAnd:
		if l, ok := left.(BoolValue); ok {
			if r, ok := right.(BoolValue); ok {
				return BoolValue{Val: l.Val && r.Val}
			}
		}
	case OpOr:
		if l, ok := left.(BoolValue); ok {
			if r, ok := right.(BoolValue); ok {
				return BoolValue{Val: l.Val || r.Val}
			}
		}
	}
	return nil
}

func (n *Normalizer) evalConstUnary(op UnaryOp, operand Value) Value {
	switch op {
	case OpNot:
		if b, ok := operand.(BoolValue); ok {
			return BoolValue{Val: !b.Val}
		}
	case OpNeg:
		if i, ok := operand.(IntValue); ok {
			return IntValue{Val: -i.Val}
		}
	}
	return nil
}

// FlattenIfElseChain flattens a nested if-else chain into a sequence.
// This is the canonical transformation for early-return patterns.
//
// Input:
//
//	if c1 { return v1 }
//	else if c2 { return v2 }
//	else { S }
//
// Output:
//
//	if c1 { return v1 }
//	if c2 { return v2 }
//	S
func (n *Normalizer) FlattenIfElseChain(stmt Stmt) Stmt {
	ifStmt, ok := stmt.(IfStmt)
	if !ok {
		return stmt
	}

	// Check if then branch terminates
	if !n.branchTerminates(ifStmt.Then) {
		return stmt
	}

	// Build flattened sequence
	var result []Stmt

	// Add the initial if (without else)
	result = append(result, IfStmt{
		Init: ifStmt.Init,
		Cond: ifStmt.Cond,
		Then: ifStmt.Then,
		Else: nil,
	})

	// Process the else branch
	current := ifStmt.Else
	for current != nil {
		if nested, ok := current.(IfStmt); ok {
			if n.branchTerminates(nested.Then) {
				result = append(result, IfStmt{
					Init: nested.Init,
					Cond: nested.Cond,
					Then: nested.Then,
					Else: nil,
				})
				current = nested.Else
			} else {
				// Non-terminating then branch, stop flattening
				result = append(result, current)
				break
			}
		} else {
			// Not an if statement, add as final statement
			result = append(result, current)
			break
		}
	}

	return Seq(result...)
}

// branchTerminates checks if a statement definitely terminates (return, break, continue).
func (n *Normalizer) branchTerminates(stmt Stmt) bool {
	switch s := stmt.(type) {
	case ReturnStmt, BreakStmt, ContinueStmt:
		return true

	case BlockStmt:
		if len(s.Stmts) == 0 {
			return false
		}
		return n.branchTerminates(s.Stmts[len(s.Stmts)-1])

	case SeqStmt:
		// Check if any statement in the sequence terminates
		if n.branchTerminates(s.First) {
			return true
		}
		return n.branchTerminates(s.Second)

	case IfStmt:
		// If terminates if both branches terminate
		thenTerms := n.branchTerminates(s.Then)
		elseTerms := s.Else != nil && n.branchTerminates(s.Else)
		return thenTerms && elseTerms

	default:
		return false
	}
}

// UnflattenToIfElseChain converts a flat sequence back to nested if-else.
// This is the inverse of FlattenIfElseChain.
func (n *Normalizer) UnflattenToIfElseChain(stmt Stmt) Stmt {
	stmts := n.collectSequence(stmt)
	if len(stmts) == 0 {
		return NoopStmt{}
	}

	// Build nested if-else from the end
	result := stmts[len(stmts)-1]
	for i := len(stmts) - 2; i >= 0; i-- {
		if ifStmt, ok := stmts[i].(IfStmt); ok && ifStmt.Else == nil {
			result = IfStmt{
				Init: ifStmt.Init,
				Cond: ifStmt.Cond,
				Then: ifStmt.Then,
				Else: result,
			}
		} else {
			// Not an if statement or has else, cannot unflatten
			result = SeqStmt{First: stmts[i], Second: result}
		}
	}

	return result
}

func (n *Normalizer) collectSequence(stmt Stmt) []Stmt {
	var result []Stmt
	n.collectSeqHelper(stmt, &result)
	return result
}

func (n *Normalizer) collectSeqHelper(stmt Stmt, result *[]Stmt) {
	switch s := stmt.(type) {
	case SeqStmt:
		n.collectSeqHelper(s.First, result)
		n.collectSeqHelper(s.Second, result)
	case BlockStmt:
		for _, st := range s.Stmts {
			*result = append(*result, st)
		}
	default:
		*result = append(*result, stmt)
	}
}
