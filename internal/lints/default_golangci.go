package lints

import (
	"encoding/json"
	"fmt"
	"go/token"
	"os/exec"
)

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

func RunGolangciLint(filename string) ([]Issue, error) {
	cmd := exec.Command("golangci-lint", "run", "--disable=gosimple", "--out-format=json", filename)
	output, _ := cmd.CombinedOutput()

	var golangciResult golangciOutput
	if err := json.Unmarshal(output, &golangciResult); err != nil {
		return nil, fmt.Errorf("error unmarshaling golangci-lint output: %w", err)
	}

	var issues []Issue
	for _, gi := range golangciResult.Issues {
		issues = append(issues, Issue{
			Rule:     gi.FromLinter,
			Filename: gi.Pos.Filename, // Use the filename from golangci-lint output
			Start:    token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1},
			Message:  gi.Text,
		})
	}

	return issues, nil
}
