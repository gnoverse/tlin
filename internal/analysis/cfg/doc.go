// # Description
//
// Package cfg provides functionality to generate and analyze Control Frow Graph (CFG) for go-like grammar languages.
//
// ## Control Flow Graph (CFG)
//
// A CFG is a representation, using graph notation, of all paths that might be traversed
// through a program during its execution. In a CFG:
//
//   - Each node in the graph represents a basic block (a straight-line piece of code without any jumps).
//   - The directed edges represent jumps in the control flow.
//
// This package constructs CFGs for Go-like language functions, providing a powerful tool for various
// types of static analysis, including:
//
//   - Data flow analysis
//   - Dead code elimination
//   - Optimization opportunities identification
//   - Complex analysis
//
// ## Package Functionality
//
// The main features of this package include:
//
//  1. CFG Construction: Generate a CFG from AST (Abstract Syntax Tree) nodes.
//  2. Use the `FromFunc` or `Build` methods to construct a CFG from the AST.
//  3. Analyze the CFG using provided methods or traverse it from custom analysis.
package cfg
