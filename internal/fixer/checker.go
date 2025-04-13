package fixer

// TODO:
// Currently, all comparisons are based on "exactly identical node content,"
// so even some safe refactoring may be detected as false.
// (e.g., variable name changes, removal of unnecessary parentheses, etc.)
//
// Need to consider tolerance or matching algorithms (e.g., AST similarity measurement) for "equivalence comparison."

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"maps"
	"reflect"
	"strings"

	"github.com/gnolang/tlin/internal/analysis/cfg"
	"github.com/gnolang/tlin/internal/trie"
)

// ContentBasedCFGChecker enhances the CFG equivalence checker to analyze node content.
type ContentBasedCFGChecker struct {
	MinConfidence  float64
	DetailedReport bool
	reportBuffer   strings.Builder
	fset           *token.FileSet // debugging
}

// NewContentBasedCFGChecker creates a new content-based CFG equivalence checker.
func NewContentBasedCFGChecker(threshold float64, detailed bool) *ContentBasedCFGChecker {
	return &ContentBasedCFGChecker{
		MinConfidence:  threshold,
		DetailedReport: detailed,
		fset:           token.NewFileSet(),
	}
}

// CheckEquivalence checks if two code snippets have equivalent control flow.
func (c *ContentBasedCFGChecker) CheckEquivalence(originalCode, modifiedCode string) (bool, string, error) {
	c.reportBuffer.Reset()

	// Parse both code snippets.
	astFileOrig, err := parser.ParseFile(c.fset, "original.go", originalCode, parser.ParseComments)
	if err != nil {
		return false, "", fmt.Errorf("failed to parse original file: %w", err)
	}
	astFileModified, err := parser.ParseFile(c.fset, "modified.go", modifiedCode, parser.ParseComments)
	if err != nil {
		return false, "", fmt.Errorf("failed to parse modified file: %w", err)
	}

	// extract and compare functions
	origFuncs := extractFuncs(astFileOrig)
	modifiedFuncs := extractFuncs(astFileModified)
	if len(origFuncs) != len(modifiedFuncs) {
		c.logf("Function count mismatch: original=%d, modified=%d", len(origFuncs), len(modifiedFuncs))
		return false, c.reportBuffer.String(), nil
	}

	// compare each function
	isEquivalent := true
	for funcName, origFunc := range origFuncs {
		modifiedFunc, exists := modifiedFuncs[funcName]
		if !exists {
			c.logf("Function '%s' exists in original but not in modified", funcName)
			isEquivalent = false
			continue
		}

		origCFG := cfg.FromFunc(origFunc)
		modifiedCFG := cfg.FromFunc(modifiedFunc)

		// CFG equivalence check
		funcEquiv, reason := c.areCFGsEquivalent(origCFG, modifiedCFG, origFunc, modifiedFunc)
		if !funcEquiv {
			c.logf("Function '%s' CFGs are not equivalent: %s", funcName, reason)
			isEquivalent = false
		} else if c.DetailedReport {
			c.logf("Function '%s' CFGs are equivalent", funcName)
		}
	}

	return isEquivalent, c.reportBuffer.String(), nil
}

// areCFGsEquivalent compares two CFGs including node content.
func (c *ContentBasedCFGChecker) areCFGsEquivalent(
	origCFG, modifiedCFG *cfg.CFG,
	origFunc, modifiedFunc *ast.FuncDecl,
) (bool, string) {
	// 1. Basic structure check
	origBlocks := origCFG.Blocks()
	modifiedBlocks := modifiedCFG.Blocks()
	if len(origBlocks) != len(modifiedBlocks) {
		return false, fmt.Sprintf("Block count mismatch: original=%d, modified=%d",
			len(origBlocks), len(modifiedBlocks))
	}

	// 2. Node type distribution check
	origNodeTypes := getNodeTypeDistribution(origBlocks)
	modifiedNodeTypes := getNodeTypeDistribution(modifiedBlocks)
	if !areNodeTypesEqual(origNodeTypes, modifiedNodeTypes) {
		return false, "Node type distribution mismatch"
	}

	// 3. Path structure check
	pathsEqual, reason := c.comparePathStructures(origCFG, modifiedCFG)
	if !pathsEqual {
		return false, fmt.Sprintf("Path structure mismatch: %s", reason)
	}

	// 4. Content-based checks
	contentEqual, reason := c.compareNodeContents(origFunc, modifiedFunc)
	if !contentEqual {
		return false, fmt.Sprintf("Node content mismatch: %s", reason)
	}

	return true, ""
}

