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

// DetectEarlyReturnOpportunities detects if-else chains that can be simplified using early returns.
// If a parent node is already a suggestion target, nested if-else chains will not generate a separate suggestion
// and will only generate a suggestion for the top-level chain.
func DetectEarlyReturnOpportunities(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	var issues []tt.Issue

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// a helper function to traverse the function body recursively
	// inQualified is true if an early-return improvement has already been detected in a parent node
	var traverseBlock func(block *ast.BlockStmt, inQualified bool)
	traverseBlock = func(block *ast.BlockStmt, inQualified bool) {
		for _, stmt := range block.List {
			switch s := stmt.(type) {
			case *ast.IfStmt:
				qualifies := s.Else != nil && blockAlwaysTerminates(s.Body)
				currentQualified := inQualified
				if qualifies && !inQualified {
					snippet := extractSnippet(s, fset, content)
					suggestion, err := generateEarlyReturnSuggestion(snippet)
					if err == nil {
						issue := tt.Issue{
							Rule:       "early-return",
							Filename:   filename,
							Start:      fset.Position(s.Pos()),
							End:        fset.Position(s.End()),
							Message:    "this if-else chain can be simplified using early returns",
							Suggestion: suggestion,
							Confidence: 0.2,
							Severity:   severity,
						}
						issues = append(issues, issue)
					}
					// do not generate a separate suggestion for child nodes
					currentQualified = true
				}
				traverseBlock(s.Body, currentQualified)
				if s.Else != nil {
					switch elseNode := s.Else.(type) {
					case *ast.BlockStmt:
						traverseBlock(elseNode, currentQualified)
					case *ast.IfStmt:
						// even if the if-else is a single if statement, wrap it in a block
						traverseBlock(&ast.BlockStmt{List: []ast.Stmt{elseNode}}, currentQualified)
					}
				}
			case *ast.BlockStmt:
				traverseBlock(s, inQualified)
			}
		}
	}

	// traverse the body of every function declaration in the file
	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Body != nil {
			traverseBlock(funcDecl.Body, false)
		}
	}

	return issues, nil
}

// blockAlwaysTerminates returns true if the block's last statement always terminates.
func blockAlwaysTerminates(block *ast.BlockStmt) bool {
	if block == nil || len(block.List) == 0 {
		return false
	}
	return stmtAlwaysTerminates(block.List[len(block.List)-1])
}

// stmtAlwaysTerminates checks if a statement is guaranteed to terminate control flow.
func stmtAlwaysTerminates(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		// consider break or continue as terminating control flow
		if s.Tok == token.BREAK || s.Tok == token.CONTINUE {
			return true
		}
		return false
	case *ast.IfStmt:
		// if all branches terminate, consider the if statement itself as terminating
		thenTerm := blockAlwaysTerminates(s.Body)
		elseTerm := false
		if s.Else != nil {
			switch elseNode := s.Else.(type) {
			case *ast.BlockStmt:
				elseTerm = blockAlwaysTerminates(elseNode)
			case *ast.IfStmt:
				elseTerm = stmtAlwaysTerminates(elseNode)
			}
		}
		return thenTerm && elseTerm
	default:
		return false
	}
}

// RemoveUnnecessaryElse applies the AST transformation to remove unnecessary else blocks.
func RemoveUnnecessaryElse(snippet string) (string, error) {
	// wrap the snippet with a valid Go file
	wrappedSnippet := "package main\nfunc main() {\n" + snippet + "\n}"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", wrappedSnippet, parser.ParseComments)
	if err != nil {
		return "", err
	}

	var funcBody *ast.BlockStmt
	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok {
			funcBody = fd.Body
			return false
		}
		return true
	})
	if funcBody == nil {
		return "", errors.New("function body not found")
	}

	// perform the AST transformation
	transformBlock(funcBody)

	var buf bytes.Buffer
	err = format.Node(&buf, fset, funcBody)
	if err != nil {
		return "", err
	}

	result := cleanUpResult(buf.String())
	return result, nil
}

// transformBlock processes each statement in the block and applies transformation to if-statements.
func transformBlock(block *ast.BlockStmt) {
	for i := 0; i < len(block.List); i++ {
		switch stmt := block.List[i].(type) {
		case *ast.IfStmt:
			transformIfStmt(stmt)
			// inline the else block's statements if the then block terminates and there is an else block
			if stmt.Else != nil && blockAlwaysTerminates(stmt.Body) {
				var elseStmts []ast.Stmt
				switch elseNode := stmt.Else.(type) {
				case *ast.BlockStmt:
					elseStmts = elseNode.List
				case *ast.IfStmt:
					elseStmts = []ast.Stmt{elseNode}
				}
				stmt.Else = nil
				newList := make([]ast.Stmt, 0, len(block.List)+len(elseStmts))
				newList = append(newList, block.List[:i+1]...)
				newList = append(newList, elseStmts...)
				newList = append(newList, block.List[i+1:]...)
				block.List = newList
				i += len(elseStmts)
			}
		}
	}
}

// transformIfStmt applies transformation recursively to the if-statement and its children.
func transformIfStmt(ifStmt *ast.IfStmt) {
	if ifStmt.Body != nil {
		transformBlock(ifStmt.Body)
	}
	if ifStmt.Else != nil {
		switch elseNode := ifStmt.Else.(type) {
		case *ast.BlockStmt:
			transformBlock(elseNode)
		case *ast.IfStmt:
			transformIfStmt(elseNode)
		}
	}
}

// cleanUpResult trims extra braces and fixes indentation.
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

// extractSnippet extracts the code corresponding to the node from the file content.
func extractSnippet(node ast.Node, fset *token.FileSet, fileContent []byte) string {
	startPos := fset.Position(node.Pos())
	endPos := fset.Position(node.End())

	snippet := fileContent[startPos.Offset:endPos.Offset]
	snippet = bytes.TrimLeft(snippet, " \t\n")

	// include the first line
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

	// include the line with the closing brace
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

	snippet = bytes.TrimRight(snippet, " \t\n")
	return string(snippet)
}

// generateEarlyReturnSuggestion applies the transformation to produce a suggestion.
func generateEarlyReturnSuggestion(snippet string) (string, error) {
	improved, err := RemoveUnnecessaryElse(snippet)
	if err != nil {
		return "", err
	}
	return improved, nil
}
