package lints

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"

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
	// find config file in current working directory
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %v", err)
	}

	configPath := filepath.Join(wd, ".golangci.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to find .golangci.yml file: %v", err)
	}

	cmd := exec.Command("golangci-lint", "run", "--out-format=json", "--config="+configPath, filename)
	output, err := cmd.CombinedOutput()

	// If golangci-lint finds issues, it returns a non-zero exit code,
	// so we ignore the error if there is json output.
	if err != nil && len(output) == 0 {
		return nil, fmt.Errorf("failed to run golangci-lint: %v", err)
	}

	var golangciResult golangciOutput
	err = json.Unmarshal(output, &golangciResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse golangci-lint output: %v", err)
	}

	issues := make([]tt.Issue, 0, len(golangciResult.Issues))
	for _, gi := range golangciResult.Issues {
		issues = append(issues, tt.Issue{
			Rule:     gi.FromLinter,
			Filename: gi.Pos.Filename,
			Start:    token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1},
			Message:  gi.Text,
			Severity: severity,
		})
	}

	return issues, nil
}
