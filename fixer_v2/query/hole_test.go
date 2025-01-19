package query

import (
	"testing"
)

func TestParseHolePattern(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		wantConfig *HoleConfig
		wantErr    bool
	}{
		{
			name:    "simple hole",
			pattern: ":[var]",
			wantConfig: &HoleConfig{
				Name:       "var",
				Type:       HoleAny,
				Quantifier: QuantNone,
			},
		},
		{
			name:    "identifier hole",
			pattern: ":[[name:identifier]]",
			wantConfig: &HoleConfig{
				Name:       "name",
				Type:       HoleIdentifier,
				Quantifier: QuantNone,
			},
		},
		{
			name:    "block hole with quantifier",
			pattern: ":[[block:block]]*",
			wantConfig: &HoleConfig{
				Name:       "block",
				Type:       HoleBlock,
				Quantifier: QuantZeroOrMore,
			},
		},
		{
			name:    "expression with plus quantifier",
			pattern: ":[[expr:expression]]+",
			wantConfig: &HoleConfig{
				Name:       "expr",
				Type:       HoleExpression,
				Quantifier: QuantOneOrMore,
			},
		},
		{
			name:    "whitespace with optional quantifier",
			pattern: ":[[ws:whitespace]]?",
			wantConfig: &HoleConfig{
				Name:       "ws",
				Type:       HoleWhitespace,
				Quantifier: QuantZeroOrOne,
			},
		},
		{
			name:    "invalid hole type",
			pattern: ":[[var:invalid]]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHolePattern(tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHolePattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if got.Name != tt.wantConfig.Name {
					t.Errorf("Name = %v, want %v", got.Name, tt.wantConfig.Name)
				}
				if got.Type != tt.wantConfig.Type {
					t.Errorf("Type = %v, want %v", got.Type, tt.wantConfig.Type)
				}
				if got.Quantifier != tt.wantConfig.Quantifier {
					t.Errorf("Quantifier = %v, want %v", got.Quantifier, tt.wantConfig.Quantifier)
				}
			}
		})
	}
}

func TestMatchHoleWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		startPos  int
		wantMatch bool
		wantToken Token
		wantPos   int
	}{
		{
			name:      "basic hole",
			input:     ":[var] rest",
			startPos:  0,
			wantMatch: true,
			wantToken: Token{
				Type:     TokenHole,
				Value:    ":[var]",
				Position: 0,
				HoleConfig: &HoleConfig{
					Name:       "var",
					Type:       HoleAny,
					Quantifier: QuantNone,
				},
			},
			wantPos: 6,
		},
		{
			name:      "typed hole",
			input:     ":[[expr:expression]] rest",
			startPos:  0,
			wantMatch: true,
			wantToken: Token{
				Type:     TokenHole,
				Value:    ":[[expr:expression]]",
				Position: 0,
				HoleConfig: &HoleConfig{
					Name:       "expr",
					Type:       HoleExpression,
					Quantifier: QuantNone,
				},
			},
			wantPos: 20,
		},
		{
			name:      "hole with quantifier",
			input:     ":[[stmts:block]]* {",
			startPos:  0,
			wantMatch: true,
			wantToken: Token{
				Type:     TokenHole,
				Value:    ":[[stmts:block]]*",
				Position: 0,
				HoleConfig: &HoleConfig{
					Name:       "stmts",
					Type:       HoleBlock,
					Quantifier: QuantZeroOrMore,
				},
			},
			wantPos: 17,
		},
		{
			name:      "whitespace hole with optional",
			input:     ":[[ws:whitespace]]? rest",
			startPos:  0,
			wantMatch: true,
			wantToken: Token{
				Type:     TokenHole,
				Value:    ":[[ws:whitespace]]?",
				Position: 0,
				HoleConfig: &HoleConfig{
					Name:       "ws",
					Type:       HoleWhitespace,
					Quantifier: QuantZeroOrOne,
				},
			},
			wantPos: 19,
		},
		{
			name:      "identifier with one or more",
			input:     ":[[ids:identifier]]+ rest",
			startPos:  0,
			wantMatch: true,
			wantToken: Token{
				Type:     TokenHole,
				Value:    ":[[ids:identifier]]+",
				Position: 0,
				HoleConfig: &HoleConfig{
					Name:       "ids",
					Type:       HoleIdentifier,
					Quantifier: QuantOneOrMore,
				},
			},
			wantPos: 20,
		},
		{
			name:      "invalid hole format",
			input:     ":[invalid rest",
			startPos:  0,
			wantMatch: false,
			wantPos:   0,
		},
		{
			name:      "invalid type",
			input:     ":[[x:invalid]] rest",
			startPos:  0,
			wantMatch: true, // should match but create basic hole
			wantToken: Token{
				Type:     TokenHole,
				Value:    ":[[x:invalid]]",
				Position: 0,
				HoleConfig: &HoleConfig{
					Name:       "x:invalid",
					Type:       HoleAny,
					Quantifier: QuantNone,
				},
			},
			wantPos: 14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input)
			l.position = tt.startPos

			got := l.matchHole()
			if got != tt.wantMatch {
				t.Errorf("matchHole() match = %v, want %v", got, tt.wantMatch)
				return
			}

			if !tt.wantMatch {
				return
			}

			if len(l.tokens) != 1 {
				t.Errorf("matchHole() produced %d tokens, want 1", len(l.tokens))
				return
			}

			gotToken := l.tokens[0]

			if gotToken.Type != tt.wantToken.Type {
				t.Errorf("Token.Type = %v, want %v", gotToken.Type, tt.wantToken.Type)
			}
			if gotToken.Value != tt.wantToken.Value {
				t.Errorf("Token.Value = %v, want %v", gotToken.Value, tt.wantToken.Value)
			}
			if gotToken.Position != tt.wantToken.Position {
				t.Errorf("Token.Position = %v, want %v", gotToken.Position, tt.wantToken.Position)
			}

			// Compare HoleConfig if present
			if tt.wantToken.HoleConfig != nil {
				if gotToken.HoleConfig == nil {
					t.Errorf("Token.HoleConfig is nil, want %+v", tt.wantToken.HoleConfig)
				} else {
					if gotToken.HoleConfig.Name != tt.wantToken.HoleConfig.Name {
						t.Errorf("HoleConfig.Name = %v, want %v",
							gotToken.HoleConfig.Name, tt.wantToken.HoleConfig.Name)
					}
					if gotToken.HoleConfig.Type != tt.wantToken.HoleConfig.Type {
						t.Errorf("HoleConfig.Type = %v, want %v",
							gotToken.HoleConfig.Type, tt.wantToken.HoleConfig.Type)
					}
					if gotToken.HoleConfig.Quantifier != tt.wantToken.HoleConfig.Quantifier {
						t.Errorf("HoleConfig.Quantifier = %v, want %v",
							gotToken.HoleConfig.Quantifier, tt.wantToken.HoleConfig.Quantifier)
					}
				}
			}

			if l.position != tt.wantPos {
				t.Errorf("Lexer position = %v, want %v", l.position, tt.wantPos)
			}
		})
	}
}

func BenchmarkParseHolePattern(b *testing.B) {
	patterns := []struct {
		name    string
		pattern string
	}{
		{
			name:    "simple",
			pattern: ":[var]",
		},
		{
			name:    "identifier_with_quantifier",
			pattern: ":[[name:identifier]]*",
		},
		{
			name:    "block_with_quantifier",
			pattern: ":[[block:block]]+",
		},
		{
			name:    "complex_expression",
			pattern: ":[[expr:expression]]?",
		},
		{
			name:    "multiple hole expressions",
			pattern: ":[[var:identifier]]+ :[[expr:expression]]?",
		},
	}

	for _, p := range patterns {
		b.Run(p.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = ParseHolePattern(p.pattern)
			}
		})
	}
}
