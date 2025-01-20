package query

import (
	"reflect"
	"testing"
)

func TestParser_ScanMetaVariable(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Token
		wantErr bool
	}{
		{
			name:  "simple metavariable",
			input: ":[var]",
			want: Token{
				Type:     TokenHole,
				Value:    ":[var]",
				Position: 0,
				HoleConfig: &HoleConfig{
					Name:       "var",
					Type:       HoleAny,
					Quantifier: QuantNone,
				},
			},
			wantErr: false,
		},
		{
			name:  "typed metavariable",
			input: ":[x:identifier]",
			want: Token{
				Type:     TokenHole,
				Value:    ":[x:identifier]",
				Position: 0,
				HoleConfig: &HoleConfig{
					Name:       "x",
					Type:       HoleIdentifier,
					Quantifier: QuantNone,
				},
			},
			wantErr: false,
		},
		{
			name:  "metavariable with quantifier",
			input: ":[var]*",
			want: Token{
				Type:     TokenHole,
				Value:    ":[var]*",
				Position: 0,
				HoleConfig: &HoleConfig{
					Name:       "var",
					Type:       HoleAny,
					Quantifier: QuantZeroOrMore,
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid start",
			input:   "[var]",
			wantErr: true,
		},
		{
			name:    "incomplete pattern",
			input:   ":[var",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			p.buffer = newBuffer(tt.input)

			got, err := p.scanMetaVariable()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.scanMetaVariable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.scanMetaVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ScanText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Token
		wantErr bool
	}{
		{
			name:  "simple text",
			input: "hello",
			want: Token{
				Type:     TokenText,
				Value:    "hello",
				Position: 0,
			},
			wantErr: false,
		},
		{
			name:  "text until metavariable",
			input: "hello:[var]",
			want: Token{
				Type:     TokenText,
				Value:    "hello",
				Position: 0,
			},
			wantErr: false,
		},
		{
			name:  "text with special characters",
			input: "hello@world#123",
			want: Token{
				Type:     TokenText,
				Value:    "hello@world#123",
				Position: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			p.buffer = newBuffer(tt.input)

			got, err := p.scanText()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.scanText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.scanText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_ScanWhitespace(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Token
		wantErr bool
	}{
		{
			name:  "space",
			input: "   ",
			want: Token{
				Type:     TokenWhitespace,
				Value:    "   ",
				Position: 0,
			},
			wantErr: false,
		},
		{
			name:  "mixed whitespace",
			input: " \t\n\r ",
			want: Token{
				Type:     TokenWhitespace,
				Value:    " \t\n\r ",
				Position: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			p.buffer = newBuffer(tt.input)

			got, err := p.scanWhitespace()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.scanWhitespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Token의 주요 필드들만 비교
				if got.Type != tt.want.Type ||
					got.Value != tt.want.Value ||
					got.Position != tt.want.Position {
					t.Errorf("Parser.scanWhitespace() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestParser_ScanBrace(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Token
		wantErr bool
	}{
		{
			name:  "left brace",
			input: "{text}",
			want: Token{
				Type:     TokenLBrace,
				Value:    "{",
				Position: 0,
			},
			wantErr: false,
		},
		{
			name:  "right brace",
			input: "}",
			want: Token{
				Type:     TokenRBrace,
				Value:    "}",
				Position: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			p.buffer = newBuffer(tt.input)

			got, err := p.scanBrace()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.scanBrace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.scanBrace() = %v, want %v", got, tt.want)
			}
		})
	}
}

// parser_test.go
func TestParser_ParseTokenNode(t *testing.T) {
	tests := []struct {
		name    string
		tokens  []Token
		current int
		want    Node
	}{
		{
			name: "text node",
			tokens: []Token{
				{
					Type:     TokenText,
					Value:    "hello",
					Position: 0,
				},
			},
			current: 0,
			want: &TextNode{
				Content: "hello",
				pos:     0,
			},
		},
		{
			name: "whitespace node",
			tokens: []Token{
				{
					Type:     TokenWhitespace,
					Value:    "  \t",
					Position: 0,
				},
			},
			current: 0,
			want: &TextNode{
				Content: "  \t",
				pos:     0,
			},
		},
		{
			name: "hole node with config",
			tokens: []Token{
				{
					Type:     TokenHole,
					Value:    ":[var]",
					Position: 0,
					HoleConfig: &HoleConfig{
						Name:       "var",
						Type:       HoleAny,
						Quantifier: QuantNone,
					},
				},
			},
			current: 0,
			want: &HoleNode{
				Config: HoleConfig{
					Name:       "var",
					Type:       HoleAny,
					Quantifier: QuantNone,
				},
				pos: 0,
			},
		},
		{
			name: "hole node without config",
			tokens: []Token{
				{
					Type:     TokenHole,
					Value:    ":[var]",
					Position: 0,
				},
			},
			current: 0,
			want: &HoleNode{
				Config: HoleConfig{
					Name:       "var",
					Type:       HoleAny,
					Quantifier: QuantNone,
				},
				pos: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				tokens:  tt.tokens,
				holes:   make(map[string]int),
				current: tt.current,
			}

			got := p.parseTokenNode(tt.current)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTokenNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Node
		wantErr bool
	}{
		{
			name:  "empty input",
			input: "",
			want:  []Node{},
		},
		{
			name:  "simple text",
			input: "hello world",
			want: []Node{
				&TextNode{
					Content: "hello world",
					pos:     0,
				},
			},
		},
		{
			name:  "text with metavariable",
			input: "hello :[name]",
			want: []Node{
				&TextNode{
					Content: "hello ",
					pos:     0,
				},
				&HoleNode{
					Config: HoleConfig{
						Name:       "name",
						Type:       HoleAny,
						Quantifier: QuantNone,
					},
					pos: 6,
				},
			},
		},
		{
			name:  "complex pattern",
			input: "if :[cond] { :[body] }",
			want: []Node{
				&TextNode{
					Content: "if ",
					pos:     0,
				},
				&HoleNode{
					Config: HoleConfig{
						Name:       "cond",
						Type:       HoleAny,
						Quantifier: QuantNone,
					},
					pos: 3,
				},
				&TextNode{
					Content: " ",
					pos:     9,
				},
				&BlockNode{
					Content: []Node{
						&TextNode{
							Content: " ",
							pos:     11,
						},
						&HoleNode{
							Config: HoleConfig{
								Name:       "body",
								Type:       HoleAny,
								Quantifier: QuantNone,
							},
							pos: 12,
						},
						&TextNode{
							Content: " ",
							pos:     18,
						},
					},
					pos: 10,
				},
			},
		},
		{
			name:  "metavariable with type",
			input: "function :[name:identifier] (:[args])",
			want: []Node{
				&TextNode{
					Content: "function ",
					pos:     0,
				},
				&HoleNode{
					Config: HoleConfig{
						Name:       "name",
						Type:       HoleIdentifier,
						Quantifier: QuantNone,
					},
					pos: 12,
				},
				&TextNode{
					Content: " (",
					pos:     25,
				},
				&HoleNode{
					Config: HoleConfig{
						Name:       "args",
						Type:       HoleAny,
						Quantifier: QuantNone,
					},
					pos: 27,
				},
				&TextNode{
					Content: ")",
					pos:     33,
				},
			},
		},
		{
			name:    "incomplete metavariable",
			input:   "hello :[name",
			wantErr: true,
		},
		{
			name:    "unmatched block",
			input:   "if { :[body]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			got, err := p.Parse(newBuffer(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
