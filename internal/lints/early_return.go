package lints

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"

	tt "github.com/gnolang/tlin/internal/types"
)

var errNoFunctionBody = errors.New("function body not found")

// TraversalContext holds shared data for AST traversal
type traversalContext struct {
	processed map[*ast.IfStmt]bool
	fset      *token.FileSet
	filename  string
	content   []byte
	severity  tt.Severity
	issues    *[]tt.Issue
}

// IfChain represents if-else chains as a binary tree structure
type ifChain struct {
	root      *ast.IfStmt    // Current if statement
	child     *ifChain       // Child chain when else branch is an if statement
	elseBlock *ast.BlockStmt // Simple block for else branch
}

// newIfChain creates a new if chain with the given root if statement
func newIfChain(root *ast.IfStmt) *ifChain {
	return &ifChain{root: root}
}

// buildIfChain recursively constructs a chain from the given if statement
func buildIfChain(ifStmt *ast.IfStmt) *ifChain {
	chain := newIfChain(ifStmt)

	if ifStmt.Else == nil {
		return chain
	}

	switch elseNode := ifStmt.Else.(type) {
	case *ast.IfStmt:
		chain.child = buildIfChain(elseNode)
	case *ast.BlockStmt:
		chain.elseBlock = elseNode
	}

	return chain
}

// markChain registers all if statements in the chain to the processed map
func markChain(chain *ifChain, processed map[*ast.IfStmt]bool) {
	processed[chain.root] = true
	if chain.child != nil {
		markChain(chain.child, processed)
	}
}

// isQualifiedChain returns true if the top-level if statement of the chain
// always terminates and has an else branch
func isQualifiedChain(chain *ifChain) bool {
	return chain.root.Else != nil && blockAlwaysTerminates(chain.root.Body)
}

// DetectEarlyReturnOpportunities traverses the AST of functions in the file,
// constructs if-else chains as binary tree models, and generates early-return suggestions
// only for the top-level chains.
func DetectEarlyReturnOpportunities(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var issues []tt.Issue
	ctx := &traversalContext{
		processed: make(map[*ast.IfStmt]bool),
		fset:      fset,
		filename:  filename,
		content:   content,
		severity:  severity,
		issues:    &issues,
	}

	// Traverse the body of all function declarations in the file
	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Body == nil {
			continue
		}

		traverseBlock(funcDecl.Body, ctx, false)
	}

	return issues, nil
}

// traverseBlock traverses if statements in the given block.
// If inChain is true, the block is considered as part of a parent chain
func traverseBlock(block *ast.BlockStmt, ctx *traversalContext, inChain bool) {
	for _, stmt := range block.List {
		traverseStatement(stmt, ctx, inChain)
	}
}

// traverseStatement processes a statement and its children for early return opportunities
func traverseStatement(stmt ast.Stmt, ctx *traversalContext, inChain bool) {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		traverseIfStatement(s, ctx, inChain)
	case *ast.BlockStmt:
		traverseBlock(s, ctx, inChain)
	}
}

// traverseIfStatement handles the traversal of if statements specifically
func traverseIfStatement(ifStmt *ast.IfStmt, ctx *traversalContext, inChain bool) {
	// Skip already processed if statements
	if ctx.processed[ifStmt] {
		return
	}

	if !inChain {
		chain := buildIfChain(ifStmt)

		if isQualifiedChain(chain) {
			processQualifiedChain(chain, ctx)

			// Continue traversal inside the chain elements
			traverseIfChainBlocks(chain, ctx)
			return
		}
	}

	// Continue with normal traversal
	traverseBlock(ifStmt.Body, ctx, inChain)

	if ifStmt.Else != nil {
		traverseElseBranch(ifStmt.Else, ctx, inChain)
	}
}

// processQualifiedChain creates an issue for a qualified if-chain
func processQualifiedChain(chain *ifChain, ctx *traversalContext) {
	if ifChainUsesInitVars(chain.root) {
		return
	}

	snippet := extractSnippet(chain.root, ctx.fset, ctx.content)
	suggestion, err := generateEarlyReturnSuggestion(snippet)
	if err != nil {
		return
	}

	issue := tt.Issue{
		Rule:       "early-return",
		Filename:   ctx.filename,
		Start:      ctx.fset.Position(chain.root.Pos()),
		End:        ctx.fset.Position(chain.root.End()),
		Message:    "this if-else chain can be simplified using early returns",
		Suggestion: suggestion,
		Severity:   ctx.severity,
	}

	*ctx.issues = append(*ctx.issues, issue)
	markChain(chain, ctx.processed)
}

