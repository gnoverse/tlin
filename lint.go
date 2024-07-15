package lint

import (
	"encoding/json"
	"fmt"
	"go/token"
	"os"
	"os/exec"
	"strings"
)

// SourceCode stores the content of a source code file.
type SourceCode struct {
	Lines []string
}

// ReadSourceFile reads the content of a file and returns it as a `SourceCode` struct.
func ReadSourceCode(filename string) (*SourceCode, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	return &SourceCode{Lines: lines}, nil
}

// Issue represents a lint issue found in the code base.
type Issue struct {
	Rule     string
	Filename string
	Start    token.Position
	End      token.Position
	Message  string
}

// Engine manages the linting process.
type Engine struct{}

// NewEngine creates a new lint engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Run applies golangci-lint to the given file and returns a slice of issues.
func (e *Engine) Run(filename string) ([]Issue, error) {
	issues, err := runGolangciLint(filename)
	if err != nil {
		return nil, fmt.Errorf("error running golangci-lint: %w", err)
	}
	return issues, nil
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

func runGolangciLint(filename string) ([]Issue, error) {
	cmd := exec.Command("golangci-lint", "run", "--out-format=json", filename)
	output, _ := cmd.CombinedOutput() // avoid non-zero exit code

	var golangciResult golangciOutput
	if err := json.Unmarshal(output, &golangciResult); err != nil {
		return nil, fmt.Errorf("error unmarshaling golangci-lint output: %w", err)
	}

	var issues []Issue
	for _, gi := range golangciResult.Issues {
		issues = append(issues, Issue{
			Rule:     gi.FromLinter,
			Filename: gi.Pos.Filename,
			Start:    token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column},
			End:      token.Position{Filename: gi.Pos.Filename, Line: gi.Pos.Line, Column: gi.Pos.Column + 1}, // Set End to Start + 1 column
			Message:  gi.Text,
		})
	}

	return issues, nil
}

func FormatIssuesWithArrows(issues []Issue, sourceCode *SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		// Write issue location and message
		builder.WriteString(fmt.Sprintf("%s:%d:%d: %s: %s\n",
			issue.Filename, issue.Start.Line, issue.Start.Column, issue.Rule, issue.Message))

		// Write the problematic line
		line := sourceCode.Lines[issue.Start.Line-1]
		builder.WriteString(line + "\n")

		// Calculate the visual column, considering tabs
		visualStartColumn := calculateVisualColumn(line, issue.Start.Column)
		visualEndColumn := calculateVisualColumn(line, issue.End.Column)

		// Write the arrow pointing to the issue
		builder.WriteString(strings.Repeat(" ", visualStartColumn-1))
		arrowLength := visualEndColumn - visualStartColumn
		if arrowLength < 1 {
			arrowLength = 1
		}
		builder.WriteString(strings.Repeat("^", arrowLength))
		builder.WriteString("\n")
	}
	return builder.String()
}

func calculateVisualColumn(line string, column int) int {
	visualColumn := 0
	for i, ch := range line {
		if i+1 >= column {
			break
		}
		if ch == '\t' {
			visualColumn += 8 - (visualColumn % 8)
		} else {
			visualColumn++
		}
	}
	return visualColumn
}
