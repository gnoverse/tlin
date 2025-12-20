package minilogic

// CondSolver attempts to resolve a condition to a concrete boolean.
// It can be backed by a symbolic engine or SMT solver.
type CondSolver interface {
	Solve(cond Expr, env *Env) (BoolValue, bool)
}

// BasicCondSolver provides simple, sound condition deductions.
type BasicCondSolver struct{}

func (BasicCondSolver) Solve(cond Expr, env *Env) (BoolValue, bool) {
	if exprHasCall(cond) {
		return BoolValue{}, false
	}

	switch c := cond.(type) {
	case LiteralExpr:
		if b, ok := c.Val.(BoolValue); ok {
			return b, true
		}
	case VarExpr:
		if val := env.Get(c.Name); val != nil {
			if b, ok := val.(BoolValue); ok {
				return b, true
			}
		}
	case UnaryExpr:
		if c.Op == OpNot {
			solver := BasicCondSolver{}
			if v, ok := solver.Solve(c.Operand, env); ok {
				return BoolValue{Val: !v.Val}, true
			}
		}
	case BinaryExpr:
		switch c.Op {
		case OpEq, OpNeq:
			if exprEqual(c.Left, c.Right) {
				return BoolValue{Val: c.Op == OpEq}, true
			}
		}
	}

	return BoolValue{}, false
}

func exprEqual(a, b Expr) bool {
	switch left := a.(type) {
	case LiteralExpr:
		right, ok := b.(LiteralExpr)
		if !ok {
			return false
		}
		return left.Val.Equal(right.Val)
	case VarExpr:
		right, ok := b.(VarExpr)
		if !ok {
			return false
		}
		return left.Name == right.Name
	case BinaryExpr:
		right, ok := b.(BinaryExpr)
		if !ok {
			return false
		}
		if left.Op != right.Op {
			return false
		}
		return exprEqual(left.Left, right.Left) && exprEqual(left.Right, right.Right)
	case UnaryExpr:
		right, ok := b.(UnaryExpr)
		if !ok {
			return false
		}
		if left.Op != right.Op {
			return false
		}
		return exprEqual(left.Operand, right.Operand)
	case CallExpr:
		right, ok := b.(CallExpr)
		if !ok {
			return false
		}
		if left.Func != right.Func || len(left.Args) != len(right.Args) {
			return false
		}
		for i := range left.Args {
			if !exprEqual(left.Args[i], right.Args[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
