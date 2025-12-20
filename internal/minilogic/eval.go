package minilogic

// CallPolicy determines how function calls are handled.
type CallPolicy int

const (
	_ CallPolicy = iota
	// OpaqueCalls treats calls as opaque effects.
	// Calls always evaluate to Continue, but call order/multiplicity is tracked.
	OpaqueCalls
	// DisallowCalls rejects any block containing calls as Unknown.
	DisallowCalls
)

// ControlFlowMode determines how control flow is handled.
type ControlFlowMode int

const (
	_ ControlFlowMode = iota
	// NoTermination does not model termination statements.
	NoTermination
	// EarlyReturnAware models return, break, continue statements.
	EarlyReturnAware
)

// EvalConfig holds configuration for the evaluator.
type EvalConfig struct {
	CallPolicy      CallPolicy
	ControlFlowMode ControlFlowMode
	InLoopContext   bool // true if we're inside a loop
}

// DefaultConfig returns the default evaluation configuration.
func DefaultConfig() EvalConfig {
	return EvalConfig{
		CallPolicy:      OpaqueCalls,
		ControlFlowMode: EarlyReturnAware,
		InLoopContext:   false,
	}
}

// Evaluator evaluates expressions and statements.
type Evaluator struct {
	config EvalConfig
}

// NewEvaluator creates a new evaluator with the given configuration.
func NewEvaluator(config EvalConfig) *Evaluator {
	return &Evaluator{config: config}
}

// EvalExpr evaluates an expression in the given environment.
// Returns the resulting value.
func (ev *Evaluator) EvalExpr(expr Expr, env *Env) Value {
	switch e := expr.(type) {
	case LiteralExpr:
		return e.Val

	case VarExpr:
		val := env.Get(e.Name)
		if val == nil {
			// Variable not found, return symbolic value
			return SymbolicValue{Name: e.Name}
		}
		return val

	case BinaryExpr:
		left := ev.EvalExpr(e.Left, env)
		right := ev.EvalExpr(e.Right, env)
		return ev.evalBinary(e.Op, left, right)

	case UnaryExpr:
		operand := ev.EvalExpr(e.Operand, env)
		return ev.evalUnary(e.Op, operand)

	case CallExpr:
		// Evaluate arguments but return symbolic value
		// (calls are side-effecting and return unknown values)
		return SymbolicValue{Name: "call_" + e.Func}

	default:
		return SymbolicValue{Name: "unknown"}
	}
}

func exprHasCall(expr Expr) bool {
	switch e := expr.(type) {
	case CallExpr:
		return true
	case BinaryExpr:
		return exprHasCall(e.Left) || exprHasCall(e.Right)
	case UnaryExpr:
		return exprHasCall(e.Operand)
	default:
		return false
	}
}

