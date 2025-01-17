/*
Package query provides a lexer and parser for processing Comby-style metavariable expressions
used in pattern matching and rewriting code.

# Overview

The query package implements a parser for Comby's metavariable syntax, which is used for
pattern matching and rewriting source code. It serves as the first parsing phase before
the main syntax parsing, handling metavariable expressions that can match arbitrary code
patterns.

# Metavariable Syntax

Metavariables in patterns are expressed using two forms:

 1. Short form: :[identifier]
    Example: :[var]

 2. Long form: :[[identifier]]
    Example: :[[function]]

These metavariables can be used in both match and rewrite patterns. When a pattern
is matched against source code, metavariables capture the corresponding text and can
be referenced in the rewrite pattern.

# Token Types

The lexer recognizes the following token types:

  - TokenText: Plain text content
    Example: "if", "return", etc.

  - TokenHole: Metavariable placeholders
    Format: ":[name]" or ":[[name]]"
    Example: ":[condition]", ":[[body]]"

  - TokenLBrace: Opening curly brace "{"
    Used for block structure

  - TokenRBrace: Closing curly brace "}"
    Used for block structure

  - TokenWhitespace: Spaces, tabs, newlines
    Preserved for accurate source mapping

  - TokenEOF: End of input marker

# AST Node Types

The parser produces an AST with the following node types:

  - PatternNode: Root node containing the entire pattern
    Children can be any other node type

  - HoleNode: Represents a metavariable
    Contains the identifier name

  - TextNode: Contains literal text content
    Includes whitespace when significant

  - BlockNode: Represents a curly brace enclosed block
    Contains child nodes between braces

# Usage Example

Basic usage of the lexer and parser:

	// Create a new lexer with input
	lexer := NewLexer("if :[condition] { :[[body]] }")

	// Tokenize the input
	tokens := lexer.Tokenize()

	// Create parser and generate AST
	parser := NewParser(tokens)
	ast := parser.Parse()

# Pattern Matching Rules

 1. Metavariables match greedily but cannot cross block boundaries
    Example: :[x] in "if :[x] {" will match everything up to the brace

 2. Block matching respects nested structures
    Matched content preserves all whitespace and formatting

 3. Whitespace is normalized in pattern matching
    Multiple spaces are treated as a single space

 4. Block boundaries must be explicit
    Curly braces must be present in the pattern to match blocks

This package is designed to work as the first phase of a multi-phase parsing system
where metavariable expressions are processed before deeper syntactic analysis.
It provides the foundation for implementing Comby-style pattern matching and
rewriting functionality.

For more details about Comby's pattern matching syntax, visit:
https://comby.dev/docs/syntax-reference
*/
package query
