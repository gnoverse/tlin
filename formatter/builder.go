package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

// rule set
const (
	UnnecessaryElse     = "unnecessary-else"
	UnnecessaryTypeConv = "unnecessary-type-conversion"
	SimplifySliceExpr   = "simplify-slice-range"
	CycloComplexity     = "high-cyclomatic-complexity"
	EmitFormat          = "emit-format"
	SliceBound          = "slice-bounds-check"
)

type IssueFormatterBuilder struct {
	result strings.Builder
	issue tt.Issue
	snippet *internal.SourceCode
}

func NewIssueFormatterBuilder(issue tt.Issue, snippet *internal.SourceCode) *IssueFormatterBuilder {
    return &IssueFormatterBuilder{
        issue:   issue,
        snippet: snippet,
    }
}

func (b *IssueFormatterBuilder) AddHeader() *IssueFormatterBuilder {
	// add error type and rule name
    b.result.WriteString(errorStyle.Sprint("error: "))
    b.result.WriteString(ruleStyle.Sprintln(b.issue.Rule))

	// add file name
    b.result.WriteString(lineStyle.Sprint(" --> "))
    b.result.WriteString(fileStyle.Sprintln(b.issue.Filename))

	// add separator
    maxLineNumWidth := calculateMaxLineNumWidth(b.issue.End.Line)
    padding := strings.Repeat(" ", maxLineNumWidth+1)
    b.result.WriteString(lineStyle.Sprintf("%s|\n", padding))

    return b
}

func (b *IssueFormatterBuilder) AddCodeSnippet() *IssueFormatterBuilder {
    startLine := b.issue.Start.Line
    endLine := b.issue.End.Line
    maxLineNumWidth := calculateMaxLineNumWidth(endLine)

    for i := startLine; i <= endLine; i++ {
		// check that the line number does not go out of range of snippet.Lines
        if i-1 < 0 || i-1 >= len(b.snippet.Lines) {
            continue
        }

        line := expandTabs(b.snippet.Lines[i-1])
        lineNum := fmt.Sprintf("%*d", maxLineNumWidth, i)
        
        b.result.WriteString(lineStyle.Sprintf("%s | ", lineNum))
        b.result.WriteString(line + "\n")
    }

    b.result.WriteString("\n")

    return b
}

func (b *IssueFormatterBuilder) AddUnderlineAndMessage() *IssueFormatterBuilder {
    startLine := b.issue.Start.Line
    endLine := b.issue.End.Line
    maxLineNumWidth := calculateMaxLineNumWidth(endLine)
    padding := strings.Repeat(" ", maxLineNumWidth+1)

    b.result.WriteString(lineStyle.Sprintf("%s| ", padding))

	// draw underline from start column to end column
    underlineStart := calculateVisualColumn(b.snippet.Lines[startLine-1], b.issue.Start.Column)
    underlineEnd := calculateVisualColumn(b.snippet.Lines[endLine-1], b.issue.End.Column)
    underlineLength := underlineEnd - underlineStart + 1
    
    b.result.WriteString(strings.Repeat(" ", underlineStart))
    b.result.WriteString(messageStyle.Sprintf("%s\n", strings.Repeat("~", underlineLength)))

    b.result.WriteString(lineStyle.Sprintf("%s| ", padding))
    b.result.WriteString(messageStyle.Sprintf("%s\n\n", b.issue.Message))

    return b
}

func (b *IssueFormatterBuilder) AddSuggestion() *IssueFormatterBuilder {
    if b.issue.Suggestion == "" {
        return b
    }

    maxLineNumWidth := calculateMaxLineNumWidth(b.issue.End.Line)
    padding := strings.Repeat(" ", maxLineNumWidth+1)

    b.result.WriteString(suggestionStyle.Sprint("Suggestion:\n"))
    b.result.WriteString(lineStyle.Sprintf("%s|\n", padding))

    // 제안사항을 여러 줄로 나누어 처리
    suggestionLines := strings.Split(b.issue.Suggestion, "\n")
    for i, line := range suggestionLines {
        lineNum := fmt.Sprintf("%*d", maxLineNumWidth, b.issue.Start.Line+i)
        b.result.WriteString(lineStyle.Sprintf("%s | %s\n", lineNum, line))
    }

    b.result.WriteString(lineStyle.Sprintf("%s|\n", padding))
    b.result.WriteString("\n") // 가독성을 위한 빈 줄 추가

    return b
}

func (b *IssueFormatterBuilder) AddNote() *IssueFormatterBuilder {
    if b.issue.Note == "" {
        return b
    }

    b.result.WriteString(suggestionStyle.Sprint("Note: "))
    b.result.WriteString(b.issue.Note)
    b.result.WriteString("\n\n")

    return b
}

type BaseFormatter struct{}

func (f *BaseFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
    builder := NewIssueFormatterBuilder(issue, snippet)
    return builder.
        AddHeader().
        AddCodeSnippet().
        AddUnderlineAndMessage().
        AddSuggestion().
        AddNote().
        Build()
}

func (b *IssueFormatterBuilder) Build() string {
	return b.result.String()
}

func calculateMaxLineNumWidth(endLine int) int {
	return len(fmt.Sprintf("%d", endLine))
}

// expandTabs replaces tab characters('\t') with spaces.
// Assuming a table width of 8.
func expandTabs(line string) string {
	var expanded strings.Builder
	for i, ch := range line {
		if ch == '\t' {
			spaceCount := tabWidth - (i % tabWidth)
			expanded.WriteString(strings.Repeat(" ", spaceCount))
		} else {
			expanded.WriteRune(ch)
		}
	}
	return expanded.String()
}

// calculateVisualColumn calculates the visual column position
// in a string. taking into account tab characters.
func calculateVisualColumn(line string, column int) int {
	visualColumn := 0
	for i, ch := range line {
		if i+1 == column {
			break
		}
		if ch == '\t' {
			visualColumn += tabWidth - (visualColumn % tabWidth)
		} else {
			visualColumn++
		}
	}
	return visualColumn
}