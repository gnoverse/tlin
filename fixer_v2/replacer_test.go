package fixerv2

import (
	"testing"
)

func TestReplacer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		patternStr     string
		replacementStr string
		subjectStr     string
		expected       string
	}{
		{
			name:           "simple replacement",
			patternStr:     ":[name]",
			replacementStr: "Hello, :[name]!",
			subjectStr:     "John",
			expected:       "Hello, John!",
		},
		{
			name:           "arithmetic expression replacement",
			patternStr:     ":[lhs] + :[rhs]",
			replacementStr: ":[rhs] - :[lhs]",
			subjectStr:     "12 + 34",
			expected:       "34 - 12",
		},
		{
			name:           "multiple replacements",
			patternStr:     ":[lhs] + :[rhs]",
			replacementStr: ":[rhs] - :[lhs]",
			subjectStr:     "12 + 34",
			expected:       "34 - 12",
		},
		{
			name:           "multiple replacements 2",
			patternStr:     ":[lhs] + :[rhs]",
			replacementStr: ":[rhs] - :[lhs]",
			subjectStr:     "12 + 34, 56 + 78",
			expected:       "34 - 12, 78 - 56",
		},
		{
			name:           "function return replacement",
			patternStr:     "return :[expr]",
			replacementStr: "return (:[expr])",
			subjectStr:     "func test() {\n    return x + 1\n}",
			expected:       "func test() {\n    return (x + 1)\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			patternTokens, err := Lex(tt.patternStr)
			if err != nil {
				t.Fatalf("pattern lex error: %v", err)
			}
			patternNodes, err := Parse(patternTokens)
			if err != nil {
				t.Fatalf("pattern parse error: %v", err)
			}

			replacementTokens, err := Lex(tt.replacementStr)
			if err != nil {
				t.Fatalf("replacement lex error: %v", err)
			}
			replacementNodes, err := Parse(replacementTokens)
			if err != nil {
				t.Fatalf("replacement parse error: %v", err)
			}

			result := ReplaceAll(patternNodes, replacementNodes, tt.subjectStr)

			if result != tt.expected {
				t.Errorf("replaceAll() = %q, want %q", result, tt.expected)
			}
		})
	}
}
