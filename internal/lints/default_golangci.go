package lints

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"

	tt "github.com/gnolang/tlin/internal/types"
)

func ParseFile(filename string, content []byte) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	var node *ast.File
	var err error
	if content == nil {
		node, err = parser.ParseFile(fset, filename, nil, parser.ParseComments)
	} else {
		node, err = parser.ParseFile(fset, filename, content, parser.ParseComments)
	}
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

func RunGolangciLint(filename string, _ *ast.File, _ *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	cmd := exec.Command("golangci-lint", "run", "--config=./.golangci.yml", "--out-format=json", filename)
	output, _ := cmd.CombinedOutput()

	var golangciResult golangciOutput

	// Best-effort decode: golangci-lint sometimes prints non-JSON output
	// (e.g. when source contains gno package imports like p/demo, r/demo,
	// std). When that happens we fall through with whatever Issues did
	// parse and surface the error so the failure is observable instead
	// of silently swallowed. EPR-1 promotes this stderr line to a
	// returned error that the engine logs via WithLogger.
	if err := json.Unmarshal(output, &golangciResult); err != nil {
		fmt.Fprintf(os.Stderr, "tlin: golangci-lint output decode failed for %s: %v\n", filename, err)
	}

	issues := make([]tt.Issue, 0, len(golangciResult.Issues))
	for _, gi := range golangciResult.Issues {
		issues = append(issues, tt.Issue{
			Rule:     gi.FromLinter,
			Filename: gi.Pos.Filename, // Use the filename from golangci-lint output
			Start:    token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1},
			Message:  gi.Text,
			Severity: severity,
		})
	}

	return issues, nil
}
