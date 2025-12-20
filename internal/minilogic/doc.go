// Package minilogic implements a partial formal verification framework
// for Go AST-based lint rewrites.
//
// MiniLogic is designed to verify the semantic equivalence of lint rule
// transformations, particularly early-return and conditional normalization
// rewrites. It provides a termination-aware semantic model that can
// prove or disprove the equivalence of statement blocks.
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
