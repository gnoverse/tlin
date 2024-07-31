package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

type SliceBoundsCheckFormatter struct{}

func (f *SliceBoundsCheckFormatter) Format(
	issue tt.Issue,
	snippet *internal.SourceCode,
) string {
	var result strings.Builder

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

	result.WriteString(warningStyle.Sprint("warning: "))
	if issue.Category == "index-access" {
		result.WriteString("Index access without bounds checking can lead to runtime panics.\n")
	} else if issue.Category == "slice-expression" {
		result.WriteString("Slice expressions without proper length checks may cause unexpected behavior.\n\n")
	}

	return result.String()
}
