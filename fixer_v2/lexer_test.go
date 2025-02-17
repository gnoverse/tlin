package fixerv2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected []Token
		wantErr  bool
	}{
		{
			name:  "basic literal",
			input: "hello",
			expected: []Token{
				{Type: TokenLiteral, Value: "hello", Line: 1, Col: 1},
				{Type: TokenEOF, Value: "", Line: 1, Col: 6},
			},
		},
		{
			name:  "meta variable",
			input: ":[name]",
			expected: []Token{
				{Type: TokenMeta, Value: "name", Ellipsis: false, Line: 1, Col: 8},
				{Type: TokenEOF, Value: "", Line: 1, Col: 8},
			},
		},
		{
			name:  "meta variable with ellipsis",
			input: ":[body...]",
			expected: []Token{
				{Type: TokenMeta, Value: "body", Ellipsis: true, Line: 1, Col: 11},
				{Type: TokenEOF, Value: "", Line: 1, Col: 11},
			},
		},
		{
			name: "complex function pattern",
			input: `package main

import "fmt"

func test() {
    return x + 1
}
`,
			expected: []Token{
				{Type: TokenLiteral, Value: "package main\n", Line: 1, Col: 0},
				{Type: TokenLiteral, Value: "\n", Line: 2, Col: 0},
				{Type: TokenLiteral, Value: "import \"fmt\"\n", Line: 3, Col: 0},
				{Type: TokenLiteral, Value: "\n", Line: 4, Col: 0},
				{Type: TokenLiteral, Value: "func test() {\n", Line: 5, Col: 0},
				{Type: TokenLiteral, Value: "    return x + 1\n", Line: 6, Col: 0},
				{Type: TokenLiteral, Value: "}\n", Line: 7, Col: 0},
				{Type: TokenEOF, Value: "", Line: 8, Col: 1},
			},
		},
		{
			name:  "complex pattern",
			input: "func :[name]() {\n    :[body...]\n}",
			expected: []Token{
				{Type: TokenLiteral, Value: "func ", Line: 1, Col: 1},
				{Type: TokenMeta, Value: "name", Ellipsis: false, Line: 1, Col: 13},
				{Type: TokenLiteral, Value: "() {\n", Line: 1, Col: 12},
				{Type: TokenLiteral, Value: "    ", Line: 2, Col: 1},
				{Type: TokenMeta, Value: "body", Ellipsis: true, Line: 2, Col: 15},
				{Type: TokenLiteral, Value: "\n", Line: 2, Col: 14},
				{Type: TokenLiteral, Value: "}", Line: 3, Col: 1},
				{Type: TokenEOF, Value: "", Line: 3, Col: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Lex(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Lex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
