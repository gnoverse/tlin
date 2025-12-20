package minilogic

// VerificationResult represents the result of equivalence verification.
type VerificationResult int

const (
	_ VerificationResult = iota
	// Equivalent indicates the two statements are semantically equivalent.
	Equivalent
	// NotEquivalent indicates the two statements are not equivalent.
	NotEquivalent
	// Unknown indicates equivalence cannot be determined.
	Unknown
)

func (r VerificationResult) String() string {
	switch r {
	case Equivalent:
		return "Equivalent"
	case NotEquivalent:
		return "NotEquivalent"
	case Unknown:
		return "Unknown"
	default:
		return "?"
	}
}

// ReasonCode provides a reason for the verification result.
type ReasonCode int

const (
	ReasonNone ReasonCode = iota
	ReasonSameResult
	ReasonDifferentKind
	ReasonDifferentEnv
	ReasonDifferentValue
	ReasonDifferentCalls
	ReasonSymbolicCondition
	ReasonOutOfScope
	ReasonInvalidBreakContinue
	ReasonCallsDisallowed
	ReasonScopeViolation
)

func (r ReasonCode) String() string {
	switch r {
	case ReasonNone:
		return "none"
	case ReasonSameResult:
		return "same result for all environments"
	case ReasonDifferentKind:
		return "different result kinds"
	case ReasonDifferentEnv:
		return "different environments"
	case ReasonDifferentValue:
		return "different return values"
	case ReasonDifferentCalls:
		return "different call sequences"
	case ReasonSymbolicCondition:
		return "symbolic condition - cannot determine branch"
	case ReasonOutOfScope:
		return "rewrite outside MiniLogic scope"
	case ReasonInvalidBreakContinue:
		return "break/continue outside loop context"
	case ReasonCallsDisallowed:
		return "function calls not allowed"
	case ReasonScopeViolation:
		return "init-scoped variable used outside scope"
	default:
		return "unknown"
	}
}

// VerificationReport provides detailed information about verification.
type VerificationReport struct {
	Result VerificationResult
	Reason ReasonCode
	Detail string
	IR     *IRReport
}

// Verifier verifies the equivalence of statement transformations.
type Verifier struct {
	evaluator *Evaluator
	config    EvalConfig
}

// NewVerifier creates a new verifier with the given configuration.
func NewVerifier(config EvalConfig) *Verifier {
	return &Verifier{
		evaluator: NewEvaluator(config),
		config:    config,
	}
}

// CheckEquivalence checks if two statements are semantically equivalent.
// It evaluates both statements with the same input environment and
// compares the results.
func (v *Verifier) CheckEquivalence(s1, s2 Stmt) VerificationReport {
	return v.CheckEquivalenceWithEnv(s1, s2, NewEnv())
}

// CheckEquivalenceWithEnv checks equivalence given an initial environment.
func (v *Verifier) CheckEquivalenceWithEnv(s1, s2 Stmt, env *Env) VerificationReport {
	// Pre-check: validate both statements are within scope
	if !v.isInScope(s1) || !v.isInScope(s2) {
		return v.withDebugIR(VerificationReport{
			Result: Unknown,
			Reason: ReasonOutOfScope,
			Detail: "statement contains constructs outside MiniLogic scope",
		}, s1, s2, env)
	}

	// Check for scope violations (init-scoped variables used outside)
	if violation := v.checkScopeViolation(s1); violation != "" {
		return v.withDebugIR(VerificationReport{
			Result: Unknown,
			Reason: ReasonScopeViolation,
			Detail: violation,
		}, s1, s2, env)
	}
	if violation := v.checkScopeViolation(s2); violation != "" {
		return v.withDebugIR(VerificationReport{
			Result: Unknown,
			Reason: ReasonScopeViolation,
			Detail: violation,
		}, s1, s2, env)
	}

	// Evaluate both statements
	r1 := v.evaluator.EvalStmt(s1, env)
	r2 := v.evaluator.EvalStmt(s2, env)

	// Check for Unknown results
	if r1.Kind == ResultUnknown || r2.Kind == ResultUnknown {
		return v.withDebugIR(VerificationReport{
			Result: Unknown,
			Reason: ReasonSymbolicCondition,
			Detail: "evaluation produced unknown result",
		}, s1, s2, env)
	}

	// Compare result kinds
	if r1.Kind != r2.Kind {
		return v.withDebugIR(VerificationReport{
			Result: NotEquivalent,
			Reason: ReasonDifferentKind,
			Detail: "result kinds differ: " + r1.Kind.String() + " vs " + r2.Kind.String(),
		}, s1, s2, env)
	}

	// Compare based on kind
	switch r1.Kind {
	case ResultContinue:
		if !r1.Env.Equal(r2.Env) {
			return v.withDebugIR(VerificationReport{
				Result: NotEquivalent,
				Reason: ReasonDifferentEnv,
				Detail: "environments differ: " + r1.Env.String() + " vs " + r2.Env.String(),
			}, s1, s2, env)
		}

	case ResultReturn:
		if r1.Value == nil && r2.Value == nil {
			// Both nil, ok
		} else if r1.Value == nil || r2.Value == nil {
			return v.withDebugIR(VerificationReport{
				Result: NotEquivalent,
				Reason: ReasonDifferentValue,
				Detail: "return values differ (nil vs non-nil)",
			}, s1, s2, env)
		} else if !r1.Value.Equal(r2.Value) {
			return v.withDebugIR(VerificationReport{
				Result: NotEquivalent,
				Reason: ReasonDifferentValue,
				Detail: "return values differ: " + r1.Value.String() + " vs " + r2.Value.String(),
			}, s1, s2, env)
		}

	case ResultBreak, ResultContinueLoop:
		// Kind match is sufficient
	}

	// Compare call sequences (for OpaqueCalls policy)
	if v.config.CallPolicy == OpaqueCalls {
		if !callSequencesEqual(r1.Calls, r2.Calls) {
			return v.withDebugIR(VerificationReport{
				Result: NotEquivalent,
				Reason: ReasonDifferentCalls,
				Detail: "call sequences differ",
			}, s1, s2, env)
		}
	}

	return v.withDebugIR(VerificationReport{
		Result: Equivalent,
		Reason: ReasonSameResult,
		Detail: "statements produce identical results",
	}, s1, s2, env)
}

