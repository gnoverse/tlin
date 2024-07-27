package lints

import (
	"encoding/json"
	_ "fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"

	tt "github.com/gnoswap-labs/lint/internal/types"
)

func ParseFile(filename string) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	return node, fset, nil
}

type golangciOutput struct {
	Issues []struct {
		FromLinter string `json:"FromLinter"`
		Text       string `json:"Text"`
		Pos        struct {
			Filename string `json:"Filename"`
			Line     int    `json:"Line"`
			Column   int    `json:"Column"`
		} `json:"Pos"`
	} `json:"Issues"`
}

func RunGolangciLint(filename string) ([]tt.Issue, error) {
	cmd := exec.Command("golangci-lint", "run", "--disable=gosimple", "--out-format=json", filename)
	output, _ := cmd.CombinedOutput()

	var golangciResult golangciOutput

	// @notJoon: Ignore Unmarshal error. We cannot unmarshal the output of golangci-lint
	// when source code contains gno package imports (i.e. p/demo, r/demo, std). [07/25/24]
	json.Unmarshal(output, &golangciResult)

	var issues []tt.Issue
	for _, gi := range golangciResult.Issues {
		issues = append(issues, tt.Issue{
			Rule:     gi.FromLinter,
			Filename: gi.Pos.Filename, // Use the filename from golangci-lint output
			Start:    token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1},
			Message:  gi.Text,
		})
	}

	return issues, nil
}
