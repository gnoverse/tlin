package lints

import (
	"fmt"
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

	for _, stat := range stats {
		if stat.Complexity > threshold {
			issue := tt.Issue{
				Rule: "high-cyclomatic-complexity",
				Filename: filename,
				Start: fset.Position(token.Pos(stat.Pos.Offset)),
				End: fset.Position(token.Pos(stat.Pos.Offset)), // End position is not provided by gocyclo
				Message: fmt.Sprintf("function %s has a cyclomatic complexity of %d (threshold %d)", stat.FuncName, stat.Complexity, threshold),
			}
			issues = append(issues, issue)
		}
	}

	return issues, nil
}
