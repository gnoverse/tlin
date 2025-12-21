# MiniLogic v2.1

## A Partial Formal Verification Framework for Go Lint Rewrites (Early-Return–Aware)

---

## 0. Revision Summary (v2 → v2.1)

MiniLogic v2.1 revises the v2 specification to better match the behavior of real Go/Gno lint rules, especially **early-return and conditional normalization rules**

Key changes:

1. **Explicit modeling of control-flow termination**
2. **Restricted support for `return`, `break`, and `continue`**
3. **Explicit semantics for `if init; cond { … } else { … }`**
4. **Clear scoping of which lint rules are eligible for verification**

The goal is to avoid returning `Unknown` for the very rewrites that linters actually perform, while still keeping the model intentionally small and sound.

---

## 1. Scope and Supported Lint Rules

### 1.1 Supported by MiniLogic

MiniLogic does **not** attempt to verify all lint rules.
Only the following classes of rewrites are within scope:

**Supported**

* if–else flattening
* early-return / early-continue / early-break normalization
* boolean condition restructuring in control flow
* merging or splitting conditional returns

**Out of Scope (always `Unknown`)**

* formatting and stylistic rewrites
* regex-based rewrites
* type-conversion simplifications
* reflection or interface-heavy rewrites
* loop transformations
* data-flow or performance-oriented optimizations

> **Policy**
> Rewrites outside the supported set must be reported as `Unknown`.
> Linters may still emit suggestions, butx must not auto-apply fixes.

---

## 2. Modeled Statement Fragment (Extended)

### 2.1 Statements

The statement language is extended to support early-return validation while remaining deliberately small.

Supported statements:

* Assignment

  ```
  x = e
  ```

* Sequencing

  ```
  S1 ; S2
  ```

* Conditional

  ```
  if [init]; cond { S1 } else { S2 }
  ```

* Termination statements

  ```
  return e?
  break
  continue
  ```

* Opaque statement (optional)

  ```
  call(...)
  ```

Excluded:

* loops (beyond recognizing `break` / `continue`)
* `defer`
* goroutines and channels
* heap mutation or aliasing
* full CFG or interprocedural analysis

---

## 3. Termination-Aware Semantic Model

### 3.1 Result Type

Statement evaluation produces a **termination-aware result**:

```
Result =
  Continue(Env)
| Return(Value?)
| Break
| ContinueLoop
| ⊥        // undefined / stuck (optional)
```

This allows equivalence to account for **how execution ends**, not just final environments.

---

### 3.2 Statement Semantics

Let `⟦S⟧ : Env → Result`.

#### Assignment

```
⟦x = e⟧(σ) = Continue(σ[x ↦ eval(e, σ)])
```

#### Sequencing (termination-aware)

```
⟦S1 ; S2⟧(σ) =
  match ⟦S1⟧(σ) with
    Continue(σ') → ⟦S2⟧(σ')
    r            → r
```

Once a termination result is produced, subsequent statements are not evaluated.

---

#### Conditional with initializer

```
⟦if init; cond { S1 } else { S2 }⟧(σ) =
  let σ' = ⟦init⟧(σ) in
  if eval(cond, σ') == true
     then ⟦S1⟧(σ')
     else ⟦S2⟧(σ')
```

* `init` supports short variable declarations (`:=`) and `var` declarations.
* Variables introduced by `init` are scoped to the condition and both branches only.
* If those variables are referenced outside the conditional, the rewrite is invalid and must yield `Unknown`.

---

### 3.3 Termination Statements

```
⟦return e⟧(σ) = Return(eval(e, σ))
⟦return⟧(σ)   = Return(nil)
⟦break⟧(σ)    = Break
⟦continue⟧(σ) = ContinueLoop
```

* `break` / `continue` are only well-formed inside loop contexts.
* If they appear outside a loop, the rewrite is invalid and must yield `Unknown`.

---

### 3.4 Panic and Calls (Policy)

* `panic(...)` is treated as a function call.
* Function calls are **not modeled semantically**, but call order and multiplicity are tracked under OpaqueCalls.

Two policies are supported:

* **OpaqueCalls (default)**
  Calls always evaluate to `Continue`.
  Calls are treated as opaque effects on external state not modeled by `Env`.
  Equivalence under OpaqueCalls additionally requires preserving call order and multiplicity.

* **DisallowCalls**
  Any block containing calls is outside the model and yields `Unknown`.

This mirrors the behavior of existing early-return lints, which do not treat `panic` as guaranteed termination.

---

## 4. Observational Equivalence (Termination-Aware)

### 4.1 Statement Equivalence

Two statements `S1` and `S2` are equivalent iff:

```
∀σ. ⟦S1⟧(σ) = ⟦S2⟧(σ)
```

Result equality is defined as:

| Result kind  | Equality condition    |
| ------------ | --------------------- |
| Continue     | Environments equal    |
| Return       | Returned values equal |
| Break        | Both Break            |
| ContinueLoop | Both ContinueLoop     |
| Mixed kinds  | Not equivalent        |

---

## 5. Early-Return Rewrite (Normative Rule)

### 5.1 Canonical Patterns

Original:

```go
if cond {
    return v
} else {
    S
}
```

Rewritten:

```go
if cond {
    return v
}
S
```

Additional pattern (else-if chain flattening):

```go
if c1 { return v1 }
else if c2 { return v2 }
else { return v3 }
```

rewritten as:

```go
if c1 { return v1 }
if c2 { return v2 }
return v3
```

---

### 5.2 Soundness Conditions

The rewrite is valid only if all of the following hold:

1. Under OpaqueCalls, `cond` may contain calls, but the rewrite must not change call order or multiplicity.
2. Every `if` body in the chain terminates (e.g., `return`, `break`, `continue`).
3. No `init`-scoped identifiers are referenced outside their `if` statement.
4. Under OpaqueCalls, the rewrite does not reorder or duplicate calls.

Failure to establish any condition results in `Unknown`.

---

## 6. Function Calls and Side Effects

### 6.1 Default Policy

MiniLogic v2.1 defaults to:

```
CallPolicy = OpaqueCalls
```

This allows early-return validation in blocks that contain calls, while preventing unsafe reordering.

---

## 7. SMT and Decision Procedures (Overview)

MiniLogic does not require full CFG encoding.

Two implementation strategies are permitted:

* **Symbolic Result Encoding**
  Encode termination kind and value explicitly in SMT.

* **Control-Normalization Strategy (recommended for PoC)**
  Rewrite statements into a single-exit normal form and reduce equivalence to expression equality using `ite`.

Either strategy must be sound with respect to the semantics in Section 3.

---

## 8. Configuration Extensions

Additional configuration flags:

```go
type ControlFlowMode int
const (
  NoTermination
  EarlyReturnAware
)

type CallPolicy int
const (
  DisallowCalls
  OpaqueCalls
)
```

For early-return validation:

```
ControlFlowMode = EarlyReturnAware
CallPolicy      = OpaqueCalls
```

---

## 9. Revised Non-Goals

MiniLogic v2.1 explicitly does **not** model:

* full loop semantics
* `defer`
* panic propagation
* goroutines or channels
* heap aliasing or memory models
* interprocedural control flow

These are unnecessary for validating the targeted lint rules.
