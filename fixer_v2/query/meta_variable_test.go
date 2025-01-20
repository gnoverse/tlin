package query

import (
	"testing"
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

			if len(tokens) != len(tt.expected) {
				t.Errorf("Lexer.Tokenize() got %d tokens, want %d tokens", len(tokens), len(tt.expected))
				return
			}

			for i, got := range tokens {
				want := tt.expected[i]
				if got.Type != want.Type ||
					got.Value != want.Value ||
					got.Position != want.Position {
					t.Errorf("Token[%d] = {Type: %v, Value: %q, Position: %d}, want {Type: %v, Value: %q, Position: %d}",
						i, got.Type, got.Value, got.Position,
						want.Type, want.Value, want.Position)
				}
			}
		})
	}
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
