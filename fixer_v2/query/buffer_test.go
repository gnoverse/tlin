package query

import (
	"testing"
)

func TestNewBuffer(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantLen   int
		wantLast  States
		wantState States
	}{
		{
			name:      "empty input",
			input:     "",
			wantLen:   0,
			wantLast:  GO,
			wantState: GO,
		},
		{
			name:      "simple input",
			input:     "test",
			wantLen:   4,
			wantLast:  GO,
			wantState: GO,
		},
		{
			name:      "with metavariable",
			input:     ":[test]",
			wantLen:   7,
			wantLast:  GO,
			wantState: GO,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuffer(tt.input)
			if b.length != tt.wantLen {
				t.Errorf("newBuffer().length = %v, want %v", b.length, tt.wantLen)
			}
			if b.last != tt.wantLast {
				t.Errorf("newBuffer().last = %v, want %v", b.last, tt.wantLast)
			}
			if b.state != tt.wantState {
				t.Errorf("newBuffer().state = %v, want %v", b.state, tt.wantState)
			}
		})
	}
}

func TestBuffer_StartToken(t *testing.T) {
	b := newBuffer("test input")
	b.index = 5
	b.tokenValue.WriteString("existing")

	b.startToken()

	if b.tokenStart != 5 {
		t.Errorf("buffer.tokenStart = %v, want %v", b.tokenStart, 5)
	}
	if b.tokenValue.Len() != 0 {
		t.Errorf("buffer.tokenValue length = %v, want 0", b.tokenValue.Len())
	}
}

func TestBuffer_GetClass(t *testing.T) {
	tests := []struct {
		name  string
		input string
		index int
		want  Classes
	}{
		{
			name:  "colon",
			input: ":",
			index: 0,
			want:  C_COLON,
		},
		{
			name:  "left bracket",
			input: "[",
			index: 0,
			want:  C_LBRACK,
		},
		{
			name:  "identifier",
			input: "abc",
			index: 0,
			want:  C_IDENT,
		},
		{
			name:  "out of bounds",
			input: "",
			index: 0,
			want:  C_OTHER,
		},
		{
			name:  "whitespace",
			input: " \t\n",
			index: 0,
			want:  C_SPACE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuffer(tt.input)
			b.index = tt.index
			if got := b.getClass(); got != tt.want {
				t.Errorf("buffer.getClass() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuffer_Transition(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		initState States
		wantState States
		wantErr   bool
	}{
		{
			name:      "start of metavariable",
			input:     ":[test]",
			initState: GO,
			wantState: CL,
			wantErr:   false,
		},
		{
			name:      "invalid transition",
			input:     "]",
			initState: GO,
			wantState: ER,
			wantErr:   true,
		},
		{
			name:      "empty input",
			input:     "",
			initState: GO,
			wantState: __,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuffer(tt.input)
			b.state = tt.initState

			got, err := b.transition()
			if (err != nil) != tt.wantErr {
				t.Errorf("buffer.transition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantState {
				t.Errorf("buffer.transition() = %v, want %v", got, tt.wantState)
			}
		})
	}
}

func TestBuffer_ParseMetaVariable(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *HoleConfig
		wantErr bool
	}{
		{
			name:  "simple metavariable",
			input: ":[test]",
			want: &HoleConfig{
				Name:       "test",
				Type:       HoleAny,
				Quantifier: QuantNone,
			},
			wantErr: false,
		},
		{
			name:  "typed metavariable",
			input: ":[test:identifier]",
			want: &HoleConfig{
				Name:       "test",
				Type:       HoleIdentifier,
				Quantifier: QuantNone,
			},
			wantErr: false,
		},
		{
			name:    "incomplete metavariable",
			input:   ":[test",
			want:    nil,
			wantErr: true,
		},
		{
			name:  "simple metavariable with plus quantifier",
			input: ":[test]+",
			want: &HoleConfig{
				Name:       "test",
				Type:       HoleAny,
				Quantifier: QuantOneOrMore,
			},
			wantErr: false,
		},
		{
			name:  "simple metavariable with star quantifier",
			input: ":[test]*",
			want: &HoleConfig{
				Name:       "test",
				Type:       HoleAny,
				Quantifier: QuantZeroOrMore,
			},
			wantErr: false,
		},
		{
			name:  "simple metavariable with question mark quantifier",
			input: ":[test]?",
			want: &HoleConfig{
				Name:       "test",
				Type:       HoleAny,
				Quantifier: QuantZeroOrOne,
			},
			wantErr: false,
		},
		{
			name:  "typed metavariable with whitespace type",
			input: ":[ws:whitespace]",
			want: &HoleConfig{
				Name:       "ws",
				Type:       HoleWhitespace,
				Quantifier: QuantNone,
			},
			wantErr: false,
		},
		{
			name:    "typed metavariable but no type",
			input:   ":[test:]",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "typed metavariable but only colon",
			input:   ":[:]",
			want:    nil,
			wantErr: true,
		},
		{
			name:  "typed metavariable with block type and quantifier",
			input: ":[b:block]*",
			want: &HoleConfig{
				Name:       "b",
				Type:       HoleBlock,
				Quantifier: QuantZeroOrMore,
			},
			wantErr: false,
		},
		{
			name:    "invalid metavariable - empty name",
			input:   ":[]",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid metavariable - invalid type",
			input:   ":[test:invalid]",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid metavariable - multiple colons",
			input:   ":[test:type:extra]",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuffer(tt.input)
			got, err := b.parseMetaVariable()
			if (err != nil) != tt.wantErr {
				t.Errorf("buffer.parseMetaVariable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareHoleConfig(got, tt.want) {
				t.Errorf("buffer.parseMetaVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuffer_ParseText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "simple text",
			input:   "hello",
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "long text",
			input:   "Hello world This is a test string with some content",
			want:    "Hello world This is a test string with some content",
			wantErr: false,
		},
		{
			name:    "text until special char",
			input:   "hello:[test]",
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "only whitespace",
			input:   "  \t\n",
			want:    "  \t\n",
			wantErr: false,
		},
		{
			name:    "text until metavariable",
			input:   "hello:[var]",
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "text between metavariables",
			input:   ":[var1]middle:[var2]",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newBuffer(tt.input)
			got, err := b.parseText()
			if (err != nil) != tt.wantErr {
				t.Errorf("buffer.parseText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("buffer.parseText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func compareHoleConfig(a, b *HoleConfig) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Name == b.Name &&
		a.Type == b.Type &&
		a.Quantifier == b.Quantifier
}

func BenchmarkBuffer_ParseMetaVariable(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "simple",
			input: ":[var]",
		},
		{
			name:  "identifier_with_quantifier",
			input: ":[test:identifier]*",
		},
		{
			name:  "block_with_quantifier",
			input: ":[block:block]+",
		},
		{
			name:  "expression_optional",
			input: ":[expr:expression]?",
		},
		{
			name:  "whitespace",
			input: ":[ws:whitespace]",
		},
		{
			name:  "multiple hole expressions",
			input: ":[[var:identifier]]+ :[[expr:expression]]?",
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			buffer := newBuffer(tc.input)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buffer.index = 0
				buffer.state = GO
				_, _ = buffer.parseMetaVariable()
			}
		})
	}
}
