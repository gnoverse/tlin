package internal

import (
	"go/ast"
	"go/parser"
	"go/token"
)

func (e *Engine) detectUnnecessaryElse(filename string) ([]Issue, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	ast.Inspect(node, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		if ifStmt.Else != nil {
			blockStmt := ifStmt.Body
			if len(blockStmt.List) > 0 {
				lastStmt := blockStmt.List[len(blockStmt.List)-1]
				if _, isReturn := lastStmt.(*ast.ReturnStmt); isReturn {
					issue := Issue{
						Rule:     "unnecessary-else",
						Filename: filename,
						Start:    fset.Position(ifStmt.Else.Pos()),
						End:      fset.Position(ifStmt.Else.End()),
						Message:  "unnecessary else block",
					}
					issues = append(issues, issue)
				}
			}
		}

		return true
	})

	return issues, nil
}
