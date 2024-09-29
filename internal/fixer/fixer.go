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
		suggestion := f.applyIndent(issue.Suggestion, indent)

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

		err = os.WriteFile(filename, buf.Bytes(), 0644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		fmt.Printf("Fixed issues in %s\n", filename)
	}

	return nil
}

func (c *Fixer) extractIndent(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// TODO: better indentation handling
func (f *Fixer) applyIndent(code, indent string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}
