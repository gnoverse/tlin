package fixerv2

import "fmt"

// Node represents a parsed pattern element that can be either a literal or a metavariable
type Node interface {
	String() string
}

// LiteralNode represents a literal string value in the pattern
type LiteralNode struct {
	Value string
}

// String returns a string representation of the LiteralNode
func (l LiteralNode) String() string {
	return fmt.Sprintf("Literal(%q)", l.Value)
}

// MetaVariableNode represents a metavariable in the pattern with an optional ellipsis
type MetaVariableNode struct {
	Name     string
	Ellipsis bool
}

// String returns a string representation of the MetaVariableNode
func (m MetaVariableNode) String() string {
	if m.Ellipsis {
		return fmt.Sprintf("MetaVariable(%q, ellipsis)", m.Name)
	}
	return fmt.Sprintf("MetaVariable(%q)", m.Name)
}

// Parse converts a sequence of tokens into a slice of Nodes.
// It processes each token and creates corresponding LiteralNode or MetaVariableNode.
// Returns an error if an unexpected token type is encountered.
//
// TODO: add TokenWhiteSpace after those types are implemented
func Parse(tokens []Token) ([]Node, error) {
	var nodes []Node
	pos := 0
	for pos < len(tokens) {
		token := tokens[pos]
		if token.Type == TokenEOF {
			break
		}
		switch token.Type {
		case TokenLiteral:
			nodes = append(nodes, LiteralNode{Value: token.Value})
		case TokenMeta:
			nodes = append(nodes, MetaVariableNode{Name: token.Value, Ellipsis: token.Ellipsis})
		default:
			return nil, fmt.Errorf("unexpected token type: %v", token.Type)
		}
		pos++
	}
	return nodes, nil
}
