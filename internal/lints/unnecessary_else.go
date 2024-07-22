package lints

import (
	"go/ast"
	"go/parser"
	"go/token"

	tt "github.com/gnoswap-labs/lint/internal/types"
)

// DetectUnnecessaryElse detects unnecessary else blocks.
// This rule considers an else block unnecessary if the if block ends with a return statement.
// In such cases, the else block can be removed and the code can be flattened to improve readability.
func DetectUnnecessaryElse(f string) ([]tt.Issue, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var issues []tt.Issue
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
					issue := tt.Issue{
						Rule:     "unnecessary-else",
						Filename: f,
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
