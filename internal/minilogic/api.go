package minilogic

import "fmt"

// MiniLogic is the main entry point for the verification framework.
// It provides a high-level API for verifying lint rule transformations.
type MiniLogic struct {
	verifier   *Verifier
	normalizer *Normalizer
	config     EvalConfig
}

// New creates a new MiniLogic instance with default configuration.
func New() *MiniLogic {
	config := DefaultConfig()
	return &MiniLogic{
		verifier:   NewVerifier(config),
		normalizer: NewNormalizer(config),
		config:     config,
	}
}

// NewWithConfig creates a new MiniLogic instance with the given configuration.
func NewWithConfig(config EvalConfig) *MiniLogic {
	return &MiniLogic{
		verifier:   NewVerifier(config),
		normalizer: NewNormalizer(config),
		config:     config,
	}
}

// NewForLoopContext creates a MiniLogic instance for verifying
// transformations inside loop bodies.
func NewForLoopContext() *MiniLogic {
	config := EvalConfig{
		CallPolicy:      OpaqueCalls,
		ControlFlowMode: EarlyReturnAware,
		InLoopContext:   true,
	}
	return NewWithConfig(config)
}

// Verify checks if two statements are semantically equivalent.
func (m *MiniLogic) Verify(original, transformed Stmt) VerificationReport {
	return m.verifier.CheckEquivalence(original, transformed)
}

// VerifyWithEnv checks equivalence given an initial environment.
func (m *MiniLogic) VerifyWithEnv(original, transformed Stmt, env *Env) VerificationReport {
	return m.verifier.CheckEquivalenceWithEnv(original, transformed, env)
}

// VerifyNormalized normalizes both statements before checking equivalence.
// This can help detect equivalence between syntactically different but
// semantically identical statements.
func (m *MiniLogic) VerifyNormalized(original, transformed Stmt) VerificationReport {
	normOrig := m.normalizer.Normalize(original)
	normTrans := m.normalizer.Normalize(transformed)
	return m.verifier.CheckEquivalence(normOrig, normTrans)
}

// VerifyEarlyReturn verifies an early-return transformation.
//
// Original:
//
//	if cond { return v } else { S }
//
// Transformed:
//
//	if cond { return v }; S
func (m *MiniLogic) VerifyEarlyReturn(cond Expr, returnVal Expr, elseStmt Stmt) VerificationReport {
	return m.verifier.VerifyEarlyReturnRewrite(cond, returnVal, elseStmt)
}

// VerifyIfElseChainFlattening verifies if-else chain flattening.
//
// Original:
//
//	if c1 { return v1 } else if c2 { return v2 } else { return v3 }
//
// Transformed:
//
//	if c1 { return v1 }; if c2 { return v2 }; return v3
func (m *MiniLogic) VerifyIfElseChainFlattening(conditions []Expr, returns []Expr, fallback Expr) VerificationReport {
	return m.verifier.VerifyIfElseChainFlattening(conditions, returns, fallback)
}

// Normalize transforms a statement into its canonical form.
func (m *MiniLogic) Normalize(stmt Stmt) Stmt {
	return m.normalizer.Normalize(stmt)
}

// FlattenIfElseChain flattens a nested if-else chain.
func (m *MiniLogic) FlattenIfElseChain(stmt Stmt) Stmt {
	return m.normalizer.FlattenIfElseChain(stmt)
}

// UnflattenToIfElseChain converts a flat sequence to nested if-else.
func (m *MiniLogic) UnflattenToIfElseChain(stmt Stmt) Stmt {
	return m.normalizer.UnflattenToIfElseChain(stmt)
}

// Evaluate evaluates a statement and returns the result.
func (m *MiniLogic) Evaluate(stmt Stmt, env *Env) Result {
	ev := NewEvaluator(m.config)
	return ev.EvalStmt(stmt, env)
}

// EvaluateExpr evaluates an expression and returns the value.
func (m *MiniLogic) EvaluateExpr(expr Expr, env *Env) Value {
	ev := NewEvaluator(m.config)
	return ev.EvalExpr(expr, env)
}

// TransformationContext provides information about a transformation for verification.
type TransformationContext struct {
	Original    Stmt
	Transformed Stmt
	RuleName    string
	Location    string // file:line information
}

// BatchVerify verifies multiple transformations and returns a summary.
func (m *MiniLogic) BatchVerify(transformations []TransformationContext) BatchVerificationReport {
	report := BatchVerificationReport{
		Total:         len(transformations),
		Equivalent:    0,
		NotEquivalent: 0,
		Unknown:       0,
		Results:       make([]TransformationResult, len(transformations)),
	}

	for i, t := range transformations {
		vr := m.Verify(t.Original, t.Transformed)
		report.Results[i] = TransformationResult{
			Context: t,
			Report:  vr,
		}

		switch vr.Result {
		case Equivalent:
			report.Equivalent++
		case NotEquivalent:
			report.NotEquivalent++
		case Unknown:
			report.Unknown++
		}
	}

	return report
}

// TransformationResult holds the result of verifying a single transformation.
type TransformationResult struct {
	Context TransformationContext
	Report  VerificationReport
}

// BatchVerificationReport summarizes the results of batch verification.
type BatchVerificationReport struct {
	Total         int
	Equivalent    int
	NotEquivalent int
	Unknown       int
	Results       []TransformationResult
}

// Summary returns a human-readable summary of the batch verification.
func (r BatchVerificationReport) Summary() string {
	return fmt.Sprintf(
		"Verified %d transformations: %d equivalent, %d not equivalent, %d unknown",
		r.Total, r.Equivalent, r.NotEquivalent, r.Unknown,
	)
}

// SafeTransformations returns only the verified-safe transformations.
func (r BatchVerificationReport) SafeTransformations() []TransformationResult {
	safe := make([]TransformationResult, 0)
	for _, res := range r.Results {
		if res.Report.Result == Equivalent {
			safe = append(safe, res)
		}
	}
	return safe
}

// UnsafeTransformations returns the transformations that failed verification.
func (r BatchVerificationReport) UnsafeTransformations() []TransformationResult {
	unsafe := make([]TransformationResult, 0)
	for _, res := range r.Results {
		if res.Report.Result == NotEquivalent {
			unsafe = append(unsafe, res)
		}
	}
	return unsafe
}

// UnknownTransformations returns transformations with unknown equivalence.
func (r BatchVerificationReport) UnknownTransformations() []TransformationResult {
	unknown := make([]TransformationResult, 0)
	for _, res := range r.Results {
		if res.Report.Result == Unknown {
			unknown = append(unknown, res)
		}
	}
	return unknown
}

// IsSafeToApply determines if a transformation can be safely auto-applied.
// Returns true only if the transformation is verified as equivalent.
func IsSafeToApply(report VerificationReport) bool {
	return report.Result == Equivalent
}

// ShouldWarn determines if a warning should be emitted for a transformation.
// Returns true for Unknown results (cannot verify safety).
func ShouldWarn(report VerificationReport) bool {
	return report.Result == Unknown
}

// IsDefinitelyUnsafe determines if a transformation is definitely incorrect.
// Returns true for NotEquivalent results.
func IsDefinitelyUnsafe(report VerificationReport) bool {
	return report.Result == NotEquivalent
}