func callSequencesEqual(a, b []CallRecord) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !callsEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// isInScope checks if a statement is within MiniLogic's modeling scope.
func (v *Verifier) isInScope(stmt Stmt) bool {
	switch s := stmt.(type) {
	case AssignStmt, DeclAssignStmt, DeclAssignMultiStmt, NoopStmt:
		return true
	case VarDeclStmt:
		return true

	case SeqStmt:
		return v.isInScope(s.First) && v.isInScope(s.Second)

	case BlockStmt:
		for _, st := range s.Stmts {
			if !v.isInScope(st) {
				return false
			}
		}
		return true

	case IfStmt:
		if s.Init != nil && !v.isInScope(s.Init) {
			return false
		}
		if !v.isInScope(s.Then) {
			return false
		}
		if s.Else != nil && !v.isInScope(s.Else) {
			return false
		}
		return true

	case ReturnStmt:
		return v.config.ControlFlowMode == EarlyReturnAware

	case BreakStmt, ContinueStmt:
		return v.config.ControlFlowMode == EarlyReturnAware && v.config.InLoopContext

	case CallStmt:
		return v.config.CallPolicy == OpaqueCalls

	default:
		return false
	}
}

// checkScopeViolation checks for init-scoped variables used outside their scope.
func (v *Verifier) checkScopeViolation(stmt Stmt) string {
	// Collect init-scoped variables
	initVars := make(map[string]bool)
	v.collectInitVars(stmt, initVars)

	// Check for violations
	return v.checkViolation(stmt, initVars, false, make(map[string]bool))
}

func (v *Verifier) collectInitVars(stmt Stmt, vars map[string]bool) {
	switch s := stmt.(type) {
	case SeqStmt:
		v.collectInitVars(s.First, vars)
		v.collectInitVars(s.Second, vars)

	case BlockStmt:
		for _, st := range s.Stmts {
			v.collectInitVars(st, vars)
		}

	case IfStmt:
		if s.Init != nil {
			switch decl := s.Init.(type) {
			case DeclAssignStmt:
				vars[decl.Var] = true
			case DeclAssignMultiStmt:
				for _, name := range decl.Vars {
					vars[name] = true
				}
			case VarDeclStmt:
				vars[decl.Var] = true
			}
		}
		v.collectInitVars(s.Then, vars)
		if s.Else != nil {
			v.collectInitVars(s.Else, vars)
		}
	}
}

