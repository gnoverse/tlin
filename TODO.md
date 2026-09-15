# TODO Plan for Implementing and Validating MiniLogic v2.1

This TODO list is organized into implementation phases by increasing semantic complexity. Each phase includes concrete implementation tasks and a corresponding test suite to verify correctness. This plan reflects the v2.1 formal specification and focuses on safe early-return validation.

---

## Priority Fixes (Spec Alignment)

### P0: OpaqueCalls must preserve call order for expression calls

* [x] Track `CallExpr` evaluations in expressions (not just `CallStmt`) so `OpaqueCalls` enforces ordering and multiplicity across all calls.
  * Must guarantee: call sequences include expression-level calls in evaluation order.
  * Must guarantee: equivalence fails when expression call order differs or calls are duplicated/dropped.
  * Recommendation: centralize call capture in `EvalExpr` and return both value+calls, or thread call logs through expression evaluation.

### P1: Allow calls in early-return conditions under OpaqueCalls

* [x] Allow `cond` to contain calls as long as call order and multiplicity are preserved.
  * Must guarantee: call order and multiplicity are identical between original and rewritten forms.
  * Must guarantee: `panic(...)` is treated as a call and therefore participates in call-order checks.
  * Recommendation: rely on OpaqueCalls call-sequence equality for safety.

### P1: Support `var` declarations in if-init (or reject explicitly)

* [x] Extend AST conversion and evaluator to support `var` in `if init; cond {}` or explicitly mark as out-of-scope with a clear reason.
  * Must guarantee: `var` in `if init` is either modeled (scoped) or yields `Unknown` with a specific reason.
  * Must guarantee: scope cleanup preserves outer variables and hides init-scoped identifiers.
  * Recommendation: add a `VarDeclStmt` node and handle it alongside `DeclAssignStmt`.

### P2: Ensure Unknown stays Unknown in lint gate fallback

* [x] In `verifyWithMiniLogic`, avoid treating `Unknown` as verified-safe solely via structural normalization/flattening.
  * Must guarantee: rewrites outside scope remain `Unknown` and do not auto-apply.
  * Must guarantee: fallback normalization does not override an explicit `Unknown`.
  * Recommendation: only use structural fallback when both sides are already within scope and the evaluator did not return `Unknown`.

## Phase 1: Core Evaluation Engine (No Termination Handling)

### Implementation Tasks

* [x] Define AST and internal IR for minimal statements: assignment, sequencing, if-else
* [x] Define symbolic environment model (`Env`) as key-value map
* [x] Implement evaluator for:

  * [x] `x = e`
  * [x] `S1 ; S2`
  * [x] `if cond { S1 } else { S2 }`
* [x] Encode semantics for `eval(e, Env)`
* [x] Define base `Result = Continue(Env)` only
* [x] Implement basic statement equivalence: equality of `Env` outputs

### Tests

* [x] Unit: `x = 1` vs. `x = 1`
* [x] Unit: `x = 1; y = 2` vs. `x = 1; y = 2`
* [x] Negative: `x = 1` vs. `x = 2`
* [x] If-else equivalence: constant true/false branches
* [ ] Property-based: permutation invariance under `Env` equality

---

## Phase 2: Add Termination-Aware Result Types and Semantics

### Implementation Tasks

* [x] Extend `Result` with: `Return(Value?)`, `Break`, `ContinueLoop`
* [x] Implement short-circuit sequencing: stop on non-`Continue`
* [x] Extend statement evaluator with:

  * [x] `return e?`
  * [x] `break`
  * [x] `continue`
* [x] Update equivalence logic to compare `Result` values (with termination kind)

### Tests

* [x] Unit: `return 1` vs. `return 1`
* [x] Unit: `if cond { return 1 } else { return 2 }` vs. same
* [x] Negative: `return 1` vs. `return 2`
* [x] Sequencing: `return 1; x = 2` == `return 1`
* [x] Mismatch type: `return 1` vs. `x = 1`

---

## Phase 3: Control-Flow Normalization & Early-Return Equivalence

### Implementation Tasks

* [x] Normalize rewrites: flattening `if { return } else { S }` into `if { return }; S`
* [x] Add canonical pattern equivalence engine
* [x] Implement constraint check: all `if` branches must terminate
* [x] Handle pure condition check assumption (assumed for now)

### Tests

* [x] Rewrite: `if cond { return } else { x = 1 }` vs. `if cond { return }; x = 1`
* [x] Nested return: `if cond { if sub { return } else { x = 1 } }` equivalence
* [x] Incomplete rewrite: one branch missing `return` → not equivalent

---

## Phase 4: Scoped Initializer Support (`if init; cond {}`)

### Implementation Tasks

* [x] Parse and represent `init` expressions in conditional statements
* [x] Restrict scope of init-bound identifiers to branch only
* [x] Detect illegal reference to scoped `init` variable in later statements
* [x] Mark such rewrites as `Unknown`

### Tests

* [x] Equivalence: `if x := f(); cond { return } else { return }` == same
* [x] Scope violation: `if x := f(); cond { return } else { x = 1 }; x = 2` → `Unknown`
* [x] Shadowing: init var does not affect outer scope

---

## Phase 5: Function Call Policy (Opaque vs. Disallowed)

### Implementation Tasks

* [x] Add support for `call(...)` statements
* [x] Add `CallPolicy = OpaqueCalls | DisallowCalls`
* [x] Under `OpaqueCalls`, track call order and count
* [x] Reject equivalence if calls are reordered or duplicated
* [x] Under `DisallowCalls`, reject any call-containing block as `Unknown`

### Tests

* [x] Equivalence with same calls in order: pass under `OpaqueCalls`
* [x] Call in one branch only: fail equivalence
* [x] Disallowed calls: any presence → `Unknown`
* [x] Panic as call: does not terminate unless explicitly modeled

---

## Phase 6: Error Reporting, SMT Hook, and Edge Case Handling

### Implementation Tasks

* [x] Surface `Unknown` status with reason code
* [x] Hook SMT solver or symbolic engine for `eval(cond, Env)`
* [x] Encode equivalence conditions as `ite` terms
* [x] Add logging and debugging output per statement pair

### Tests

* [x] Log trace: successful proof vs. unknown vs. mismatch
* [x] Regression: verify known safe and unsafe early-return rewrites
* [x] Stress: long sequence of nested early-return blocks

---

## Optional Phase 7: Fuzzing and Soundness Checks

### Implementation Tasks

* [ ] Fuzz original vs. randomly permuted rewrite variants
* [ ] Use property-based testing to confirm: `equivalent → same output`
* [ ] Use mutation testing to confirm: `non-equivalent → detected`

### Tests

* [ ] Fuzz over all valid rewrites under `OpaqueCalls`
* [ ] Ensure no false positives: mutated programs fail equivalence
* [ ] Catch scope violations, reorderings, incorrect termination
