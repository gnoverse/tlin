package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

type SimplifySliceExpressionFormatter struct{}

func (f *SimplifySliceExpressionFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	var result strings.Builder

	// 1. Calculate dimensions
	startLine := issue.Start.Line
	endLine := issue.Start.Line // Only one line for slice expression
	maxLineNumWidth := calculateMaxLineNumWidth(endLine)
	maxLineLength := calculateMaxLineLength(snippet.Lines, startLine, endLine)

	// 2. Write header
	padding := strings.Repeat(" ", maxLineNumWidth+1)
	result.WriteString(lineStyle.Sprintf("%s|\n", padding))

	// 3. Write code snippet
	line := expandTabs(snippet.Lines[startLine-1])
	lineNum := fmt.Sprintf("%*d", maxLineNumWidth, startLine)
	result.WriteString(lineStyle.Sprintf("%s | %s\n", lineNum, line))

	// 4. Write underline and message
	result.WriteString(lineStyle.Sprintf("%s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n", strings.Repeat("~", maxLineLength)))
	result.WriteString(lineStyle.Sprintf("%s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n\n", issue.Message))

	// 5. Write suggestion
	if issue.Suggestion != "" {
		buildSuggestion(&result, issue, lineStyle, suggestionStyle, startLine)
	}

	// 6. Write note
	if issue.Note != "" {
		buildNote(&result, issue, suggestionStyle)
	}

	return result.String()
}
