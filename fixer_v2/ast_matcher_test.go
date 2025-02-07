package fixerv2

import (
	"go/ast"
	"go/types"
	"testing"
)

func TestParseTypeHint(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantName     string
		wantTypeHint string
	}{
		{
			name:         "expression hint",
			input:        "expr:expression",
			wantName:     "expr",
			wantTypeHint: "expression",
		},
		{
			name:         "no type hint",
			input:        "varname",
			wantName:     "varname",
			wantTypeHint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotHint := parseTypeHint(tt.input)
			if gotName != tt.wantName || gotHint != tt.wantTypeHint {
				t.Errorf("parseTypeHint(%q) = (%q, %q), want (%q, %q)",
					tt.input, gotName, gotHint, tt.wantName, tt.wantTypeHint)
			}
		})
	}
}

func TestPrepareASTMatching(t *testing.T) {
	src := `
package test

func example() int {
	x := 42
	return x
}
`
	config, err := PrepareASTMatching("test.go", src)
	if err != nil {
		t.Fatalf("PrepareASTMatching failed: %v", err)
	}

	if config.File == nil {
		t.Error("AST file is nil")
	}
	if config.TypeInfo == nil {
		t.Error("Type info is nil")
	}
}

func TestMatchAST(t *testing.T) {
	src := `
package test

func example() {
	x := 42
	y := "hello"

	println(x)
	println(y)
}
`
	config, err := PrepareASTMatching("test.go", src)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		pattern ASTMetaVariableNode
		node    ast.Node
		want    bool
	}{
		{
			name: "match expression",
			pattern: ASTMetaVariableNode{
				MetaVariableNode: MetaVariableNode{Name: "expr"},
				Kind:             MatchExact,
				ASTKind:          &ast.BasicLit{},
			},
			node: &ast.BasicLit{},
			want: true,
		},
		{
			name: "match identifier",
			pattern: ASTMetaVariableNode{
				MetaVariableNode: MetaVariableNode{Name: "id"},
				Kind:             MatchExact,
				ASTKind:          &ast.Ident{},
			},
			node: &ast.Ident{Name: "x"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchAST(tt.pattern, tt.node, *config)
			if got != tt.want {
				t.Errorf("matchAST() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtendedMatch(t *testing.T) {
	src := `
package test

func example() {
	x := 42
	print(x)
}
`
	config, err := PrepareASTMatching("test.go", src)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		pattern string
		want    bool
	}{
		{
			name:    "match simple expression",
			pattern: ":[expr:expression]",
			want:    true,
		},
		{
			name:    "match function call",
			pattern: "print(:[arg:expression])",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("pattern: %s", tt.pattern)
			tokens, err := Lex(tt.pattern)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("tokens: %v", tokens)

			nodes, err := ParseWithAST(tokens, config)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("nodes: %v", nodes)

			got, _ := ExtendedMatch(nodes, tt.pattern, config)
			if got != tt.want {
				t.Errorf("ExtendedMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchTypeInfo(t *testing.T) {
	src := `
package test

func example() {
	x := 42
	y := "hello"

	println(x)
	println(y)
}
`
	config, err := PrepareASTMatching("test.go", src)
	if err != nil {
		t.Fatal(err)
	}

	intType := types.Typ[types.Int]
	strType := types.Typ[types.String]

	tests := []struct {
		name    string
		pattern ASTMetaVariableNode
		node    ast.Node
		want    bool
	}{
		{
			name: "match int type",
			pattern: ASTMetaVariableNode{
				MetaVariableNode: MetaVariableNode{Name: "x"},
				Kind:             MatchType,
				TypeInfo:         intType,
			},
			node: &ast.Ident{Name: "x"},
			want: true,
		},
		{
			name: "match string type",
			pattern: ASTMetaVariableNode{
				MetaVariableNode: MetaVariableNode{Name: "y"},
				Kind:             MatchType,
				TypeInfo:         strType,
			},
			node: &ast.Ident{Name: "y"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchTypeInfo(tt.pattern, tt.node, *config)
			if got != tt.want {
				t.Errorf("matchTypeInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntegration(t *testing.T) {
	src := `
package test

func example() {
	x := 42
	y := "hello"
	print(x + 1)
	println(y + "world")
}
`
	testCases := []struct {
		name        string
		pattern     string
		replacement string
		want        string
	}{
		{
			name:        "replace int expression",
			pattern:     ":[expr:expression]",
			replacement: "changed",
			want:        "changed",
		},
		{
			name:        "replace function call",
			pattern:     "print(:[arg:expression])",
			replacement: "log.Print(:[arg])",
			want:        "log.Print(x + 1)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := PrepareASTMatching("test.go", src)
			if err != nil {
				t.Fatal(err)
			}

			patternTokens, err := Lex(tc.pattern)
			if err != nil {
				t.Fatal(err)
			}

			replacementTokens, err := Lex(tc.replacement)
			if err != nil {
				t.Fatal(err)
			}

			patternNodes, err := ParseWithAST(patternTokens, config)
			if err != nil {
				t.Fatal(err)
			}

			replacementNodes, err := ParseWithAST(replacementTokens, config)
			if err != nil {
				t.Fatal(err)
			}

			replacer := NewASTReplacer(patternNodes, replacementNodes, config)
			result := replacer.ReplaceAll(src)

			if result == src {
				t.Error("No replacements were made")
			}
		})
	}
}
