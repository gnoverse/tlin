package fixerv2

import (
	"go/ast"
	"go/parser"
	"go/token"
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
	
	type MyInt int
	var z MyInt = 10
	println(z)
	
	if true {
		w := 100
		println(w)
	}
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
		// TODO: fix this test
		// {
		// 	name: "match type - basic type",
		// 	pattern: ASTMetaVariableNode{
		// 		MetaVariableNode: MetaVariableNode{Name: "type"},
		// 		Kind:             MatchType,
		// 		ASTKind:          &ast.Ident{Name: "int"},
		// 	},
		// 	node: &ast.Ident{Name: "MyInt"},
		// 	want: true,
		// },
		{
			name: "match type - different types",
			pattern: ASTMetaVariableNode{
				MetaVariableNode: MetaVariableNode{Name: "type"},
				Kind:             MatchType,
				ASTKind:          &ast.Ident{Name: "string"},
			},
			node: &ast.Ident{Name: "int"},
			want: false,
		},
		{
			name: "match scope - different scope",
			pattern: ASTMetaVariableNode{
				MetaVariableNode: MetaVariableNode{Name: "scope"},
				Kind:             MatchScope,
				ASTKind:          &ast.BlockStmt{},
			},
			node: &ast.FuncDecl{},
			want: false,
		},
		{
			name: "match any - basic literal",
			pattern: ASTMetaVariableNode{
				MetaVariableNode: MetaVariableNode{Name: "any"},
				Kind:             MatchAny,
			},
			node: &ast.BasicLit{Kind: token.INT, Value: "42"},
			want: true,
		},
		{
			name: "match any - identifier",
			pattern: ASTMetaVariableNode{
				MetaVariableNode: MetaVariableNode{Name: "any"},
				Kind:             MatchAny,
			},
			node: &ast.Ident{Name: "x"},
			want: true,
		},
		{
			name: "match any - binary expression",
			pattern: ASTMetaVariableNode{
				MetaVariableNode: MetaVariableNode{Name: "any"},
				Kind:             MatchAny,
			},
			node: &ast.BinaryExpr{
				X:  &ast.Ident{Name: "x"},
				Op: token.ADD,
				Y:  &ast.Ident{Name: "y"},
			},
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
			name: "nil ASTKind should match any expression (BinaryExpr)",
			pattern: ASTMetaVariableNode{
				ASTKind: nil,
			},
			node: &ast.BinaryExpr{
				X:  &ast.Ident{Name: "x"},
				Op: token.ADD,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "20"},
			},
			want: true,
		},
		{
			name: "nil ASTKind should match any expression (CallExpr)",
			pattern: ASTMetaVariableNode{
				ASTKind: nil,
			},
			node: &ast.CallExpr{
				Fun: &ast.Ident{Name: "foo"},
				Args: []ast.Expr{
					&ast.BasicLit{Kind: token.INT, Value: "42"},
				},
			},
			want: true,
		},
		{
			name: "nil ASTKind should fail if node is not expression (e.g. File)",
			pattern: ASTMetaVariableNode{
				ASTKind: nil,
			},
			node: &ast.File{},
			want: false,
		},
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
				Args: []ast.Expr{
					&ast.BasicLit{Kind: token.STRING, Value: `"Hello"`},
				},
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
		{
			name: "binary expr - same Op token",
			pattern: ASTMetaVariableNode{
				ASTKind: &ast.BinaryExpr{
					Op: token.ADD,
				},
			},
			node: &ast.BinaryExpr{
				X:  &ast.Ident{Name: "x"},
				Op: token.ADD,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "10"},
			},
			want: true,
		},
		{
			name: "binary expr - different Op token",
			pattern: ASTMetaVariableNode{
				ASTKind: &ast.BinaryExpr{
					Op: token.ADD,
				},
			},
			node: &ast.BinaryExpr{
				X:  &ast.Ident{Name: "x"},
				Op: token.MUL,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "10"},
			},
			want: false,
		},
		{
			name: "call expr - same number of args",
			pattern: ASTMetaVariableNode{
				ASTKind: &ast.CallExpr{
					Args: []ast.Expr{
						&ast.BasicLit{},
						&ast.BasicLit{},
					},
				},
			},
			node: &ast.CallExpr{
				Fun: &ast.Ident{Name: "doSomething"},
				Args: []ast.Expr{
					&ast.BasicLit{Kind: token.STRING, Value: `"arg1"`},
					&ast.BasicLit{Kind: token.INT, Value: "123"},
				},
			},
			want: true,
		},
		{
			name: "call expr - different number of args",
			pattern: ASTMetaVariableNode{
				ASTKind: &ast.CallExpr{
					Args: []ast.Expr{
						&ast.BasicLit{},
						&ast.BasicLit{},
					},
				},
			},
			node: &ast.CallExpr{
				Fun: &ast.Ident{Name: "doSomething"},
				Args: []ast.Expr{
					&ast.BasicLit{Kind: token.STRING, Value: `"arg1"`},
				},
			},
			want: false,
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

const sampleSource = `
package main

import "fmt"

func doSomething() {
	fmt.Println("Hello")
}
`

func TestParseWithAST(t *testing.T) {
	t.Parallel()

	cfg, err := PrepareASTMatching("sample.go", sampleSource)
	if err != nil {
		t.Fatalf("PrepareASTMatching error: %v", err)
	}

	tests := []struct {
		name         string
		patternStr   string
		wantLen      int
		wantNodeType reflect.Type
		wantKind     ASTMatchKind
		wantEllipsis bool
		wantErr      bool
	}{
		{
			name:         "no hint => normal metavariable",
			patternStr:   ":[foo]",
			wantLen:      1,
			wantNodeType: reflect.TypeOf(MetaVariableNode{}),
			wantErr:      false,
		},
		{
			name:         "expression hint => ASTMetaVariableNode",
			patternStr:   ":[expr:expression]",
			wantLen:      1,
			wantNodeType: reflect.TypeOf(ASTMetaVariableNode{}),
			wantKind:     MatchExact,
			wantErr:      false,
		},
		{
			name:         "call_expr hint => ASTMetaVariableNode",
			patternStr:   ":[call:call_expr]",
			wantLen:      1,
			wantNodeType: reflect.TypeOf(ASTMetaVariableNode{}),
			wantKind:     MatchExact,
			wantErr:      false,
		},
		{
			name:         "no AST hint but ellipsis",
			patternStr:   ":[body...]",
			wantLen:      1,
			wantNodeType: reflect.TypeOf(MetaVariableNode{}),
			wantEllipsis: true,
			wantErr:      false,
		},
		{
			name:         "AST hint + ellipsis",
			patternStr:   ":[any:expression...]",
			wantLen:      1,
			wantNodeType: reflect.TypeOf(ASTMetaVariableNode{}),
			wantEllipsis: true,
			wantKind:     MatchExact,
			wantErr:      false,
		},
		{
			name:       "invalid pattern => missing bracket",
			patternStr: ":[broken",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokens, lexErr := Lex(tt.patternStr)
			if lexErr != nil {
				if !tt.wantErr {
					t.Errorf("Lex error = %v, wantErr = %v", lexErr, tt.wantErr)
				}
				return
			}
			if tt.wantErr && lexErr == nil {
				t.Errorf("expected error but got none")
				return
			}

			if tt.wantErr {
				return
			}

			nodes, parseErr := ParseWithAST(tokens, cfg)
			if parseErr != nil {
				if !tt.wantErr {
					t.Errorf("ParseWithAST error = %v, wantErr = %v", parseErr, tt.wantErr)
				}
				return
			}
			if tt.wantErr && parseErr == nil {
				t.Errorf("expected error but got none")
				return
			}

			if len(nodes) != tt.wantLen {
				t.Fatalf("len(nodes) = %d, want %d", len(nodes), tt.wantLen)
			}

			// let assume pattern has only one node
			node := nodes[0]
			nodeType := reflect.TypeOf(node)
			if nodeType != tt.wantNodeType {
				t.Errorf("node type = %v, want %v", nodeType, tt.wantNodeType)
			}

			// check ellipsis and AST hint
			switch n := node.(type) {
			case MetaVariableNode:
				if n.Ellipsis != tt.wantEllipsis {
					t.Errorf("node.Ellipsis = %v, want %v", n.Ellipsis, tt.wantEllipsis)
				}
			case ASTMetaVariableNode:
				if n.Ellipsis != tt.wantEllipsis {
					t.Errorf("node.Ellipsis = %v, want %v", n.Ellipsis, tt.wantEllipsis)
				}
				if tt.wantKind != 0 && n.Kind != tt.wantKind {
					t.Errorf("node.Kind = %v, want %v", n.Kind, tt.wantKind)
				}
			}
		})
	}
}
