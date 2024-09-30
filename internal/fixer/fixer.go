package fixer

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"

	tt "github.com/gnolang/tlin/internal/types"
)

type Fixer struct {
	DryRun        bool
	MinConfidence float64 // threshold for fixing issues
}

func New(dryRun bool, threshold float64) *Fixer {
	return &Fixer{
		DryRun:        dryRun,
		MinConfidence: threshold,
	}
}

func (f *Fixer) Fix(filename string, issues []tt.Issue) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].End.Offset > issues[j].End.Offset
	})

	lines := strings.Split(string(content), "\n")

	for _, issue := range issues {
		if issue.Confidence < f.MinConfidence {
			continue
		}

		if f.DryRun {
			fmt.Printf("Would fix issue in %s at line %d: %s\n", filename, issue.Start.Line, issue.Message)
			fmt.Printf("Suggestion:\n%s\n", issue.Suggestion)
			continue
		}

		startLine := issue.Start.Line - 1
		endLine := issue.End.Line - 1

		indent := f.extractIndent(lines[startLine])
		suggestion := applyIndent(issue.Suggestion, indent, issue.Start)

		lines = append(lines[:startLine], append([]string{suggestion}, lines[endLine+1:]...)...)
	}

	if !f.DryRun {
		newContent := strings.Join(lines, "\n")

		fset := token.NewFileSet()
		astFile, err := parser.ParseFile(fset, filename, newContent, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse file: %w", err)
		}

		var buf bytes.Buffer
		if err := format.Node(&buf, fset, astFile); err != nil {
			return fmt.Errorf("failed to format file: %w", err)
		}

		err = os.WriteFile(filename, buf.Bytes(), 0o644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Printf("Fixed issues in %s\n", filename)
	}

	return nil
}

func (f *Fixer) extractIndent(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func applyIndent(content, suggestion string, start token.Position) string {
	lines := strings.Split(content, "\n")
	sugLines := strings.Split(suggestion, "\n")
	offset := getOffset(lines, start.Line-1)

	// apply the offset to all suggestion lines
	for i := range sugLines {
		sugLines[i] = strings.Repeat(" ", offset) + sugLines[i]
	}

	// replace the lines in the original content
	for i := 0; i < len(sugLines); i++ {
		if start.Line-1+i < len(lines) {
			lines[start.Line-1+i] = sugLines[i]
		}
	}

	return strings.Join(lines, "\n")
}

// get the offset (indentation) of a line
func getOffset(lines []string, lineIndex int) int {
	if lineIndex < 0 || lineIndex >= len(lines) {
		return 0
	}
	return len(lines[lineIndex]) - len(strings.TrimLeft(lines[lineIndex], " \t"))
}
