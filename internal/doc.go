// Package internal provides the core functionality for a Go-compatible linting tool.
//
// This package implements a flexible and extensible linting engine that can be used
// to analyze Go and Gno code for potential issues, style violations, and areas of improvement.
// It is designed to be easily extendable with custom lint rules while providing a set of
// default rules out of the box.
//
// Key components:
//
// Engine: The main linting engine that coordinates the linting process.
// It manages a collection of lint rules and applies them to the given source files.
//
// LintRule: An interface that defines the contract for all lint rules.
// Each lint rule must implement the Check method to analyze the code and return issues.
//
// Issue: Represents a single lint issue found in the code, including its location and description.
//
// SymbolTable: A data structure that keeps track of defined symbols across the codebase,
// helping to reduce false positives in certain lint rules.
//
// SourceCode: A simple structure to represent the content of a source file as a collection of lines.
//
// The package also includes several helper functions for file operations, running external tools,
// and managing temporary files during the linting process.
//
// Usage:
//
//	engine, err := internal.NewEngine("path/to/root/dir")
//	if err != nil {
//	    // handle error
//	}
//
//	// Optionally add custom rules
//	engine.AddRule(myCustomRule)
//
//	issues, err := engine.Run("path/to/file.go")
//	if err != nil {
//	    // handle error
//	}
//
//	// Process the found issues
//	for _, issue := range issues {
//	    fmt.Printf("Found issue: %s at %s\n", issue.Message, issue.Start)
//	}
//
// This package is intended for internal use within the linting tool and should not be
// imported by external packages.
package internal