// compareNodeContents analyzes the actual content of corresponding nodes.
func (c *ContentBasedCFGChecker) compareNodeContents(
	origFunc, modifiedFunc *ast.FuncDecl,
) (bool, string) {
	// Create maps to track conditions and expressions.
	origConditions := extractConditions(origFunc)
	modifiedConditions := extractConditions(modifiedFunc)

	// Compare the number of conditions.
	if len(origConditions) != len(modifiedConditions) {
		return false, fmt.Sprintf("Condition count mismatch: original=%d, modified=%d",
			len(origConditions), len(modifiedConditions))
	}

	// Compare each condition's string representation using normalized printing.
	for i, origCond := range origConditions {
		if i < len(modifiedConditions) {
			origStr := normalizedNodeToString(c.fset, origCond)
			modifiedStr := normalizedNodeToString(c.fset, modifiedConditions[i])
			if origStr != modifiedStr {
				return false, fmt.Sprintf("Condition mismatch: original='%s', modified='%s'",
					origStr, modifiedStr)
			}
		}
	}

	// Check loop iteration variables and conditions.
	origLoops := extractLoops(origFunc)
	modifiedLoops := extractLoops(modifiedFunc)
	if len(origLoops) != len(modifiedLoops) {
		return false, fmt.Sprintf("Loop count mismatch: original=%d, modified=%d",
			len(origLoops), len(modifiedLoops))
	}
	for i, origLoop := range origLoops {
		if i < len(modifiedLoops) {
			equal, reason := compareLoops(c.fset, origLoop, modifiedLoops[i])
			if !equal {
				return false, reason
			}
		}
	}

	// Check branch statements (break, continue, return).
	origBranches := extractBranchStmts(origFunc)
	modifiedBranches := extractBranchStmts(modifiedFunc)
	if len(origBranches) != len(modifiedBranches) {
		return false, fmt.Sprintf("Branch statement count mismatch: original=%d, modified=%d",
			len(origBranches), len(modifiedBranches))
	}
	for i, origBranch := range origBranches {
		if i < len(modifiedBranches) {
			if origBranch.Tok != modifiedBranches[i].Tok {
				return false, fmt.Sprintf("Branch token mismatch: original=%v, modified=%v",
					origBranch.Tok, modifiedBranches[i].Tok)
			}
			// Compare labels if they exist.
			if origBranch.Label != nil && modifiedBranches[i].Label != nil {
				if origBranch.Label.Name != modifiedBranches[i].Label.Name {
					return false, fmt.Sprintf("Branch label mismatch: original=%s, modified=%s",
						origBranch.Label.Name, modifiedBranches[i].Label.Name)
				}
			} else if (origBranch.Label == nil) != (modifiedBranches[i].Label == nil) {
				return false, "Branch label presence mismatch"
			}
		}
	}

	return true, ""
}

// logf writes formatted logs to the report buffer.
func (c *ContentBasedCFGChecker) logf(format string, args ...interface{}) {
	c.reportBuffer.WriteString(fmt.Sprintf(format+"\n", args...))
}

/*
	The following functions utilize a trie data structure to analyze and compare the control flow paths
	extracted from a CFG. We opted for a trie-based approach because it enables efficient compression
	of common prefixes among many paths.

	By converting each path to a sequence of node types and inserting
	these sequences into a trie, we can succinctly capture the structure of the code's execution paths.
	This method not only reduces memory usage when handling numerous or similar paths but also allows
	a holistic comparison of the overall path structure between the original and modified code.

	If the trie structures are identical, we can infer that the control flow remains unchanged despite potential
	modifications in the code.
*/

func (c *ContentBasedCFGChecker) comparePathStructures(origCFG, fixedCFG *cfg.CFG) (bool, string) {
	return c.comparePathStructuresTrie(origCFG, fixedCFG)
}

