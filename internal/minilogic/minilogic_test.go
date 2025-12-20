package minilogic

import (
	"testing"
)

// =======================
// Core Evaluation Engine Tests
// =======================

func TestAssignmentEquivalence(t *testing.T) {
	ml := New()

	// x = 1 vs x = 1 should be equivalent
	s1 := Assign("x", IntLit(1))
	s2 := Assign("x", IntLit(1))

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestSequenceEquivalence(t *testing.T) {
	ml := New()

	// x = 1; y = 2 vs x = 1; y = 2 should be equivalent
	s1 := Seq(Assign("x", IntLit(1)), Assign("y", IntLit(2)))
	s2 := Seq(Assign("x", IntLit(1)), Assign("y", IntLit(2)))

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestAssignmentNotEquivalent(t *testing.T) {
	ml := New()

	// x = 1 vs x = 2 should not be equivalent
	s1 := Assign("x", IntLit(1))
	s2 := Assign("x", IntLit(2))

	report := ml.Verify(s1, s2)
	if report.Result != NotEquivalent {
		t.Errorf("Expected NotEquivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestIfElseConstantTrue(t *testing.T) {
	ml := New()

	// if true { x = 1 } else { x = 2 } is equivalent to x = 1
	s1 := If(BoolLit(true), Assign("x", IntLit(1)), Assign("x", IntLit(2)))
	s2 := Assign("x", IntLit(1))

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestIfElseConstantFalse(t *testing.T) {
	ml := New()

	// if false { x = 1 } else { x = 2 } is equivalent to x = 2
	s1 := If(BoolLit(false), Assign("x", IntLit(1)), Assign("x", IntLit(2)))
	s2 := Assign("x", IntLit(2))

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestExpressionEvaluation(t *testing.T) {
	ml := New()
	env := NewEnv()
	env.Set("x", IntValue{Val: 10})

	// Test arithmetic
	val := ml.EvaluateExpr(Binary(OpAdd, Var("x"), IntLit(5)), env)
	if iv, ok := val.(IntValue); !ok || iv.Val != 15 {
		t.Errorf("Expected 15, got %v", val)
	}

	// Test comparison
	val = ml.EvaluateExpr(Binary(OpGt, Var("x"), IntLit(5)), env)
	if bv, ok := val.(BoolValue); !ok || !bv.Val {
		t.Errorf("Expected true, got %v", val)
	}
}

// =======================
// Termination-Aware Tests
// =======================

func TestReturnEquivalence(t *testing.T) {
	ml := New()

	// return 1 vs return 1 should be equivalent
	s1 := Return(IntLit(1))
	s2 := Return(IntLit(1))

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestReturnNotEquivalent(t *testing.T) {
	ml := New()

	// return 1 vs return 2 should not be equivalent
	s1 := Return(IntLit(1))
	s2 := Return(IntLit(2))

	report := ml.Verify(s1, s2)
	if report.Result != NotEquivalent {
		t.Errorf("Expected NotEquivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestIfReturnEquivalence(t *testing.T) {
	ml := New()

	// if cond { return 1 } else { return 2 } vs same should be equivalent
	cond := BoolLit(true)
	s1 := If(cond, Return(IntLit(1)), Return(IntLit(2)))
	s2 := If(cond, Return(IntLit(1)), Return(IntLit(2)))

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestSequencingShortCircuit(t *testing.T) {
	ml := New()

	// return 1; x = 2 is equivalent to return 1 (short-circuit)
	s1 := Seq(Return(IntLit(1)), Assign("x", IntLit(2)))
	s2 := Return(IntLit(1))

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestReturnVsAssignNotEquivalent(t *testing.T) {
	ml := New()

	// return 1 vs x = 1 should not be equivalent (different result kinds)
	s1 := Return(IntLit(1))
	s2 := Assign("x", IntLit(1))

	report := ml.Verify(s1, s2)
	if report.Result != NotEquivalent {
		t.Errorf("Expected NotEquivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestBreakInLoop(t *testing.T) {
	ml := NewForLoopContext()

	// break vs break should be equivalent
	s1 := Break()
	s2 := Break()

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestContinueInLoop(t *testing.T) {
	ml := NewForLoopContext()

	// continue vs continue should be equivalent
	s1 := ContinueS()
	s2 := ContinueS()

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestBreakOutsideLoop(t *testing.T) {
	ml := New() // Not in loop context

	// break outside loop should return Unknown
	s1 := Break()
	s2 := Break()

	report := ml.Verify(s1, s2)
	if report.Result != Unknown {
		t.Errorf("Expected Unknown, got %v: %s", report.Result, report.Detail)
	}
}

// =======================
// Early-Return Equivalence Tests
// =======================

func TestEarlyReturnRewrite(t *testing.T) {
	ml := New()

	// Original: if cond { return v } else { S }
	// Rewritten: if cond { return v }; S
	cond := BoolLit(true)
	returnVal := IntLit(1)
	elseStmt := Assign("x", IntLit(2))

	report := ml.VerifyEarlyReturn(cond, returnVal, elseStmt)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestEarlyReturnRewriteFalseCondition(t *testing.T) {
	ml := New()

	// With false condition, both should execute the else/trailing statement
	cond := BoolLit(false)
	returnVal := IntLit(1)
	elseStmt := Assign("x", IntLit(2))

	report := ml.VerifyEarlyReturn(cond, returnVal, elseStmt)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestIfElseChainFlattening(t *testing.T) {
	ml := New()

	// if c1 { return v1 } else if c2 { return v2 } else { return v3 }
	// =>
	// if c1 { return v1 }; if c2 { return v2 }; return v3

	conditions := []Expr{BoolLit(false), BoolLit(true)}
	returns := []Expr{IntLit(1), IntLit(2)}
	fallback := IntLit(3)

	report := ml.VerifyIfElseChainFlattening(conditions, returns, fallback)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestNestedReturnEquivalence(t *testing.T) {
	ml := New()

	// if cond { if sub { return 1 } else { return 2 } } else { return 3 }
	// With cond=true, sub=true: returns 1
	cond := BoolLit(true)
	sub := BoolLit(true)

	s1 := If(cond, If(sub, Return(IntLit(1)), Return(IntLit(2))), Return(IntLit(3)))
	s2 := If(cond, If(sub, Return(IntLit(1)), Return(IntLit(2))), Return(IntLit(3)))

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestIncompleteRewriteNotEquivalent(t *testing.T) {
	ml := New()

	// If one branch doesn't terminate, rewrite changes semantics
	// Original: if true { x = 1 } else { return 2 }
	// "Rewritten": if true { x = 1 }; return 2

	// These are NOT equivalent because:
	// - Original with true: x = 1, continues
	// - Rewritten with true: x = 1, then return 2
	original := If(BoolLit(true), Assign("x", IntLit(1)), Return(IntLit(2)))
	rewritten := Seq(If(BoolLit(true), Assign("x", IntLit(1)), nil), Return(IntLit(2)))

	report := ml.Verify(original, rewritten)
	if report.Result != NotEquivalent {
		t.Errorf("Expected NotEquivalent, got %v: %s", report.Result, report.Detail)
	}
}

// =======================
// Scoped Initializer Tests
// =======================

func TestInitializerEquivalence(t *testing.T) {
	ml := New()

	// if x := 1; x > 0 { return x } else { return 0 }
	// Both branches return based on x, which is 1
	s1 := IfInit(
		DeclAssign("x", IntLit(1)),
		Binary(OpGt, Var("x"), IntLit(0)),
		Return(Var("x")),
		Return(IntLit(0)),
	)
	s2 := IfInit(
		DeclAssign("x", IntLit(1)),
		Binary(OpGt, Var("x"), IntLit(0)),
		Return(Var("x")),
		Return(IntLit(0)),
	)

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestShadowingInitVar(t *testing.T) {
	ml := New()
	env := NewEnv()
	env.Set("x", IntValue{Val: 100}) // Outer x

	// if x := 1; x > 0 { return x } else { return 0 }
	// The inner x (1) shadows the outer x (100)
	s := IfInit(
		DeclAssign("x", IntLit(1)),
		Binary(OpGt, Var("x"), IntLit(0)),
		Return(Var("x")),
		Return(IntLit(0)),
	)

	result := ml.Evaluate(s, env)
	if result.Kind != ResultReturn {
		t.Errorf("Expected Return, got %v", result.Kind)
	}
	if iv, ok := result.Value.(IntValue); !ok || iv.Val != 1 {
		t.Errorf("Expected return value 1, got %v", result.Value)
	}
}

func TestInitVarNotVisibleAfterIf(t *testing.T) {
	ml := New()

	// if x := 1; false {} should be equivalent to noop
	// because x is not visible after the if statement
	s1 := IfInit(
		DeclAssign("x", IntLit(1)),
		BoolLit(false),
		Assign("y", IntLit(2)), // then branch (not taken)
		nil,                    // no else
	)
	s2 := NoopStmt{}

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent (init var should not leak), got %v: %s", report.Result, report.Detail)
	}
}

func TestInitSideEffectsWithFalseNoElse(t *testing.T) {
	ml := New()
	env := NewEnv()
	env.Set("y", IntValue{Val: 1})

	// if y = 2; false {} should leave y updated to 2
	s := IfInit(
		Assign("y", IntLit(2)),
		BoolLit(false),
		NoopStmt{},
		nil,
	)

	result := ml.Evaluate(s, env)
	if result.Kind != ResultContinue {
		t.Errorf("Expected Continue, got %v", result.Kind)
	}
	if yVal := result.Env.Get("y"); yVal == nil {
		t.Error("y should be set")
	} else if iv, ok := yVal.(IntValue); !ok || iv.Val != 2 {
		t.Errorf("Expected y = 2, got %v", yVal)
	}
}

func TestInitVarScopeWithTrueBranch(t *testing.T) {
	ml := New()

	// if x := 1; true { y = x } should assign y = 1
	// but x should not be visible after the if
	env := NewEnv()
	s := IfInit(
		DeclAssign("x", IntLit(1)),
		BoolLit(true),
		Assign("y", Var("x")),
		nil,
	)

	result := ml.Evaluate(s, env)
	if result.Kind != ResultContinue {
		t.Errorf("Expected Continue, got %v", result.Kind)
	}

	// y should be set to 1
	if yVal := result.Env.Get("y"); yVal == nil {
		t.Error("y should be set")
	} else if iv, ok := yVal.(IntValue); !ok || iv.Val != 1 {
		t.Errorf("Expected y = 1, got %v", yVal)
	}

	// x should NOT be visible (init-scoped)
	if xVal := result.Env.Get("x"); xVal != nil {
		t.Errorf("x should not be visible after if, got %v", xVal)
	}
}

func TestInitVarDoesNotAffectOuterScope(t *testing.T) {
	ml := New()
	env := NewEnv()
	env.Set("x", IntValue{Val: 100}) // Outer x

	// if x := 1; true { y = x } should not modify outer x
	s := IfInit(
		DeclAssign("x", IntLit(1)),
		BoolLit(true),
		Assign("y", Var("x")),
		nil,
	)

	result := ml.Evaluate(s, env)
	if result.Kind != ResultContinue {
		t.Errorf("Expected Continue, got %v", result.Kind)
	}

	// Outer x should still be 100
	if xVal := result.Env.Get("x"); xVal == nil {
		t.Error("outer x should still exist")
	} else if iv, ok := xVal.(IntValue); !ok || iv.Val != 100 {
		t.Errorf("Expected outer x = 100, got %v", xVal)
	}

	// y should be 1 (from inner x)
	if yVal := result.Env.Get("y"); yVal == nil {
		t.Error("y should be set")
	} else if iv, ok := yVal.(IntValue); !ok || iv.Val != 1 {
		t.Errorf("Expected y = 1, got %v", yVal)
	}
}

// =======================
// Function Call Policy Tests
// =======================

func TestOpaqueCalls_SameOrder(t *testing.T) {
	config := EvalConfig{
		CallPolicy:      OpaqueCalls,
		ControlFlowMode: EarlyReturnAware,
	}
	ml := NewWithConfig(config)

	// f(); g() vs f(); g() should be equivalent
	s1 := Seq(Call("f"), Call("g"))
	s2 := Seq(Call("f"), Call("g"))

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestOpaqueCalls_DifferentOrder(t *testing.T) {
	config := EvalConfig{
		CallPolicy:      OpaqueCalls,
		ControlFlowMode: EarlyReturnAware,
	}
	ml := NewWithConfig(config)

	// f(); g() vs g(); f() should NOT be equivalent (different call order)
	s1 := Seq(Call("f"), Call("g"))
	s2 := Seq(Call("g"), Call("f"))

	report := ml.Verify(s1, s2)
	if report.Result != NotEquivalent {
		t.Errorf("Expected NotEquivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestOpaqueCalls_InOneBranch(t *testing.T) {
	config := EvalConfig{
		CallPolicy:      OpaqueCalls,
		ControlFlowMode: EarlyReturnAware,
	}
	ml := NewWithConfig(config)

	// if true { f() } else { } vs f() should be equivalent
	s1 := If(BoolLit(true), Call("f"), NoopStmt{})
	s2 := Call("f")

	report := ml.Verify(s1, s2)
	if report.Result != Equivalent {
		t.Errorf("Expected Equivalent, got %v: %s", report.Result, report.Detail)
	}
}

func TestDisallowCalls(t *testing.T) {
	config := EvalConfig{
		CallPolicy:      DisallowCalls,
		ControlFlowMode: EarlyReturnAware,
	}
	ml := NewWithConfig(config)

	// Any call should result in Unknown
	s1 := Call("f")
	s2 := Call("f")

	report := ml.Verify(s1, s2)
	if report.Result != Unknown {
		t.Errorf("Expected Unknown, got %v: %s", report.Result, report.Detail)
	}
}

func TestDisallowCallsInExpr(t *testing.T) {
	config := EvalConfig{
		CallPolicy:      DisallowCalls,
		ControlFlowMode: EarlyReturnAware,
	}
	ml := NewWithConfig(config)

	// Calls inside expressions should also result in Unknown
	s1 := Assign("x", CallExpr{Func: "f"})
	s2 := Assign("x", CallExpr{Func: "f"})

	report := ml.Verify(s1, s2)
	if report.Result != Unknown {
		t.Errorf("Expected Unknown, got %v: %s", report.Result, report.Detail)
	}
}

// =======================
// API Tests
// =======================

func TestBatchVerify(t *testing.T) {
	ml := New()

	transformations := []TransformationContext{
		{
			Original:    Assign("x", IntLit(1)),
			Transformed: Assign("x", IntLit(1)),
			RuleName:    "test1",
		},
		{
			Original:    Assign("x", IntLit(1)),
			Transformed: Assign("x", IntLit(2)),
			RuleName:    "test2",
		},
		{
			Original:    Return(IntLit(1)),
			Transformed: Return(IntLit(1)),
			RuleName:    "test3",
		},
	}

	report := ml.BatchVerify(transformations)

	if report.Total != 3 {
		t.Errorf("Expected 3 total, got %d", report.Total)
	}
	if report.Equivalent != 2 {
		t.Errorf("Expected 2 equivalent, got %d", report.Equivalent)
	}
	if report.NotEquivalent != 1 {
		t.Errorf("Expected 1 not equivalent, got %d", report.NotEquivalent)
	}
}

func TestSafetyHelpers(t *testing.T) {
	equiv := VerificationReport{Result: Equivalent}
	notEquiv := VerificationReport{Result: NotEquivalent}
	unknown := VerificationReport{Result: Unknown}

	if !IsSafeToApply(equiv) {
		t.Error("IsSafeToApply should return true for Equivalent")
	}
	if IsSafeToApply(notEquiv) {
		t.Error("IsSafeToApply should return false for NotEquivalent")
	}
	if IsSafeToApply(unknown) {
		t.Error("IsSafeToApply should return false for Unknown")
	}

	if ShouldWarn(equiv) {
		t.Error("ShouldWarn should return false for Equivalent")
	}
	if ShouldWarn(notEquiv) {
		t.Error("ShouldWarn should return false for NotEquivalent")
	}
	if !ShouldWarn(unknown) {
		t.Error("ShouldWarn should return true for Unknown")
	}

	if IsDefinitelyUnsafe(equiv) {
		t.Error("IsDefinitelyUnsafe should return false for Equivalent")
	}
	if !IsDefinitelyUnsafe(notEquiv) {
		t.Error("IsDefinitelyUnsafe should return true for NotEquivalent")
	}
	if IsDefinitelyUnsafe(unknown) {
		t.Error("IsDefinitelyUnsafe should return false for Unknown")
	}
}

// =======================
// Normalization Tests
// =======================

func TestNormalize_ConstantFolding(t *testing.T) {
	ml := New()

	// x = 1 + 2 should normalize to x = 3
	s := Assign("x", Binary(OpAdd, IntLit(1), IntLit(2)))
	normalized := ml.Normalize(s)

	if assign, ok := normalized.(AssignStmt); ok {
		if lit, ok := assign.Expr.(LiteralExpr); ok {
			if iv, ok := lit.Val.(IntValue); ok {
				if iv.Val != 3 {
					t.Errorf("Expected 3, got %d", iv.Val)
				}
			} else {
				t.Errorf("Expected IntValue, got %T", lit.Val)
			}
		} else {
			t.Errorf("Expected LiteralExpr, got %T", assign.Expr)
		}
	} else {
		t.Errorf("Expected AssignStmt, got %T", normalized)
	}
}

func TestNormalize_DoubleNegation(t *testing.T) {
	ml := New()

	// !!x should normalize to x
	s := Assign("result", Not(Not(Var("x"))))
	normalized := ml.Normalize(s)

	if assign, ok := normalized.(AssignStmt); ok {
		if v, ok := assign.Expr.(VarExpr); ok {
			if v.Name != "x" {
				t.Errorf("Expected variable x, got %s", v.Name)
			}
		} else {
			t.Errorf("Expected VarExpr after double negation elimination, got %T", assign.Expr)
		}
	}
}

func TestNormalize_BooleanSimplification(t *testing.T) {
	ml := New()

	// true && x should normalize to x
	s := Assign("result", And(BoolLit(true), Var("x")))
	normalized := ml.Normalize(s)

	if assign, ok := normalized.(AssignStmt); ok {
		if v, ok := assign.Expr.(VarExpr); ok {
			if v.Name != "x" {
				t.Errorf("Expected variable x, got %s", v.Name)
			}
		} else {
			t.Errorf("Expected VarExpr, got %T", assign.Expr)
		}
	}
}

func TestFlattenIfElseChain(t *testing.T) {
	ml := New()

	// if c1 { return 1 } else { if c2 { return 2 } else { return 3 } }
	original := If(
		Var("c1"),
		Return(IntLit(1)),
		If(
			Var("c2"),
			Return(IntLit(2)),
			Return(IntLit(3)),
		),
	)

	flattened := ml.FlattenIfElseChain(original)

	// Should be: if c1 { return 1 }; if c2 { return 2 }; return 3
	// Verify it's a sequence
	if _, ok := flattened.(SeqStmt); !ok {
		t.Errorf("Expected SeqStmt, got %T", flattened)
	}
}

// =======================
// Environment Tests
// =======================

func TestEnv_BasicOperations(t *testing.T) {
	env := NewEnv()

	// Test Set and Get
	env.Set("x", IntValue{Val: 42})
	val := env.Get("x")
	if iv, ok := val.(IntValue); !ok || iv.Val != 42 {
		t.Errorf("Expected 42, got %v", val)
	}

	// Test Get non-existent
	val = env.Get("y")
	if val != nil {
		t.Errorf("Expected nil for non-existent var, got %v", val)
	}
}

func TestEnv_Clone(t *testing.T) {
	env := NewEnv()
	env.Set("x", IntValue{Val: 1})

	clone := env.Clone()
	clone.Set("x", IntValue{Val: 2})

	// Original should be unchanged
	if iv, ok := env.Get("x").(IntValue); !ok || iv.Val != 1 {
		t.Errorf("Original env was modified")
	}

	// Clone should have new value
	if iv, ok := clone.Get("x").(IntValue); !ok || iv.Val != 2 {
		t.Errorf("Clone should have new value")
	}
}

func TestEnv_ChildScope(t *testing.T) {
	parent := NewEnv()
	parent.Set("x", IntValue{Val: 1})

	child := NewChildEnv(parent)
	child.Set("y", IntValue{Val: 2})

	// Child can see parent's variable
	if iv, ok := child.Get("x").(IntValue); !ok || iv.Val != 1 {
		t.Errorf("Child should see parent's x")
	}

	// Child has its own variable
	if iv, ok := child.Get("y").(IntValue); !ok || iv.Val != 2 {
		t.Errorf("Child should have y")
	}

	// Parent doesn't see child's variable
	if parent.Get("y") != nil {
		t.Errorf("Parent should not see child's y")
	}

	// Shadowing
	child.Set("x", IntValue{Val: 100})
	if iv, ok := child.Get("x").(IntValue); !ok || iv.Val != 100 {
		t.Errorf("Child should shadow parent's x")
	}
	if iv, ok := parent.Get("x").(IntValue); !ok || iv.Val != 1 {
		t.Errorf("Parent's x should be unchanged")
	}
}

func TestEnv_Equality(t *testing.T) {
	env1 := NewEnv()
	env1.Set("x", IntValue{Val: 1})
	env1.Set("y", IntValue{Val: 2})

	env2 := NewEnv()
	env2.Set("x", IntValue{Val: 1})
	env2.Set("y", IntValue{Val: 2})

	if !env1.Equal(env2) {
		t.Error("Environments with same bindings should be equal")
	}

	env2.Set("y", IntValue{Val: 3})
	if env1.Equal(env2) {
		t.Error("Environments with different bindings should not be equal")
	}
}

// =======================
// Value Tests
// =======================

func TestValue_Equality(t *testing.T) {
	tests := []struct {
		a, b  Value
		equal bool
	}{
		{IntValue{Val: 1}, IntValue{Val: 1}, true},
		{IntValue{Val: 1}, IntValue{Val: 2}, false},
		{BoolValue{Val: true}, BoolValue{Val: true}, true},
		{BoolValue{Val: true}, BoolValue{Val: false}, false},
		{StringValue{Val: "hello"}, StringValue{Val: "hello"}, true},
		{StringValue{Val: "hello"}, StringValue{Val: "world"}, false},
		{NilValue{}, NilValue{}, true},
		{IntValue{Val: 1}, BoolValue{Val: true}, false},
		{SymbolicValue{Name: "x"}, SymbolicValue{Name: "x"}, true},
		{SymbolicValue{Name: "x"}, SymbolicValue{Name: "y"}, false},
	}

	for _, tt := range tests {
		got := tt.a.Equal(tt.b)
		if got != tt.equal {
			t.Errorf("%v.Equal(%v) = %v, want %v", tt.a, tt.b, got, tt.equal)
		}
	}
}

// =======================
// Result Tests
// =======================

func TestResult_Equality(t *testing.T) {
	env1 := NewEnv()
	env1.Set("x", IntValue{Val: 1})

	env2 := NewEnv()
	env2.Set("x", IntValue{Val: 1})

	tests := []struct {
		a, b  Result
		equal bool
	}{
		{ContinueResult(env1), ContinueResult(env2), true},
		{ReturnResult(IntValue{Val: 1}, nil), ReturnResult(IntValue{Val: 1}, nil), true},
		{ReturnResult(IntValue{Val: 1}, nil), ReturnResult(IntValue{Val: 2}, nil), false},
		{BreakResult(nil), BreakResult(nil), true},
		{ContinueLoopResult(nil), ContinueLoopResult(nil), true},
		{BreakResult(nil), ContinueLoopResult(nil), false},
	}

	for _, tt := range tests {
		got := tt.a.Equal(tt.b)
		if got != tt.equal {
			t.Errorf("%v.Equal(%v) = %v, want %v", tt.a, tt.b, got, tt.equal)
		}
	}
}

// =======================
// Regression Tests
// =======================

func TestRegression_EarlyReturnWithElseStatements(t *testing.T) {
	ml := New()

	// This is a key pattern for early-return lint rules:
	// if err != nil { return err } else { doSomething() }
	// should be equivalent to:
	// if err != nil { return err }; doSomething()

	// Test with condition = true (return path)
	original := If(BoolLit(true), Return(IntLit(1)), Assign("x", IntLit(2)))
	rewritten := Seq(If(BoolLit(true), Return(IntLit(1)), nil), Assign("x", IntLit(2)))

	report := ml.Verify(original, rewritten)
	if report.Result != Equivalent {
		t.Errorf("Early return pattern should be equivalent (true case), got %v: %s", report.Result, report.Detail)
	}

	// Test with condition = false (else path)
	original = If(BoolLit(false), Return(IntLit(1)), Assign("x", IntLit(2)))
	rewritten = Seq(If(BoolLit(false), Return(IntLit(1)), nil), Assign("x", IntLit(2)))

	report = ml.Verify(original, rewritten)
	if report.Result != Equivalent {
		t.Errorf("Early return pattern should be equivalent (false case), got %v: %s", report.Result, report.Detail)
	}
}

func TestRegression_LongSequence(t *testing.T) {
	ml := New()

	// Test a long sequence of early-return blocks
	// if c1 { return 1 }; if c2 { return 2 }; if c3 { return 3 }; return 4
	s1 := Seq(
		If(BoolLit(false), Return(IntLit(1)), nil),
		Seq(
			If(BoolLit(false), Return(IntLit(2)), nil),
			Seq(
				If(BoolLit(true), Return(IntLit(3)), nil),
				Return(IntLit(4)),
			),
		),
	)

	// This should return 3 (third condition is true)
	result := ml.Evaluate(s1, NewEnv())
	if result.Kind != ResultReturn {
		t.Errorf("Expected Return, got %v", result.Kind)
	}
	if iv, ok := result.Value.(IntValue); !ok || iv.Val != 3 {
		t.Errorf("Expected return value 3, got %v", result.Value)
	}
}
