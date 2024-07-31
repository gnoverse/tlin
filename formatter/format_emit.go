package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

type EmitFormatFormatter struct{}

func (f *EmitFormatFormatter) Format(
	issue tt.Issue,
	snippet *internal.SourceCode,
) string {
	var result strings.Builder

	result.WriteString(formatIssueHeader(issue))

	maxLineNumWidth := calculateMaxLineNumWidth(issue.End.Line)
	padding := strings.Repeat(" ", maxLineNumWidth+1)

	startLine := issue.Start.Line
	endLine := issue.End.Line
	for i := startLine; i <= endLine; i++ {
		line := expandTabs(snippet.Lines[i-1])
		result.WriteString(lineStyle.Sprintf("%s%d | ", padding[:maxLineNumWidth-len(fmt.Sprintf("%d", i))], i))
		result.WriteString(line + "\n")
	}

	result.WriteString(lineStyle.Sprintf("%s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n", strings.Repeat("~", calculateMaxLineLength(snippet.Lines, startLine, endLine))))
	result.WriteString(lineStyle.Sprintf("%s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n\n", issue.Message))

	buildSuggestion(&result, issue, lineStyle, suggestionStyle, startLine)

	// buildNote(&result, issue, suggestionStyle)

	return result.String()
}