// comparePathStructuresTrie uses a trie to compare the path structures of two CFGs.
func (c *ContentBasedCFGChecker) comparePathStructuresTrie(origCFG, fixedCFG *cfg.CFG) (bool, string) {
	// find all paths from entry to exit.
	origPaths := findAllPaths(origCFG, origCFG.Entry, origCFG.Exit)
	fixedPaths := findAllPaths(fixedCFG, fixedCFG.Entry, fixedCFG.Exit)

	// extract sequences (node type sequences) from paths.
	origSequences := convertPathsToSequences(origPaths)
	fixedSequences := convertPathsToSequences(fixedPaths)

	// build Trie structures from sequences.
	origTrie := buildTrieFromSequences(origSequences)
	fixedTrie := buildTrieFromSequences(fixedSequences)

	if c.DetailedReport {
		c.logf("Original Trie: %s", origTrie.String())
		c.logf("Modified Trie: %s", fixedTrie.String())
	}

	// once the trie structure is equal, the path structure is also equal.
	if !origTrie.Eq(fixedTrie) {
		return false, "Trie-based path structure mismatch"
	}
	return true, ""
}

// normalizeAST traverses the given AST node and resets all token.Pos fields to token.NoPos.
// This removes unnecessary differences due to position information.
func normalizeAST(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		// n is mostly a pointer, so access the actual value.
		v := reflect.ValueOf(n)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		// traverse fields of struct.
		if v.Kind() != reflect.Struct {
			return true
		}
		for i := range v.NumField() {
			field := v.Field(i)
			if !field.CanSet() {
				continue
			}
			// initialize token.Pos to token.NoPos.
			if field.Type() == reflect.TypeOf(token.NoPos) {
				field.Set(reflect.ValueOf(token.NoPos))
			}
			// TODO: consider additional processing for slices or nested structures.
		}
		return true
	})
}

// normalizedNodeToString converts an AST node to its normalized string representation.
// It applies normalizeAST internally to ensure that formatting and position information differences do not affect the comparison.
//
// NOTE: This function normalizes the AST in-place.
func normalizedNodeToString(fset *token.FileSet, node ast.Node) string {
	normalizeAST(node)
	var buf bytes.Buffer
	err := printer.Fprint(&buf, fset, node)
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return buf.String()
}

// extractConditions collects all condition expressions from a function.
func extractConditions(funcDecl *ast.FuncDecl) []ast.Expr {
	var conditions []ast.Expr
	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			conditions = appendConds(conditions, node.Cond)
		case *ast.ForStmt:
			conditions = appendConds(conditions, node.Cond)
		case *ast.SwitchStmt:
			conditions = appendConds(conditions, node.Tag)
		case *ast.TypeSwitchStmt:
			// do nothing
		}
		return true
	})
	return conditions
}

func appendConds(conds []ast.Expr, expr ast.Expr) []ast.Expr {
	if expr != nil {
		conds = append(conds, expr)
	}
	return conds
}

// extractLoops collects all loop statements from a function.
func extractLoops(funcDecl *ast.FuncDecl) []ast.Stmt {
	var loops []ast.Stmt
	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			loops = append(loops, n.(ast.Stmt))
		}
		return true
	})
	return loops
}

// compareNodePair compares two AST nodes and returns a detailed error message if they differ
func compareNodePair(fset *token.FileSet, name string, orig, modified ast.Node) (bool, string) {
	if (orig == nil) != (modified == nil) {
		return false, fmt.Sprintf("%s presence mismatch", name)
	}
	if orig == nil && modified == nil {
		return true, ""
	}

	origStr := normalizedNodeToString(fset, orig)
	modifiedStr := normalizedNodeToString(fset, modified)
	if origStr != modifiedStr {
		return false, fmt.Sprintf("%s mismatch: original='%s', modified='%s'",
			name, origStr, modifiedStr)
	}
	return true, ""
}

// compareLoopBody compares the bodies of two loop statements
func compareLoopBody(fset *token.FileSet, orig, modified ast.Stmt) (bool, string) {
	return compareNodePair(fset, "Loop body", loopBody(orig), loopBody(modified))
}

func loopBody(stmt ast.Stmt) *ast.BlockStmt {
	switch loop := stmt.(type) {
	case *ast.ForStmt:
		return loop.Body
	case *ast.RangeStmt:
		return loop.Body
	default:
		return nil
	}
}

