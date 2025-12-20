// Package minilogic implements a partial, heuristic verifier for
// Go AST-based lint rewrites.
//
// MiniLogic is designed to check semantic equivalence for a restricted
// statement fragment (not full Go). It is not a formal verification
// engine: it uses normalization and a limited abstract interpreter,
// and it can return Unknown for symbolic conditions or unsupported
// constructs.
//
// Limitations (non-exhaustive):
//   - Not full Go AST; no loops, switch, defer, goto, or closures.
//   - No memory model, aliasing, pointers, or heap effects.
//   - Calls are opaque (no side-effect modeling beyond call order).
//   - Equivalence is inferred via evaluation, not universal proof.
//
// Supported lint rule transformations:
//   - if-else flattening
//   - early-return / early-continue / early-break normalization
//   - boolean condition restructuring in control flow
//   - merging or splitting conditional returns
//
// Out of scope (returns Unknown):
//   - formatting and stylistic rewrites
//   - type-conversion simplifications
//   - reflection or interface-heavy rewrites
//   - loop transformations
//   - data-flow or performance-oriented optimizations
package minilogic
