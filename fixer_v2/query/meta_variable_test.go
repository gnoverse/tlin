package query

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
)

func TestLexer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "empty input",
			input: "",
			expected: []Token{
				{Type: TokenEOF, Value: "", Position: 0},
			},
		},
		{
			name:  "only whitespace",
			input: "   \t\n  ",
			expected: []Token{
				{Type: TokenWhitespace, Value: "   \t\n  ", Position: 0},
				{Type: TokenEOF, Value: "", Position: 7},
			},
		},
		{
			name:  "adjacent holes",
			input: ":[a]:[b]:[[c]]",
			expected: []Token{
				{Type: TokenHole, Value: ":[a]", Position: 0},
				{Type: TokenHole, Value: ":[b]", Position: 4},
				{Type: TokenHole, Value: ":[[c]]", Position: 8},
				{Type: TokenEOF, Value: "", Position: 14},
			},
		},
		{
			name:  "incomplete hole pattern",
			input: ":[test",
			expected: []Token{
				{Type: TokenText, Value: ":", Position: 0},
				{Type: TokenText, Value: "[test", Position: 1},
				{Type: TokenEOF, Value: "", Position: 6},
			},
		},
		{
			name:  "simple if condition",
			input: "if :[[cond]] { return true }",
			expected: []Token{
				{Type: TokenText, Value: "if", Position: 0},
				{Type: TokenWhitespace, Value: " ", Position: 2},
				{Type: TokenHole, Value: ":[[cond]]", Position: 3},
				{Type: TokenWhitespace, Value: " ", Position: 12},
				{Type: TokenLBrace, Value: "{", Position: 13},
				{Type: TokenWhitespace, Value: " ", Position: 14},
				{Type: TokenText, Value: "return", Position: 15},
				{Type: TokenWhitespace, Value: " ", Position: 21},
				{Type: TokenText, Value: "true", Position: 22},
				{Type: TokenWhitespace, Value: " ", Position: 26},
				{Type: TokenRBrace, Value: "}", Position: 27},
				{Type: TokenEOF, Value: "", Position: 28},
			},
		},
		{
			name:  "simple hole pattern",
			input: ":[name]",
			expected: []Token{
				{Type: TokenHole, Value: ":[name]", Position: 0},
				{Type: TokenEOF, Value: "", Position: 7},
			},
		},
		{
			name:  "double bracket hole pattern",
			input: ":[[variable]]",
			expected: []Token{
				{Type: TokenHole, Value: ":[[variable]]", Position: 0},
				{Type: TokenEOF, Value: "", Position: 13},
			},
		},
		{
			name:  "multiple holes",
			input: "test :[a] :[b] :[[c]]",
			expected: []Token{
				{Type: TokenText, Value: "test", Position: 0},
				{Type: TokenWhitespace, Value: " ", Position: 4},
				{Type: TokenHole, Value: ":[a]", Position: 5},
				{Type: TokenWhitespace, Value: " ", Position: 9},
				{Type: TokenHole, Value: ":[b]", Position: 10},
				{Type: TokenWhitespace, Value: " ", Position: 14},
				{Type: TokenHole, Value: ":[[c]]", Position: 15},
				{Type: TokenEOF, Value: "", Position: 21},
			},
		},
		{
			name:  "special characters between text",
			input: "hello:{world}:test",
			expected: []Token{
				{Type: TokenText, Value: "hello:", Position: 0},
				{Type: TokenLBrace, Value: "{", Position: 6},
				{Type: TokenText, Value: "world", Position: 7},
				{Type: TokenRBrace, Value: "}", Position: 12},
				{Type: TokenText, Value: ":", Position: 13},
				{Type: TokenText, Value: "test", Position: 14},
				{Type: TokenEOF, Value: "", Position: 18},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens := lexer.Tokenize()

			if !reflect.DeepEqual(tokens, tt.expected) {
				t.Errorf("Lexer.Tokenize() got = %v, want %v", tokens, tt.expected)
			}
		})
	}
}

func TestParser(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: `PatternNode(0 children):`,
		},
		{
			name:  "multiple adjacent holes",
			input: ":[a]:[b]:[[c]]",
			expected: `PatternNode(3 children):
  0: HoleNode(a)
  1: HoleNode(b)
  2: HoleNode(c)`,
		},
		{
			name:  "if statement with condition",
			input: "if :[[cond]] { return true }",
			expected: `PatternNode(5 children):
  0: TextNode(if)
  1: TextNode( )
  2: HoleNode(cond)
  3: TextNode( )
  4: BlockNode(5 children):
    0: TextNode( )
    1: TextNode(return)
    2: TextNode( )
    3: TextNode(true)
    4: TextNode( )`,
		},
		{
			name:  "simple hole",
			input: "test :[name] here",
			expected: `PatternNode(5 children):
  0: TextNode(test)
  1: TextNode( )
  2: HoleNode(name)
  3: TextNode( )
  4: TextNode(here)`,
		},
		{
			name:  "nested blocks",
			input: "if { if { return } }",
			expected: `PatternNode(3 children):
  0: TextNode(if)
  1: TextNode( )
  2: BlockNode(5 children):
    0: TextNode( )
    1: TextNode(if)
    2: TextNode( )
    3: BlockNode(3 children):
      0: TextNode( )
      1: TextNode(return)
      2: TextNode( )
    4: TextNode( )`,
		},
		{
			name:  "complex nested blocks",
			input: "if :[[cond]] { try { :[code] } catch { :[handler] } }",
			expected: `PatternNode(5 children):
  0: TextNode(if)
  1: TextNode( )
  2: HoleNode(cond)
  3: TextNode( )
  4: BlockNode(9 children):
    0: TextNode( )
    1: TextNode(try)
    2: TextNode( )
    3: BlockNode(3 children):
      0: TextNode( )
      1: HoleNode(code)
      2: TextNode( )
    4: TextNode( )
    5: TextNode(catch)
    6: TextNode( )
    7: BlockNode(3 children):
      0: TextNode( )
      1: HoleNode(handler)
      2: TextNode( )
    8: TextNode( )`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens := lexer.Tokenize()
			parser := NewParser(tokens)
			ast := parser.Parse()

			got := removeWhitespace(t, ast.String())
			want := removeWhitespace(t, tt.expected)

			if got != want {
				t.Errorf("Parser.Parse()\ngot =\n%v\nwant =\n%v", ast.String(), tt.expected)
			}
		})
	}
}

func removeWhitespace(t *testing.T, s string) string {
	t.Helper()
	var result strings.Builder
	for _, ch := range s {
		if !unicode.IsSpace(ch) {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

func TestHoleExtraction(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty hole name",
			input:    ":[[]]",
			expected: "",
		},
		{
			name:     "simple hole",
			input:    ":[name]",
			expected: "name",
		},
		{
			name:     "double bracket hole",
			input:    ":[[variable]]",
			expected: "variable",
		},
		{
			name:     "hole with numbers",
			input:    ":[test123]",
			expected: "test123",
		},
		{
			name:     "hole with underscores",
			input:    ":[test_var_123]",
			expected: "test_var_123",
		},
		{
			name:     "hole with special characters",
			input:    ":[[test-var]]",
			expected: "test-var",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHoleName(tt.input)
			if result != tt.expected {
				t.Errorf("extractHoleName() got = %v, want %v", result, tt.expected)
			}
		})
	}
}