// compareLoops checks if two loop statements are equivalent
func compareLoops(fset *token.FileSet, orig, modified ast.Stmt) (bool, string) {
	switch origLoop := orig.(type) {
	case *ast.ForStmt:
		modifiedLoop, ok := modified.(*ast.ForStmt)
		if !ok {
			return false, "Loop type mismatch"
		}

		// Compare init, condition, and post statements
		if ok, msg := compareNodePair(fset, "Loop init", origLoop.Init, modifiedLoop.Init); !ok {
			return false, msg
		}
		if ok, msg := compareNodePair(fset, "Loop condition", origLoop.Cond, modifiedLoop.Cond); !ok {
			return false, msg
		}
		if ok, msg := compareNodePair(fset, "Loop post", origLoop.Post, modifiedLoop.Post); !ok {
			return false, msg
		}

		// Compare loop body
		if ok, msg := compareLoopBody(fset, orig, modified); !ok {
			return false, msg
		}

	case *ast.RangeStmt:
		modifiedLoop, ok := modified.(*ast.RangeStmt)
		if !ok {
			return false, "Loop type mismatch"
		}

		// compare key, value, and expressions
		if ok, msg := compareNodePair(fset, "Range key", origLoop.Key, modifiedLoop.Key); !ok {
			return false, msg
		}
		if ok, msg := compareNodePair(fset, "Range value", origLoop.Value, modifiedLoop.Value); !ok {
			return false, msg
		}
		if ok, msg := compareNodePair(fset, "Range expression", origLoop.X, modifiedLoop.X); !ok {
			return false, msg
		}

		// Compare loop body
		if ok, msg := compareLoopBody(fset, orig, modified); !ok {
			return false, msg
		}
	}

	return true, ""
}

// findAllPaths finds all paths from start node to end node in CFG.
func findAllPaths(graph *cfg.CFG, start, end ast.Stmt) [][]ast.Stmt {
	var paths [][]ast.Stmt
	visited := make(map[ast.Stmt]bool)
	currentPath := make([]ast.Stmt, 0)

	var dfs func(current ast.Stmt)
	dfs = func(current ast.Stmt) {
		if visited[current] {
			return
		}
		visited[current] = true
		currentPath = append(currentPath, current)
		if current == end {
			pathCopy := make([]ast.Stmt, len(currentPath))
			copy(pathCopy, currentPath)
			paths = append(paths, pathCopy)
		} else {
			for _, succ := range graph.Succs(current) {
				dfs(succ)
			}
		}
		currentPath = currentPath[:len(currentPath)-1]
		visited[current] = false
	}
	dfs(start)
	return paths
}

// extractBranchStmts collects all branch statements (break, continue, return) from a function.
func extractBranchStmts(funcDecl *ast.FuncDecl) []*ast.BranchStmt {
	var branches []*ast.BranchStmt
	ast.Inspect(funcDecl, func(n ast.Node) bool {
		if branch, ok := n.(*ast.BranchStmt); ok {
			branches = append(branches, branch)
		}
		return true
	})
	return branches
}

// extractFuncs extracts function declarations from the AST.
func extractFuncs(file *ast.File) map[string]*ast.FuncDecl {
	funcs := make(map[string]*ast.FuncDecl)
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			funcs[funcDecl.Name.Name] = funcDecl
		}
	}
	return funcs
}

// getNodeTypeDistribution returns the node type distribution of CFG.
func getNodeTypeDistribution(blocks []ast.Stmt) map[string]int {
	distribution := make(map[string]int)
	for _, block := range blocks {
		typeStr := fmt.Sprintf("%T", block)
		distribution[typeStr]++
	}
	return distribution
}

// areNodeTypesEqual checks if two node type distributions are identical.
func areNodeTypesEqual(orig, fixed map[string]int) bool {
	return maps.Equal(orig, fixed)
}

// convertPathsToSequences converts a list of paths (slice of ast.Stmt slices)
// into a slice of string sequences where each string represents the node type.
// e.g., "*ast.IfStmt" -> "IfStmt"
func convertPathsToSequences(paths [][]ast.Stmt) [][]string {
	sequences := make([][]string, len(paths))
	for i, path := range paths {
		var seq []string
		for _, node := range path {
			nodeType := fmt.Sprintf("%T", node)
			nodeType = strings.TrimPrefix(nodeType, "*ast.")
			seq = append(seq, nodeType)
		}
		sequences[i] = seq
	}
	return sequences
}

// buildTrieFromSequences builds a trie from a list of string sequences.
func buildTrieFromSequences(sequences [][]string) *trie.Trie {
	root := trie.New()
	for _, seq := range sequences {
		root.Insert(seq)
	}
	return root
}
