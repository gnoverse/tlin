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

const (
	defaultFilePermissions = 0o644
)

// Fixer handles the fixing of issues in Gno code files.
type Fixer struct {
	buffer        bytes.Buffer
	MinConfidence float64
	DryRun        bool
}

// New creates a new Fixer instance.
func New(dryRun bool, threshold float64) *Fixer {
	return &Fixer{
		DryRun:        dryRun,
		MinConfidence: threshold,
	}
}

// Fix applies fixes to the given file based on the provided issues.
func (f *Fixer) Fix(filename string, issues []tt.Issue) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	sortIssuesByEndOffset(issues)

	for _, issue := range issues {
		if f.DryRun {
			f.printDryRunInfo(filename, issue)
			continue
		}

		lines = f.applyFix(lines, issue)
	}

	if !f.DryRun {
		if err := f.writeFixedContent(filename, lines); err != nil {
			return err
		}
		fmt.Printf("Fixed issues in %s\n", filename)
	}

	return nil
}

func (f *Fixer) printDryRunInfo(filename string, issue tt.Issue) {
	fmt.Printf("Would fix issue in %s at line %d: %s\n", filename, issue.Start.Line, issue.Message)
	fmt.Printf("Suggestion:\n%s\n", issue.Suggestion)
}

func (f *Fixer) applyFix(lines []string, issue tt.Issue) []string {
	startLine := issue.Start.Line - 1
	endLine := issue.End.Line

	indent := extractIndent(lines[startLine])
	suggestion := applyIndent(issue.Suggestion, indent)

	fixedLines := strings.Split(suggestion, "\n")

	modified := make([]string, 0, len(lines[:startLine])+len(fixedLines)+len(lines[endLine:]))
	modified = append(modified, lines[:startLine]...)
	modified = append(modified, fixedLines...)
	modified = append(modified, lines[endLine:]...)

	original := strings.Join(lines, "\n")
	fixed := strings.Join(modified, "\n")

	// do not apply the fix if the AST equivalence check fails
	checker := NewContentBasedCFGChecker(f.MinConfidence, false)
	eq, report, err := checker.CheckEquivalence(original, fixed)
	if err != nil {
		fmt.Printf("AST equivalence check error at line %d: %v\n", issue.Start.Line, err)
		return lines
	}

	if !eq {
		fmt.Printf("AST equivalence check failed at line %d: %s\n", issue.Start.Line, report)
		return lines
	}

	return modified
}

func (f *Fixer) writeFixedContent(filename string, lines []string) error {
	f.buffer.Reset()
	for i, line := range lines {
		f.buffer.WriteString(line)
		if i < len(lines)-1 {
			f.buffer.WriteByte('\n')
		}
	}

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, filename, f.buffer.Bytes(), parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	f.buffer.Reset()
	if err := format.Node(&f.buffer, fset, astFile); err != nil {
		return fmt.Errorf("failed to format file: %w", err)
	}

	if err := os.WriteFile(filename, f.buffer.Bytes(), defaultFilePermissions); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// sorts the issues by the end offset of the issue.
// By doing this, we ensure that the issues are applied in the correct order.
func sortIssuesByEndOffset(issues []tt.Issue) {
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].End.Offset > issues[j].End.Offset
	})
}

// extractIndent extracts the indentation from the first line of the issue.
func extractIndent(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// applyIndent applies the indentation to each line of the content.
// It splits the content into lines, prepends the given indent to each line,
// and then joins them back together with newlines.
func applyIndent(content, indent string) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = indent + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}