// traverseIfChainBlocks traverses all blocks within a processed chain
func traverseIfChainBlocks(chain *ifChain, ctx *traversalContext) {
	traverseBlock(chain.root.Body, ctx, true)

	if chain.root.Else != nil {
		traverseElseBranch(chain.root.Else, ctx, true)
	}
}

// traverseElseBranch handles traversal of else branches (either block or if-statement)
func traverseElseBranch(elseBranch ast.Stmt, ctx *traversalContext, inChain bool) {
	switch node := elseBranch.(type) {
	case *ast.BlockStmt:
		traverseBlock(node, ctx, inChain)
	case *ast.IfStmt:
		// Wrap the if statement in a block to fit our traversal model
		traverseBlock(&ast.BlockStmt{List: []ast.Stmt{node}}, ctx, inChain)
	}
}

// blockAlwaysTerminates determines if the last statement in the given block always terminates (return, break, etc.)
func blockAlwaysTerminates(block *ast.BlockStmt) bool {
	if block == nil || len(block.List) == 0 {
		return false
	}

	return stmtAlwaysTerminates(block.List[len(block.List)-1])
}

var alwaysTerminates = map[token.Token]bool{
	token.BREAK:    true,
	token.CONTINUE: true,
	token.RETURN:   true,
}

// stmtAlwaysTerminates determines if a statement terminates control flow
//
// TODO: Consider using branch.StmtBranch(stmt).Deviates() to determine if a statement terminates control flow
func stmtAlwaysTerminates(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		return alwaysTerminates[s.Tok]
	case *ast.IfStmt:
		if !blockAlwaysTerminates(s.Body) {
			return false
		}

		if s.Else == nil {
			return false
		}

		switch elseNode := s.Else.(type) {
		case *ast.BlockStmt:
			return blockAlwaysTerminates(elseNode)
		case *ast.IfStmt:
			return stmtAlwaysTerminates(elseNode)
		}
	}

	return false
}

// flattenIfChain recursively flattens if-else (including else-if) chains.
//
// Example:
//
//	if x > 10 { return "1" } else if x > 5 { return "2" } else { return "3" }
//
// becomes:
//
//	if x > 10 { return "1" }
//	if x > 5 { return "2" }
//	return "3"
func flattenIfChain(ifStmt *ast.IfStmt) []ast.Stmt {
	// Create copy without Else branch
	newIf := *ifStmt
	newIf.Else = nil

	result := []ast.Stmt{&newIf}

	if ifStmt.Else == nil {
		return result
	}

	switch elseNode := ifStmt.Else.(type) {
	case *ast.IfStmt:
		result = append(result, flattenIfChain(elseNode)...)
	case *ast.BlockStmt:
		result = append(result, elseNode.List...)
	}

	return result
}

// transformBlock checks if statements in the block and applies flattenIfChain if conditions are met
func transformBlock(block *ast.BlockStmt) {
	var newList []ast.Stmt

	for _, stmt := range block.List {
		switch s := stmt.(type) {
		case *ast.IfStmt:
			transformIfStmt(s)

			if s.Else != nil && blockAlwaysTerminates(s.Body) && !ifChainUsesInitVars(s) {
				flattened := flattenIfChain(s)
				newList = append(newList, flattened...)
				continue
			}

			newList = append(newList, s)
		default:
			newList = append(newList, s)
		}
	}

	block.List = newList
}

// transformIfStmt recursively transforms if statements and their branches
func transformIfStmt(ifStmt *ast.IfStmt) {
	if ifStmt.Body != nil {
		transformBlock(ifStmt.Body)
	}

	if ifStmt.Else == nil {
		return
	}

	switch elseNode := ifStmt.Else.(type) {
	case *ast.BlockStmt:
		transformBlock(elseNode)
	case *ast.IfStmt:
		transformIfStmt(elseNode)
	}
}

func ifChainUsesInitVars(ifStmt *ast.IfStmt) bool {
	if ifStmt == nil {
		return false
	}

	initNames := declaredNamesFromInit(ifStmt.Init)
	if len(initNames) > 0 && elseBranchUsesNames(ifStmt.Else, initNames) {
		return true
	}

	if elseIf, ok := ifStmt.Else.(*ast.IfStmt); ok {
		return ifChainUsesInitVars(elseIf)
	}

	return false
}

