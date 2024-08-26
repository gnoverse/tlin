package lints

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/gnoswap-labs/lint/internal/branch"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

// DetectEarlyReturnOpportunities detects if-else chains that can be simplified using early returns.
// This rule considers an else block unnecessary if the if block ends with a return statement.
// In such cases, the else block can be removed and the code can be flattened to improve readability.
func DetectEarlyReturnOpportunities(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	var issues []tt.Issue

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var inspectNode func(n ast.Node) bool
	inspectNode = func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		chain := analyzeIfElseChain(ifStmt)
		if canUseEarlyReturn(chain) {
			snippet := extractSnippet(ifStmt, fset, content)

			suggestion, err := generateEarlyReturnSuggestion(snippet)
			if err != nil {
				return false
			}

			issue := tt.Issue{
				Rule:       "early-return",
				Filename:   filename,
				Start:      fset.Position(ifStmt.Pos()),
				End:        fset.Position(ifStmt.End()),
				Message:    "This if-else chain can be simplified using early returns",
				Suggestion: suggestion,
			}
			issues = append(issues, issue)
		}

		// recursively check the body of the if statement
		ast.Inspect(ifStmt.Body, inspectNode)

		if ifStmt.Else != nil {
			if elseIf, ok := ifStmt.Else.(*ast.IfStmt); ok {
				inspectNode(elseIf)
			} else {
				ast.Inspect(ifStmt.Else, inspectNode)
			}
		}

		return false
	}

	ast.Inspect(node, inspectNode)

	return issues, nil
}

func analyzeIfElseChain(ifStmt *ast.IfStmt) branch.Chain {
	chain := branch.Chain{
		If:   branch.BlockBranch(ifStmt.Body),
		Else: branch.Branch{BranchKind: branch.Empty},
	}

	if ifStmt.Else != nil {
		if elseIfStmt, ok := ifStmt.Else.(*ast.IfStmt); ok {
			chain.Else = analyzeIfElseChain(elseIfStmt).If
		} else if elseBlock, ok := ifStmt.Else.(*ast.BlockStmt); ok {
			chain.Else = branch.BlockBranch(elseBlock)
		}
	}

	return chain
}

func canUseEarlyReturn(chain branch.Chain) bool {
	// If the 'if' branch deviates (returns, breaks, etc.) and there's an else branch,
	// we might be able to use an early return
	return chain.If.BranchKind.Deviates() && !chain.Else.BranchKind.IsEmpty()
}

func RemoveUnnecessaryElse(snippet string) (string, error) {
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

	removeUnnecessaryElseAndEarlyReturnRecursive(funcBody)

	var buf strings.Builder
	err = format.Node(&buf, fset, funcBody)
	if err != nil {
		return "", err
	}

	result := cleanUpResult(buf.String())

	return result, nil
}

func removeUnnecessaryElseAndEarlyReturnRecursive(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			processIfStmt(ifStmt, node)
			removeUnnecessaryElseAndEarlyReturnRecursive(ifStmt.Body)
			if ifStmt.Else != nil {
				removeUnnecessaryElseAndEarlyReturnRecursive(ifStmt.Else)
			}
			return false
		}
		return true
	})
}

func processIfStmt(ifStmt *ast.IfStmt, node ast.Node) {
	if ifStmt.Else != nil {
		ifBranch := branch.BlockBranch(ifStmt.Body)
		if ifBranch.BranchKind.Deviates() {
			parent := findParentBlockStmt(node, ifStmt)
			if parent != nil {
				switch elseBody := ifStmt.Else.(type) {
				case *ast.BlockStmt:
					insertStatementsAfter(parent, ifStmt, elseBody.List)
				case *ast.IfStmt:
					insertStatementsAfter(parent, ifStmt, []ast.Stmt{elseBody})
				}
				ifStmt.Else = nil
			}
		} else if elseIfStmt, ok := ifStmt.Else.(*ast.IfStmt); ok {
			processIfStmt(elseIfStmt, ifStmt)
		}
	}
}

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

func findParentBlockStmt(root ast.Node, child ast.Node) *ast.BlockStmt {
	var parent *ast.BlockStmt
	ast.Inspect(root, func(n ast.Node) bool {
		if n == child {
			return false
		}
		if block, ok := n.(*ast.BlockStmt); ok {
			for _, stmt := range block.List {
				if stmt == child {
					parent = block
					return false
				}
			}
		}
		return true
	})
	return parent
}

func insertStatementsAfter(block *ast.BlockStmt, target ast.Stmt, stmts []ast.Stmt) {
	for i, stmt := range block.List {
		if stmt == target {
			// insert new statements after the target statement
			block.List = append(block.List[:i+1], append(stmts, block.List[i+1:]...)...)

			for j := i + 1; j < len(block.List); j++ {
				if newIfStmt, ok := block.List[j].(*ast.IfStmt); ok {
					processIfStmt(newIfStmt, block)
				}
			}
			break
		}
	}
}

func extractSnippet(node ast.Node, fset *token.FileSet, fileContent []byte) string {
	startPos := fset.Position(node.Pos())
	endPos := fset.Position(node.End())

	// extract the relevant portion of the file content
	snippet := fileContent[startPos.Offset:endPos.Offset]
	snippet = bytes.TrimLeft(snippet, " \t\n")

	// ensure we include the entire first line
	if startPos.Column > 1 {
		lineStart := bytes.LastIndex(fileContent[:startPos.Offset], []byte{'\n'})
		if lineStart == -1 {
			lineStart = 0
		} else {
			lineStart++ // Move past the newline
		}
		prefix := fileContent[lineStart:startPos.Offset]
		snippet = append(bytes.TrimLeft(prefix, " \t"), snippet...)
	}

	// ensure we include any closing brace on its own line
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

func generateEarlyReturnSuggestion(snippet string) (string, error) {
	improved, err := RemoveUnnecessaryElse(snippet)
	if err != nil {
		return "", err
	}
	return improved, nil
}