func (ev *Evaluator) evalBinary(op BinaryOp, left, right Value) Value {
	// Handle symbolic values
	_, leftSym := left.(SymbolicValue)
	_, rightSym := right.(SymbolicValue)
	if leftSym || rightSym {
		return SymbolicValue{Name: "binary_result"}
	}

	switch op {
	case OpAdd:
		if l, ok := left.(IntValue); ok {
			if r, ok := right.(IntValue); ok {
				return IntValue{Val: l.Val + r.Val}
			}
		}
		if l, ok := left.(StringValue); ok {
			if r, ok := right.(StringValue); ok {
				return StringValue{Val: l.Val + r.Val}
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

	return SymbolicValue{Name: "binary_result"}
}

func (ev *Evaluator) evalUnary(op UnaryOp, operand Value) Value {
	if _, ok := operand.(SymbolicValue); ok {
		return SymbolicValue{Name: "unary_result"}
	}

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

	return SymbolicValue{Name: "unary_result"}
}

// IsTruthy returns true if the value is truthy.
func IsTruthy(v Value) bool {
	switch val := v.(type) {
	case BoolValue:
		return val.Val
	case IntValue:
		return val.Val != 0
	case StringValue:
		return val.Val != ""
	case NilValue:
		return false
	case SymbolicValue:
		// Symbolic values are treated as potentially truthy
		return true
	default:
		return true
	}
}

// IsKnownTrue returns true if the value is definitively true.
func IsKnownTrue(v Value) bool {
	if b, ok := v.(BoolValue); ok {
		return b.Val
	}
	return false
}

// IsKnownFalse returns true if the value is definitively false.
func IsKnownFalse(v Value) bool {
	if b, ok := v.(BoolValue); ok {
		return !b.Val
	}
	return false
}

// EvalStmt evaluates a statement in the given environment.
// Returns the result of execution.
func (ev *Evaluator) EvalStmt(stmt Stmt, env *Env) Result {
	return ev.evalStmt(stmt, env, nil)
}

func (ev *Evaluator) evalStmt(stmt Stmt, env *Env, calls []CallRecord) Result {
	switch s := stmt.(type) {
	case AssignStmt:
		if ev.config.CallPolicy == DisallowCalls && exprHasCall(s.Expr) {
			return UnknownResult()
		}
		val := ev.EvalExpr(s.Expr, env)
		newEnv := env.Clone()
		newEnv.Set(s.Var, val)
		return ContinueResultWithCalls(newEnv, calls)

	case DeclAssignStmt:
		if ev.config.CallPolicy == DisallowCalls && exprHasCall(s.Expr) {
			return UnknownResult()
		}
		val := ev.EvalExpr(s.Expr, env)
		newEnv := env.Clone()
		newEnv.Set(s.Var, val)
		return ContinueResultWithCalls(newEnv, calls)

	case SeqStmt:
		// Evaluate first statement
		r1 := ev.evalStmt(s.First, env, calls)
		// If not Continue, short-circuit and return the result
		if r1.Kind != ResultContinue {
			return r1
		}
		// Continue with second statement
		return ev.evalStmt(s.Second, r1.Env, r1.Calls)

	case BlockStmt:
		if len(s.Stmts) == 0 {
			return ContinueResultWithCalls(env, calls)
		}
		currentEnv := env
		currentCalls := calls
		for _, stmt := range s.Stmts {
			r := ev.evalStmt(stmt, currentEnv, currentCalls)
			if r.Kind != ResultContinue {
				return r
			}
			currentEnv = r.Env
			currentCalls = r.Calls
		}
		return ContinueResultWithCalls(currentEnv, currentCalls)

	case IfStmt:
		return ev.evalIfStmt(s, env, calls)

	case ReturnStmt:
		if ev.config.ControlFlowMode == NoTermination {
			return UnknownResult()
		}
		var val Value
		if s.Value != nil {
			if ev.config.CallPolicy == DisallowCalls && exprHasCall(s.Value) {
				return UnknownResult()
			}
			val = ev.EvalExpr(s.Value, env)
		} else {
			val = NilValue{}
		}
		return ReturnResult(val, calls)

	case BreakStmt:
		if ev.config.ControlFlowMode == NoTermination {
			return UnknownResult()
		}
		if !ev.config.InLoopContext {
			// break outside loop is invalid
			return UnknownResult()
		}
		return BreakResult(calls)

	case ContinueStmt:
		if ev.config.ControlFlowMode == NoTermination {
			return UnknownResult()
		}
		if !ev.config.InLoopContext {
			// continue outside loop is invalid
			return UnknownResult()
		}
		return ContinueLoopResult(calls)

	case CallStmt:
		return ev.evalCallStmt(s, env, calls)

	case NoopStmt:
		return ContinueResultWithCalls(env, calls)

	default:
		return UnknownResult()
	}
}

func (ev *Evaluator) evalIfStmt(s IfStmt, env *Env, calls []CallRecord) Result {
	workingEnv := env
	workingCalls := calls
	var initVarNames []string

	// Handle init statement
	if s.Init != nil {
		// Track variables declared in init (for scope cleanup later)
		if decl, ok := s.Init.(DeclAssignStmt); ok {
			initVarNames = append(initVarNames, decl.Var)
		}

		// Create a child scope for init-bound variables
		childEnv := NewChildEnv(env)
		initResult := ev.evalStmt(s.Init, childEnv, workingCalls)
		if initResult.Kind != ResultContinue {
			return initResult
		}
		workingEnv = initResult.Env
		workingCalls = initResult.Calls
	}

	// Evaluate condition
	if ev.config.CallPolicy == DisallowCalls && exprHasCall(s.Cond) {
		return UnknownResult()
	}
	condVal := ev.EvalExpr(s.Cond, workingEnv)

	// If condition is symbolic, we cannot determine which branch to take
	if _, ok := condVal.(SymbolicValue); ok {
		// For symbolic conditions, we need to analyze both branches
		// and return Unknown if they differ
		result := ev.evalSymbolicIf(s, workingEnv, workingCalls)
		return ev.cleanupInitVars(result, env, initVarNames)
	}

	// Execute appropriate branch
	var branchResult Result
	if IsTruthy(condVal) {
		branchResult = ev.evalStmt(s.Then, workingEnv, workingCalls)
	} else if s.Else != nil {
		branchResult = ev.evalStmt(s.Else, workingEnv, workingCalls)
	} else {
		// No else branch, condition is false: fall through with init effects
		// while removing init-scoped variables.
		return ev.cleanupInitVars(ContinueResultWithCalls(workingEnv, workingCalls), env, initVarNames)
	}

	// Clean up init-scoped variables from Continue results
	return ev.cleanupInitVars(branchResult, env, initVarNames)
}

// cleanupInitVars removes init-scoped variables from Continue results.
// This ensures Go's scoping rules are respected: variables declared in
// if-init are not visible after the if statement.
func (ev *Evaluator) cleanupInitVars(result Result, originalEnv *Env, initVarNames []string) Result {
	// Only clean up Continue results; terminating results don't expose the environment
	if result.Kind != ResultContinue || len(initVarNames) == 0 {
		return result
	}

	// Build a set for fast lookup
	initVars := make(map[string]bool)
	for _, name := range initVarNames {
		initVars[name] = true
	}

	// Create a new environment based on the original, then copy non-init changes
	newEnv := originalEnv.Clone()

	// Copy all variables from result.Env that are not init vars
	for _, key := range result.Env.Keys() {
		if !initVars[key] {
			newEnv.Set(key, result.Env.Get(key))
		}
	}

	return ContinueResultWithCalls(newEnv, result.Calls)
}

func (ev *Evaluator) evalSymbolicIf(s IfStmt, env *Env, calls []CallRecord) Result {
	// For symbolic conditions, evaluate both branches
	thenResult := ev.evalStmt(s.Then, env, calls)

	var elseResult Result
	if s.Else != nil {
		elseResult = ev.evalStmt(s.Else, env, calls)
	} else {
		elseResult = ContinueResultWithCalls(env, calls)
	}

	// If both branches have the same result kind and value, we can return that
	if thenResult.Equal(elseResult) {
		return thenResult
	}

	// Otherwise, return Unknown (we cannot determine the result)
	return UnknownResult()
}

func (ev *Evaluator) evalCallStmt(s CallStmt, env *Env, calls []CallRecord) Result {
	if ev.config.CallPolicy == DisallowCalls {
		return UnknownResult()
	}

	// OpaqueCalls: track the call but continue execution
	args := make([]Value, len(s.Call.Args))
	for i, arg := range s.Call.Args {
		args[i] = ev.EvalExpr(arg, env)
	}

	newCalls := append(calls, CallRecord{Func: s.Call.Func, Args: args})
	return ContinueResultWithCalls(env, newCalls)
}
