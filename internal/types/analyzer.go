package types

import (
	"go/ast"
	"go/parser"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// TODO: keep for the future

// RunAnalyzer runs the analyzer for the given code and returns the issues
func RunAnalyzer(code string, analyzer *analysis.Analyzer) ([]Issue, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var issues []Issue
	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
		Report: func(d analysis.Diagnostic) {
			pos := fset.Position(d.Pos)
			end := fset.Position(d.End)

			issues = append(issues, Issue{
				Rule:     analyzer.Name,
				Message:  d.Message,
				Category: d.Category,
				Start:    pos,
				End:      end,
			})
		},
	}

	_, err = analyzer.Run(pass)
	if err != nil {
		return nil, err
	}

	return issues, nil
}