func declaredNamesFromInit(init ast.Stmt) map[string]struct{} {
	if init == nil {
		return nil
	}

	declared := map[string]struct{}{}

	switch s := init.(type) {
	case *ast.AssignStmt:
		if s.Tok != token.DEFINE {
			return nil
		}
		for _, expr := range s.Lhs {
			if ident, ok := expr.(*ast.Ident); ok {
				declared[ident.Name] = struct{}{}
			}
		}
	case *ast.DeclStmt:
		decl, ok := s.Decl.(*ast.GenDecl)
		if !ok {
			return nil
		}
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range valueSpec.Names {
				declared[name.Name] = struct{}{}
			}
		}
	}

	if len(declared) == 0 {
		return nil
	}

	return declared
}

func elseBranchUsesNames(branch ast.Stmt, names map[string]struct{}) bool {
	if branch == nil || len(names) == 0 {
		return false
	}

	return usesIdentNames(branch, names)
}

func usesIdentNames(node ast.Node, names map[string]struct{}) bool {
	if node == nil {
		return false
	}

	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil || found {
			return false
		}

		switch v := n.(type) {
		case *ast.AssignStmt:
			if v.Tok == token.DEFINE {
				for _, rhs := range v.Rhs {
					if usesIdentNames(rhs, names) {
						found = true
						return false
					}
				}
				return false
			}
		case *ast.ValueSpec:
			for _, rhs := range v.Values {
				if usesIdentNames(rhs, names) {
					found = true
					return false
				}
			}
			return false
		case *ast.Ident:
			if _, ok := names[v.Name]; ok {
				found = true
				return false
			}
		}

		return true
	})

	return found
}

// cleanUpResult cleans up unnecessary braces and indentation in the final code
func cleanUpResult(result string) string {
	result = strings.TrimSpace(result)
	result = strings.TrimPrefix(result, "{")
	result = strings.TrimSuffix(result, "}")
	result = strings.TrimSpace(result)

	lines := strings.Split(result, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, "\t")
	}

	return strings.Join(lines, "\n")
}

// extractSnippet extracts code area corresponding to the AST node from file content
func extractSnippet(node ast.Node, fset *token.FileSet, fileContent []byte) string {
	startPos := fset.Position(node.Pos())
	endPos := fset.Position(node.End())

	snippet := fileContent[startPos.Offset:endPos.Offset]
	snippet = bytes.TrimLeft(snippet, " \t\n")

	// Include leading indentation
	if startPos.Column > 1 {
		lineStart := bytes.LastIndex(fileContent[:startPos.Offset], []byte{'\n'})
		if lineStart == -1 {
			lineStart = 0
		} else {
			lineStart++
		}

		prefix := fileContent[lineStart:startPos.Offset]
		snippet = append(bytes.TrimLeft(prefix, " \t"), snippet...)
	}

	// Include trailing brace if it's on the next line
	if endPos.Column > 1 {
		nextNewline := bytes.Index(fileContent[endPos.Offset:], []byte{'\n'})
		if nextNewline != -1 {
			line := bytes.TrimSpace(fileContent[endPos.Offset : endPos.Offset+nextNewline])
			if len(line) == 1 && line[0] == '}' {
				snippet = append(snippet, line...)
				snippet = append(snippet, '\n')
			}
		}
	}

	return string(bytes.TrimRight(snippet, " \t\n"))
}

// generateEarlyReturnSuggestion generates refactored code with early returns
func generateEarlyReturnSuggestion(snippet string) (string, error) {
	return RemoveUnnecessaryElse(snippet)
}

// RemoveUnnecessaryElse removes unnecessary else blocks from the given code snippet
func RemoveUnnecessaryElse(snippet string) (string, error) {
	fset := token.NewFileSet()
	// We need to wrap the snippet in a minimal valid program structure
	// because go/parser.ParseFile requires a complete Go source file.
	// Using "package p" (any valid name) and an anonymous function "_"
	// provides the minimal context needed to parse the statement block
	// while keeping the wrapping overhead as small as possible.
	file, err := parser.ParseFile(fset, "", "package p; func _() { "+snippet+" }", parser.ParseComments)
	if err != nil {
		return "", err
	}

	var block *ast.BlockStmt
	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			block = fd.Body
			return false
		}
		return true
	})

	if block == nil {
		return "", errNoFunctionBody
	}

	transformBlock(block)

	var buf bytes.Buffer
	err = format.Node(&buf, fset, block)
	if err != nil {
		return "", err
	}

	return cleanUpResult(buf.String()), nil
}