func (v *Verifier) checkViolation(stmt Stmt, initVars map[string]bool, inIfScope bool, currentScopeVars map[string]bool) string {
	switch s := stmt.(type) {
	case AssignStmt:
		// Using init var outside if scope is a violation
		// But if it's in currentScopeVars, it's allowed in this scope
		if !inIfScope && initVars[s.Var] && !currentScopeVars[s.Var] {
			return "init-scoped variable '" + s.Var + "' assigned outside if scope"
		}
		// Check expression for variable references
		if ref := v.checkExprViolation(s.Expr, initVars, inIfScope, currentScopeVars); ref != "" {
			return ref
		}

	case DeclAssignStmt:
		if ref := v.checkExprViolation(s.Expr, initVars, inIfScope, currentScopeVars); ref != "" {
			return ref
		}

	case DeclAssignMultiStmt:
		if ref := v.checkExprViolation(s.Expr, initVars, inIfScope, currentScopeVars); ref != "" {
			return ref
		}

	case VarDeclStmt:
		if s.Expr != nil {
			if ref := v.checkExprViolation(s.Expr, initVars, inIfScope, currentScopeVars); ref != "" {
				return ref
			}
		}

	case SeqStmt:
		if ref := v.checkViolation(s.First, initVars, inIfScope, currentScopeVars); ref != "" {
			return ref
		}
		if ref := v.checkViolation(s.Second, initVars, inIfScope, currentScopeVars); ref != "" {
			return ref
		}

	case BlockStmt:
		for _, st := range s.Stmts {
			if ref := v.checkViolation(st, initVars, inIfScope, currentScopeVars); ref != "" {
				return ref
			}
		}

	case IfStmt:
		// Variables declared in init are scoped to this if statement
		localInitVars := make(map[string]bool)
		for k, val := range initVars {
			localInitVars[k] = val
		}

		// Track variables declared in this if's init
		localScopeVars := make(map[string]bool)
		if s.Init != nil {
			switch decl := s.Init.(type) {
			case DeclAssignStmt:
				localInitVars[decl.Var] = true
				localScopeVars[decl.Var] = true
			case DeclAssignMultiStmt:
				for _, name := range decl.Vars {
					localInitVars[name] = true
					localScopeVars[name] = true
				}
			case VarDeclStmt:
				localInitVars[decl.Var] = true
				localScopeVars[decl.Var] = true
			}
		}

		// Check condition - init vars are valid here
		if ref := v.checkExprViolation(s.Cond, initVars, true, localScopeVars); ref != "" {
			return ref
		}

		// Then and Else are within the if scope, init vars are valid
		if ref := v.checkViolation(s.Then, localInitVars, true, localScopeVars); ref != "" {
			return ref
		}
		if s.Else != nil {
			if ref := v.checkViolation(s.Else, localInitVars, true, localScopeVars); ref != "" {
				return ref
			}
		}

	case ReturnStmt:
		if s.Value != nil {
			if ref := v.checkExprViolation(s.Value, initVars, inIfScope, currentScopeVars); ref != "" {
				return ref
			}
		}

	case CallStmt:
		for _, arg := range s.Call.Args {
			if ref := v.checkExprViolation(arg, initVars, inIfScope, currentScopeVars); ref != "" {
				return ref
			}
		}
	}

	return ""
}

func (v *Verifier) checkExprViolation(expr Expr, initVars map[string]bool, inIfScope bool, currentScopeVars map[string]bool) string {
	switch e := expr.(type) {
	case VarExpr:
		// A variable is valid if:
		// 1. We're in if scope and it's in currentScopeVars (declared in this if's init)
		// 2. We're in if scope generally
		// 3. It's not an init-scoped variable
		if initVars[e.Name] && !inIfScope && !currentScopeVars[e.Name] {
			return "init-scoped variable '" + e.Name + "' referenced outside if scope"
		}

	case BinaryExpr:
		if ref := v.checkExprViolation(e.Left, initVars, inIfScope, currentScopeVars); ref != "" {
			return ref
		}
		if ref := v.checkExprViolation(e.Right, initVars, inIfScope, currentScopeVars); ref != "" {
			return ref
		}

	case UnaryExpr:
		if ref := v.checkExprViolation(e.Operand, initVars, inIfScope, currentScopeVars); ref != "" {
			return ref
		}

	case CallExpr:
		for _, arg := range e.Args {
			if ref := v.checkExprViolation(arg, initVars, inIfScope, currentScopeVars); ref != "" {
				return ref
			}
		}
	}

	return ""
}

// VerifyEarlyReturnRewrite verifies that an early-return rewrite is valid.
// Original:
//
//	if cond { return v } else { S }
//
// Rewritten:
//
//	if cond { return v }; S
func (v *Verifier) VerifyEarlyReturnRewrite(cond Expr, returnVal Expr, elseStmt Stmt) VerificationReport {
	// Original form
	original := If(cond, Return(returnVal), elseStmt)

	// Rewritten form
	rewritten := Seq(If(cond, Return(returnVal), nil), elseStmt)

	return v.CheckEquivalence(original, rewritten)
}

// VerifyIfElseChainFlattening verifies that if-else chain flattening is valid.
// Original:
//
//	if c1 { return v1 }
//	else if c2 { return v2 }
//	else { return v3 }
//
// Rewritten:
//
//	if c1 { return v1 }
//	if c2 { return v2 }
//	return v3
func (v *Verifier) VerifyIfElseChainFlattening(conditions []Expr, returns []Expr, fallback Expr) VerificationReport {
	if len(conditions) != len(returns) {
		return VerificationReport{
			Result: Unknown,
			Reason: ReasonOutOfScope,
			Detail: "mismatched conditions and returns",
		}
	}

	if len(conditions) == 0 {
		return VerificationReport{
			Result: Equivalent,
			Reason: ReasonSameResult,
			Detail: "empty chain",
		}
	}

	// Build original (nested if-else chain)
	var original Stmt
	original = Return(fallback)
	for i := len(conditions) - 1; i >= 0; i-- {
		original = If(conditions[i], Return(returns[i]), original)
	}

	// Build rewritten (flat if sequence)
	stmts := make([]Stmt, 0, len(conditions)+1)
	for i := 0; i < len(conditions); i++ {
		stmts = append(stmts, If(conditions[i], Return(returns[i]), nil))
	}
	stmts = append(stmts, Return(fallback))
	rewritten := Seq(stmts...)

	return v.CheckEquivalence(original, rewritten)
}
