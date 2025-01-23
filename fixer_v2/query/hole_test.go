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
				if !got.Equal(*tt.wantConfig) {
					t.Errorf("HoleConfig = %v, want %v", got, tt.wantConfig)
				}
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
