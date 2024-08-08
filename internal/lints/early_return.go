package lints

import (
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
	src := string(content)

	var inspectNode func(n ast.Node) bool
	inspectNode = func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		chain := analyzeIfElseChain(ifStmt)
		if canUseEarlyReturn(chain) {
			startLine := fset.Position(ifStmt.Pos()).Line - 1
			endLine := fset.Position(ifStmt.End()).Line
			snippet := ExtractSnippet(src, startLine, endLine)

			suggestion, err := generateEarlyReturnSuggestion(snippet)
			if err != nil {
				return false
			}

			issue := tt.Issue{
				Rule:     "early-return",
				Filename: filename,
				Start:    fset.Position(ifStmt.Pos()),
				End:      fset.Position(ifStmt.End()),
				Message:  "This if-else chain can be simplified using early returns",
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

func ExtractSnippet(code string, startLine, endLine int) string {
	lines := strings.Split(code, "\n")

	// ensure we don't go out of bounds
	if startLine < 0 {
		startLine = 0
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	// extract the relevant lines
	snippet := lines[startLine:endLine]

	// trim any leading empty lines
	for len(snippet) > 0 && strings.TrimSpace(snippet[0]) == "" {
		snippet = snippet[1:]
	}

	// ensure the last line is included if it's a closing brace
	if endLine < len(lines) && strings.TrimSpace(lines[endLine]) == "}" {
		snippet = append(snippet, lines[endLine])
	}

	// trim any trailing empty lines
	for len(snippet) > 0 && strings.TrimSpace(snippet[len(snippet)-1]) == "" {
		snippet = snippet[:len(snippet)-1]
	}

	return strings.Join(snippet, "\n")
}

func generateEarlyReturnSuggestion(snippet string) (string, error) {
	improved, err := RemoveUnnecessaryElse(snippet)
	if err != nil {
		return "", err
	}
	return improved, nil
}
