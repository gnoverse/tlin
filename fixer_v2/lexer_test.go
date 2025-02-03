package fixerv2

import (
	"reflect"
	"testing"
)

func TestLexer(t *testing.T) {
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
			name:  "basic meta variable",
			input: ":[name]",
			expected: []Token{
				{Type: TokenMeta, Value: "name", Ellipsis: false, Line: 1, Col: 8},
				{Type: TokenEOF, Value: "", Line: 1, Col: 8},
			},
		},
		{
			name:  "meta variable with whitespace",
			input: ":[ name ]",
			expected: []Token{
				{Type: TokenMeta, Value: "name", Ellipsis: false, Line: 1, Col: 10},
				{Type: TokenEOF, Value: "", Line: 1, Col: 10},
			},
		},
		{
			name:  "meta variable with ellipsis",
			input: ":[name...]",
			expected: []Token{
				{Type: TokenMeta, Value: "name", Ellipsis: true, Line: 1, Col: 11},
				{Type: TokenEOF, Value: "", Line: 1, Col: 11},
			},
		},
		{
			name:  "complex pattern",
			input: "func :[name]() {\n    :[body...]\n}",
			expected: []Token{
				{Type: TokenLiteral, Value: "func ", Line: 1, Col: 1},
				{Type: TokenMeta, Value: "name", Ellipsis: false, Line: 1, Col: 13},
				{Type: TokenLiteral, Value: "() {\n    ", Line: 2, Col: -4},
				{Type: TokenMeta, Value: "body", Ellipsis: true, Line: 2, Col: 15},
				{Type: TokenLiteral, Value: "\n}", Line: 3, Col: 0},
				{Type: TokenEOF, Value: "", Line: 3, Col: 2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Lex(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("lex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("lex() = %v, want %v", got, tt.expected)
			}
		})
	}
}
