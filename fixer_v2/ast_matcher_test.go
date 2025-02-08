package fixerv2

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"
)

func TestParseTypeHint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantHint string
	}{
		{
			name:     "basic expression hint",
			input:    "expr:expression",
			wantName: "expr",
			wantHint: "expression",
		},
		{
			name:     "identifier hint",
			input:    "id:ident",
			wantName: "id",
			wantHint: "ident",
		},
		{
			name:     "no hint",
			input:    "name",
			wantName: "name",
			wantHint: "",
		},
		{
			name:     "empty string",
			input:    "",
			wantName: "",
			wantHint: "",
		},
		{
			name:     "multiple colons",
			input:    "a:b:c",
			wantName: "a:b:c",
			wantHint: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotHint := parseTypeHint(tt.input)
			if gotName != tt.wantName || gotHint != tt.wantHint {
				t.Errorf("parseTypeHint() = (%q, %q), want (%q, %q)",
					gotName, gotHint, tt.wantName, tt.wantHint)
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

func TestIsSameNodeType(t *testing.T) {
	ident := &ast.Ident{Name: "test"}
	basicLit := &ast.BasicLit{Kind: token.INT, Value: "1"}
	assignStmt := &ast.AssignStmt{Tok: token.ASSIGN}
	declStmt := &ast.DeclStmt{}
	genDecl := &ast.GenDecl{Tok: token.VAR}
	funcDecl := &ast.FuncDecl{}

	tests := []struct {
		name    string
		pattern ast.Node
		node    ast.Node
		want    bool
	}{
		{
			name:    "same concrete type - Ident",
			pattern: ident,
			node:    &ast.Ident{Name: "other"},
			want:    true,
		},
		{
			name:    "both implement Expr interface - Ident vs BasicLit",
			pattern: ident,
			node:    basicLit,
			want:    true,
		},
		{
			name:    "both implement Stmt interface",
			pattern: assignStmt,
			node:    declStmt,
			want:    true,
		},
		{
			name:    "both implement Decl interface",
			pattern: genDecl,
			node:    funcDecl,
			want:    true,
		},
		{
			name:    "different interfaces - Expr vs Stmt",
			pattern: ident,
			node:    assignStmt,
			want:    false,
		},
		{
			name:    "complex expressions - BinaryExpr vs UnaryExpr",
			pattern: &ast.BinaryExpr{},
			node:    &ast.UnaryExpr{},
			want:    true, // both implement ast.Expr
		},
		{
			name:    "type implementing both Expr and Stmt",
			pattern: &ast.ExprStmt{},
			node:    assignStmt,
			want:    true, // true based on Stmt interface
		},
		{
			name:    "nested node structure",
			pattern: &ast.ExprStmt{X: ident},
			node:    &ast.ExprStmt{X: basicLit},
			want:    true, // outer node types match
		},
		{
			name:    "nil pattern",
			pattern: nil,
			node:    ident,
			want:    false,
		},
		{
			name:    "nil node",
			pattern: ident,
			node:    nil,
			want:    false,
		},
		{
			name:    "both nil",
			pattern: nil,
			node:    nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSameNodeType(tt.pattern, tt.node)
			if got != tt.want {
				t.Errorf("isSameNodeType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchExactAST(t *testing.T) {
	tests := []struct {
		name    string
		pattern ASTMetaVariableNode
		node    ast.Node
		want    bool
	}{
		{
			name: "match basic literal",
			pattern: ASTMetaVariableNode{
				ASTKind: &ast.BasicLit{},
			},
			node: &ast.BasicLit{
				Kind:  token.INT,
				Value: "42",
			},
			want: true,
		},
		{
			name: "match call expression",
			pattern: ASTMetaVariableNode{
				ASTKind: &ast.CallExpr{},
			},
			node: &ast.CallExpr{
				Fun: &ast.Ident{Name: "print"},
			},
			want: true,
		},
		{
			name: "match identifier",
			pattern: ASTMetaVariableNode{
				ASTKind: &ast.Ident{},
			},
			node: &ast.Ident{Name: "x"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchExactAST(tt.pattern, tt.node)
			if got != tt.want {
				t.Errorf("matchExactAST() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindASTNodeAtPos(t *testing.T) {
	src := `package test

func example() {
    x := 42
    print(x)
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	// find and store nodes
	var nodes struct {
		assignStmt *ast.AssignStmt
		funcDecl   *ast.FuncDecl
		exprStmt   *ast.ExprStmt
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.AssignStmt:
			nodes.assignStmt = v
		case *ast.FuncDecl:
			nodes.funcDecl = v
		case *ast.ExprStmt:
			nodes.exprStmt = v
		}
		return true
	})

	tests := []struct {
		name     string
		pos      token.Pos
		wantType string
	}{
		{
			name:     "find assign statement",
			pos:      nodes.assignStmt.Lhs[0].Pos(), // position of variable x
			wantType: "*ast.AssignStmt",
		},
		{
			name:     "find function declaration",
			pos:      nodes.funcDecl.Name.Pos(), // position of function name
			wantType: "*ast.FuncDecl",
		},
		{
			name:     "find expression statement",
			pos:      nodes.exprStmt.X.Pos(), // print call pos
			wantType: "*ast.ExprStmt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findASTNodeAtPos(tt.pos, file)
			if got == nil {
				t.Fatal("findASTNodeAtPos() returned nil")
			}
			gotType := reflect.TypeOf(got).String()
			if gotType != tt.wantType {
				t.Errorf("findASTNodeAtPos() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}
