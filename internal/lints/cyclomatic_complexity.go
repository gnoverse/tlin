package lints

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/fzipp/gocyclo"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

func DetectHighCyclomaticComplexity(filename string, threshold int) ([]tt.Issue, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	stats := gocyclo.AnalyzeASTFile(f, fset, nil)
	var issues []tt.Issue

	funcNodes := make(map[string]*ast.FuncDecl)
	ast.Inspect(f, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			funcNodes[fn.Name.Name] = fn
		}
		return true
	})

	for _, stat := range stats {
		if stat.Complexity > threshold {

			funcNode, ok := funcNodes[stat.FuncName]
			if !ok {
				continue
			}

			issue := tt.Issue{
				Rule:       "high-cyclomatic-complexity",
				Filename:   filename,
				Start:      fset.Position(funcNode.Pos()),
				End:        fset.Position(funcNode.End()),
				Message:    fmt.Sprintf("function %s has a cyclomatic complexity of %d (threshold %d)", stat.FuncName, stat.Complexity, threshold),
				Suggestion: "Consider refactoring this function to reduce its complexity. You can split it into smaller functions or simplify the logic.\n",
				Note:       "High cyclomatic complexity can make the code harder to understand, test, and maintain. Aim for a complexity score of 10 or less for most functions.\n",
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}
