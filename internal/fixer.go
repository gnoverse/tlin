package internal

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"

	tt "github.com/gnoswap-labs/lint/internal/types"
)

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

	removeUnnecessaryElseRecursive(funcBody)

	var buf strings.Builder
	err = format.Node(&buf, fset, funcBody)
	if err != nil {
		return "", err
	}

	result := cleanUpResult(buf.String())

	return result, nil
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

func removeUnnecessaryElseRecursive(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			processIfStmt(ifStmt, node)
			removeUnnecessaryElseRecursive(ifStmt.Body)
			if ifStmt.Else != nil {
				removeUnnecessaryElseRecursive(ifStmt.Else)
			}
			return false
		}
		return true
	})
}

func processIfStmt(ifStmt *ast.IfStmt, node ast.Node) {
	if ifStmt.Else != nil && endsWithReturn(ifStmt.Body) {
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
	} else if ifStmt.Else != nil {
		if elseIfStmt, ok := ifStmt.Else.(*ast.IfStmt); ok && endsWithReturn(elseIfStmt.Body) {
			processIfStmt(elseIfStmt, ifStmt)
		}
	}
}

func endsWithReturn(block *ast.BlockStmt) bool {
	if len(block.List) == 0 {
		return false
	}
	_, isReturn := block.List[len(block.List)-1].(*ast.ReturnStmt)
	return isReturn
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
			block.List = append(block.List[:i+1], append(stmts, block.List[i+1:]...)...)
			break
		}
	}
}

func ExtractSnippet(issue tt.Issue, code string, startLine, endLine int) string {
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
