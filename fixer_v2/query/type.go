package query

import (
	"fmt"
	"strconv"
	"strings"
)

// TokenType defines different types of tokens that can be produced by the lexer.
type TokenType int

const (
	TokenText       TokenType = iota // Plain text
	TokenHole                        // :[name] or :[[name]]
	TokenLBrace                      // '{'
	TokenRBrace                      // '}'
	TokenWhitespace                  // spaces, tabs, newlines, etc.
	TokenEOF                         // End of file (input)
)

// Token represents a single lexical token with type, value, and position.
type Token struct {
	Type       TokenType   // type of this token
	Value      string      // the literal string for this token
	Position   int         // the starting position in the original input
	HoleConfig *HoleConfig // configuration for hole tokens (nil for non-hole tokens)
}

// NodeType defines different node types for AST construction.
type NodeType int

const (
	NodePattern NodeType = iota
	NodeHole
	NodeText
	NodeBlock
)

// Node is an interface that any AST node must implement.
type Node interface {
	Type() NodeType // returns the node type
	String() string // debugging or printing purpose
	Position() int  // where the node starts in the input
}

var (
	_ Node = (*PatternNode)(nil)
	_ Node = (*HoleNode)(nil)
	_ Node = (*TextNode)(nil)
	_ Node = (*BlockNode)(nil)
)

// PatternNode is a top-level AST node that can contain multiple child nodes.
type PatternNode struct {
	Children []Node
	pos      int
}

func (p *PatternNode) Type() NodeType { return NodePattern }
func (p *PatternNode) String() string {
	result := fmt.Sprintf("PatternNode(%d children):\n", len(p.Children))
	for i, child := range p.Children {
		childStr := strings.ReplaceAll(child.String(), "\n", "\n  ")
		result += fmt.Sprintf("  %d: %s\n", i, childStr)
	}
	return strings.TrimRight(result, "\n")
}
func (p *PatternNode) Position() int { return p.pos }

// HoleConfig stores configuration for a hole pattern
type HoleConfig struct {
	Type       HoleType
	Quantifier Quantifier
	Name       string
}

// HoleNode represents a placeholder in the pattern like :[name] or :[[name]].
type HoleNode struct {
	Config HoleConfig
	pos    int
}

func NewHoleNode(name string, pos int) *HoleNode {
	return &HoleNode{
		Config: HoleConfig{
			Name:       name,
			Type:       HoleAny,
			Quantifier: QuantNone,
		},
		pos: pos,
	}
}

func (h *HoleNode) Type() NodeType { return NodeHole }

func (h *HoleNode) String() string {
	if h.Config.Type == HoleAny && h.Config.Quantifier == QuantNone {
		return fmt.Sprintf("HoleNode(%s)", h.Config.Name)
	}
	return fmt.Sprintf("HoleNode(%s:%s)%s", h.Config.Name, h.Config.Type, h.Config.Quantifier)
}

func (h *HoleNode) Position() int { return h.pos }
func (h *HoleNode) Name() string  { return h.Config.Name }

// TextNode represents normal text in the pattern.
type TextNode struct {
	Content string
	pos     int
}

func (t *TextNode) Type() NodeType { return NodeText }
func (t *TextNode) String() string {
	escaped := strconv.Quote(t.Content)
	return fmt.Sprintf("TextNode(%s)", escaped[1:len(escaped)-1])
}

func (t *TextNode) Position() int { return t.pos }

// BlockNode could represent a block enclosed by '{' and '}' in your syntax.
type BlockNode struct {
	Content []Node
	pos     int
}

func (b *BlockNode) Type() NodeType { return NodeBlock }
func (b *BlockNode) String() string {
	result := fmt.Sprintf("BlockNode(%d children):\n", len(b.Content))
	for i, child := range b.Content {
		// apply indentation for children node
		childStr := strings.ReplaceAll(child.String(), "\n", "\n  ")
		result += fmt.Sprintf("  %d: %s\n", i, childStr)
	}
	return strings.TrimRight(result, "\n")
}
func (b *BlockNode) Position() int { return b.pos }
