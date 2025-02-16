package fixerv2

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strings"
)

// ASTMatchKind defines how AST nodes should be matched
type ASTMatchKind int

const (
	_ ASTMatchKind = iota
	MatchAny
	MatchExact
	MatchType
	MatchScope
)

// ASTMetaVariableNode extends MetaVariableNode with AST-specific matching capabilities
type ASTMetaVariableNode struct {
	MetaVariableNode
	Kind     ASTMatchKind
	TypeInfo types.Type
	ASTKind  ast.Node
}

// parseTypeHint splits a metavariable name by the first colon (if any).
//
//	"expr:expression" → ("expr", "expression")
//	"call:call_expr"  → ("call", "call_expr")
//	"foo"             → ("foo", "")
func parseTypeHint(name string) (string, string) {
	parts := strings.SplitN(name, ":", 2)
	if len(parts) < 2 {
		return name, ""
	}
	return parts[0], parts[1]
}

// determineMatchKind converts type hint to ASTMatchKind
func determineMatchKind(hint string) ASTMatchKind {
	switch hint {
	case "expression", "call_expr":
		return MatchExact
	case "type":
		return MatchType
	case "scope":
		return MatchScope
	default:
		return MatchAny
	}
}

// findMatchingASTNode locates corresponding AST node for type hint
func findMatchingASTNode(hint string) ast.Node {
	switch hint {
	case "expression":
		return nil // allow all ast.Expr
	case "call_expr":
		return &ast.CallExpr{}
	case "ident":
		return &ast.Ident{}
	default:
		return nil
	}
}

// matchAST verifies if node matches pattern based on AST constraints
func matchAST(pattern ASTMetaVariableNode, node ast.Node, config Config) bool {
	switch pattern.Kind {
	case MatchExact:
		return matchExactAST(pattern, node)
	case MatchType:
		return matchTypeInfo(pattern, node, config)
	case MatchScope:
		return matchScope(pattern, node, config)
	case MatchAny:
		return true
	}
	return false
}

// matchExactAST determines if the given node matches the exact type or shape
// specified by pattern.ASTKind. When ASTKind is nil, it is interpreted as
// "any ast.Expr" (typically used when the hint is 'expression').
func matchExactAST(pattern ASTMetaVariableNode, node ast.Node) bool {
	// accept any kind of expression
	if pattern.ASTKind == nil {
		return isType[ast.Expr](node)
	}

	if !isSameNodeType(pattern.ASTKind, node) {
		return false
	}

	switch p := pattern.ASTKind.(type) {
	case *ast.BinaryExpr:
		if n, ok := node.(*ast.BinaryExpr); ok {
			return p.Op == n.Op
		}
		return false

	case *ast.CallExpr:
		// parameter count check
		if n, ok := node.(*ast.CallExpr); ok {
			if len(p.Args) > 0 {
				return len(p.Args) == len(n.Args)
			}
			return true
		}
		return false

	default:
		// For all other types, type match is sufficient
		return true
	}
}

func matchTypeInfo(pattern ASTMetaVariableNode, node ast.Node, config Config) bool {
	if config.TypeInfo == nil {
		return false
	}

	patIdent, ok := pattern.ASTKind.(*ast.Ident)
	if !ok {
		return false
	}

	ident, ok := node.(*ast.Ident)
	if !ok {
		return false
	}

	// type objects
	objPat := config.TypeInfo.ObjectOf(patIdent)
	objNode := config.TypeInfo.ObjectOf(ident)

	if objPat == nil || objNode == nil {
		return false
	}

	patUnder := objPat.Type().Underlying()
	nodeUnder := objNode.Type().Underlying()

	return types.Identical(patUnder, nodeUnder)
}

func matchScope(_ ASTMetaVariableNode, node ast.Node, config Config) bool {
	sc := config.TypeInfo.Scopes[node]
	if sc == nil {
		return false
	}

	if sc != config.Pkg.Scope() {
		return true
	}
	return false
}

// PrepareASTMatching sets up AST parsing and type checking for a Go file
func PrepareASTMatching(filename string, src string) (*Config, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	conf := types.Config{
		Importer: importer.Default(), // handle import paths
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Scopes:     make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	pkg, err := conf.Check("", fset, []*ast.File{file}, info)
	if err != nil {
		return nil, fmt.Errorf("type check error: %w", err)
	}

	return &Config{
		TypeInfo: info,
		FileSet:  fset,
		File:     file,
		Pkg:      pkg,
	}, nil
}

// ParseWithAST enhances existing parser with AST awareness
func ParseWithAST(tokens []Token, config *Config) ([]Node, error) {
	baseNodes, err := Parse(tokens)
	if err != nil {
		return nil, err
	}

	nodes := make([]Node, len(baseNodes))
	for i, node := range baseNodes {
		if meta, ok := node.(MetaVariableNode); ok {
			name, hint := parseTypeHint(meta.Name)
			if hint != "" {
				astNode := findMatchingASTNode(hint)
				nodes[i] = ASTMetaVariableNode{
					MetaVariableNode: MetaVariableNode{
						Name:     name,
						Ellipsis: meta.Ellipsis,
					},
					Kind:    determineMatchKind(hint),
					ASTKind: astNode,
				}
				continue
			}
		}
		nodes[i] = node
	}
	return nodes, nil
}

// NewASTReplacer creates a replacer with AST awareness
func NewASTReplacer(pattern, replacement []Node, config *Config) *Replacer {
	return &Replacer{
		patternNodes:     pattern,
		replacementNodes: replacement,
		config:           config,
	}
}

// findASTNodeAtPos locates AST node at given position.
// It traverses the AST and returns the most specific node that contains the position.
func findASTNodeAtPos(pos token.Pos, root *ast.File) ast.Node {
	var (
		result ast.Node
		stack  = make([]ast.Node, 0)
	)

	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1] // pop
			}
			return true
		}

		// manage stack before position check here,
		// to ensuer parent nodes are stacked properly.
		stack = append(stack, n)

		// check if current node contains the position
		if n.Pos() <= pos && pos <= n.End() {
			switch n.(type) {
			case *ast.AssignStmt, *ast.FuncDecl, *ast.ExprStmt:
				result = n
			}
			return true
		}

		stack = stack[:len(stack)-1]
		return true
	})

	// return closest ancestor from stack, if no target type was found
	if result == nil && len(stack) > 0 {
		result = stack[len(stack)-1]
	}

	return result
}

func isSameNodeType(pattern, node ast.Node) bool {
	if pattern == nil || node == nil {
		return false
	}

	patternT := reflect.TypeOf(pattern)
	nodeT := reflect.TypeOf(node)

	if isIdentType[ast.Expr](pattern, node) ||
		isIdentType[ast.Stmt](pattern, node) ||
		isIdentType[ast.Decl](pattern, node) {
		return true
	}
	return patternT == nodeT
}

func isType[T any](node ast.Node) bool {
	_, ok := node.(T)
	return ok
}

func isIdentType[T any](pattern, node ast.Node) bool {
	if isType[T](pattern) {
		return isType[T](node)
	}
	return false
}
