package fixerv2

import (
	"reflect"
	"testing"
)

func TestMatchHelper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		patternStr   string
		subject      string
		start        int
		wantMatch    bool
		wantEnd      int
		wantCaptures map[string]string
	}{
		{
			name:         "empty pattern, empty subject",
			patternStr:   "",
			subject:      "",
			start:        0,
			wantMatch:    true,
			wantEnd:      0,
			wantCaptures: map[string]string{},
		},
		{
			name:         "empty pattern, non-empty subject",
			patternStr:   "",
			subject:      "abc",
			start:        0,
			wantMatch:    true,
			wantEnd:      0,
			wantCaptures: map[string]string{},
		},
		{
			name:         "literal exact match",
			patternStr:   "abc",
			subject:      "abc",
			start:        0,
			wantMatch:    true,
			wantEnd:      3,
			wantCaptures: map[string]string{},
		},
		{
			name:       "literal match fail",
			patternStr: "abc",
			subject:    "ab",
			start:      0,
			wantMatch:  false,
		},
		{
			name:         "single meta variable match",
			patternStr:   ":[name]",
			subject:      "John",
			start:        0,
			wantMatch:    true,
			wantEnd:      4,
			wantCaptures: map[string]string{"name": "John"},
		},
		{
			name:       "meta with literal following match",
			patternStr: ":[name]X",
			subject:    "JohnX",
			start:      0,
			wantMatch:  true,
			// subject "JohnX": "John" (4 characters) + "X" (1 character) → total 5
			wantEnd:      5,
			wantCaptures: map[string]string{"name": "John"},
		},
		{
			name:       "meta with literal following fail",
			patternStr: ":[name]X",
			subject:    "JohnY",
			start:      0,
			wantMatch:  false,
		},
		{
			name:       "arithmetic expression match",
			patternStr: ":[lhs] + :[rhs]",
			subject:    "12 + 34",
			start:      0,
			wantMatch:  true,
			// subject "12 + 34": "12" (2 characters) + " + " (3 characters) + "34" (2 characters) → total 7
			wantEnd:      7,
			wantCaptures: map[string]string{"lhs": "12", "rhs": "34"},
		},
		{
			name:       "meta last with trailing spaces",
			patternStr: ":[expr]",
			subject:    "x + 1   ",
			start:      0,
			wantMatch:  true,
			// last meta captures the end of subject
			wantEnd:      len("x + 1   "),
			wantCaptures: map[string]string{"expr": "x + 1   "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.patternStr)
			if err != nil {
				t.Fatalf("lex error for pattern %q: %v", tt.patternStr, err)
			}
			nodes, err := Parse(tokens)
			if err != nil {
				t.Fatalf("parse error for pattern %q: %v", tt.patternStr, err)
			}
			ok, end, caps := matcher(nodes, 0, tt.subject, tt.start, map[string]string{})
			if ok != tt.wantMatch {
				t.Errorf("matchHelper() ok = %v, want %v", ok, tt.wantMatch)
			}
			if ok && end != tt.wantEnd {
				t.Errorf("matchHelper() end = %d, want %d", end, tt.wantEnd)
			}
			if ok {
				if len(caps) != len(tt.wantCaptures) {
					t.Errorf("matchHelper() captures = %v, want %v", caps, tt.wantCaptures)
				}
				for k, v := range tt.wantCaptures {
					if caps[k] != v {
						t.Errorf("matchHelper() captures[%q] = %q, want %q", k, caps[k], v)
					}
				}
			}
		})
	}
}

func TestMatcher(t *testing.T) {
	tests := []struct {
		name           string
		patternStr     string
		subjectStr     string
		expectedMatch  bool
		expectedValues map[string]string
	}{
		{
			name:           "simple literal match",
			patternStr:     "hello",
			subjectStr:     "hello",
			expectedMatch:  true,
			expectedValues: map[string]string{},
		},
		{
			name:           "plain literal match 2",
			patternStr:     ":[name] must be matched.",
			subjectStr:     "Alice must be matched.",
			expectedMatch:  true,
			expectedValues: map[string]string{"name": "Alice"},
		},
		{
			name:           "arithmetic expression",
			patternStr:     ":[lhs] + :[rhs]",
			subjectStr:     "12 + 34",
			expectedMatch:  true,
			expectedValues: map[string]string{"lhs": "12", "rhs": "34"},
		},
		{
			name:           "function pattern",
			patternStr:     "func :[name]() {\n    return :[expr]\n}",
			subjectStr:     "func test() {\n    return x + 1\n}",
			expectedMatch:  true,
			expectedValues: map[string]string{"name": "test", "expr": "x + 1"},
		},
		{
			name:           "no match",
			patternStr:     ":[lhs] + :[rhs]",
			subjectStr:     "12 - 34",
			expectedMatch:  false,
			expectedValues: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patternTokens, err := Lex(tt.patternStr)
			if err != nil {
				t.Fatalf("pattern lex error: %v", err)
			}
			patternNodes, err := Parse(patternTokens)
			if err != nil {
				t.Fatalf("pattern parse error: %v", err)
			}

			match, captures := Match(patternNodes, tt.subjectStr)

			if match != tt.expectedMatch {
				t.Errorf("matchPattern() match = %v, want %v", match, tt.expectedMatch)
			}

			if !reflect.DeepEqual(captures, tt.expectedValues) {
				t.Errorf("matchPattern() captures = %v, want %v", captures, tt.expectedValues)
			}
		})
	}
}
