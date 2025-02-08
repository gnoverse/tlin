package fixerv2

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strings"
)

// ASTMatchKind defines how AST nodes should be matched
type ASTMatchKind int

const (
	MatchAny ASTMatchKind = iota + 1
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

// ParseTypeHint parses type hints from metavariable names
// e.g., "expr:expression" -> (expr, expression)
func parseTypeHint(name string) (string, string) {
	parts := strings.Split(name, ":")
	if len(parts) != 2 {
		return name, ""
	}
	return parts[0], parts[1]
}

// determineMatchKind converts type hint to ASTMatchKind
func determineMatchKind(hint string) ASTMatchKind {
	switch hint {
	case "expression":
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
func findMatchingASTNode(hint string, file *ast.File) ast.Node {
	switch hint {
	case "expression":
		return &ast.BasicLit{}
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

func matchExactAST(pattern ASTMetaVariableNode, node ast.Node) bool {
	if !isSameNodeType(pattern.ASTKind, node) {
		return false
	}

	switch p := pattern.ASTKind.(type) {
	case *ast.BinaryExpr:
		if n, ok := node.(*ast.BinaryExpr); ok {
			return p.Op == n.Op
		}
	case *ast.CallExpr:
		if n, ok := node.(*ast.CallExpr); ok {
			return len(p.Args) == len(n.Args)
		}
	}
	return true
}

func matchTypeInfo(pattern ASTMetaVariableNode, node ast.Node, config Config) bool {
	if config.TypeInfo == nil {
		return false
	}

	ident, ok := node.(*ast.Ident)
	if !ok {
		return false
	}

	if obj := config.TypeInfo.ObjectOf(ident); obj != nil {
		if pattern.TypeInfo != nil {
			return types.Identical(pattern.TypeInfo, obj.Type())
		}
	}
	return false
}

func matchScope(pattern ASTMetaVariableNode, node ast.Node, config Config) bool {
	ident, ok := node.(*ast.Ident)
	if !ok {
		return false
	}

	if obj := config.TypeInfo.ObjectOf(ident); obj != nil {
		switch obj.(type) {
		case *types.Var:
			// Check if variable is local to function
			return obj.Parent() != config.Pkg.Scope()
		}
	}
	return false
}

// PrepareASTMatching sets up AST parsing and type checking for a Go file
func PrepareASTMatching(filename string, src string) (*Config, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}

	conf := types.Config{Importer: nil}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Scopes:     make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}

	pkg, err := conf.Check("", fset, []*ast.File{file}, info)
	if err != nil {
		return nil, fmt.Errorf("type check error: %v", err)
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
				astNode := findMatchingASTNode(hint, config.File)
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

// matcherWithAST combines text-based and AST-based matching
func matcherWithAST(nodes []Node, pIdx int, subject string, sIdx int, captures map[string]string, config *Config, astNode ast.Node) (bool, int, map[string]string) {
	if pIdx == len(nodes) {
		return true, sIdx, captures
	}

	switch node := nodes[pIdx].(type) {
	case ASTMetaVariableNode:
		if !matchAST(node, astNode, *config) {
			return false, 0, nil
		}
		// Continue with text-based matching
		return matcher(nodes, pIdx+1, subject, sIdx, captures)

	default:
		return matcher(nodes, pIdx+1, subject, sIdx, captures)
	}
}

// findASTNodeAtPos locates AST node at given position.
// It traverses the AST and returns the most specific node that contains the position.
// If no specific node type (AssignStmt, FuncDecl, ExprStmt) is found,
// it returns the closest ancestor node from the traversal stack.
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
			// continue traversal to find most specific node
			return true
		}

		// remove node from stack if it doesn't contain the position
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

// ExtendedMatch performs both text and AST-based matching
func ExtendedMatch(nodes []Node, subject string, config *Config) (bool, map[string]string) {
	// First, perform text-based matching
	ok, end, captures := matcher(nodes, 0, subject, 0, map[string]string{})
	if !ok || end != len(subject) {
		return false, nil
	}

	// Then verify AST constraints
	for _, node := range nodes {
		if astNode, ok := node.(ASTMetaVariableNode); ok {
			pos := config.FileSet.File(config.File.Pos()).Pos(0) // Convert to token.Pos
			matchingNode := findASTNodeAtPos(pos, config.File)
			if matchingNode == nil || !matchAST(astNode, matchingNode, *config) {
				return false, nil
			}
		}
	}

	return true, captures
}
