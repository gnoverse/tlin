package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
)

// UnnecessaryElseFormatter is a formatter specifically designed for the "unnecessary-else" rule.
type UnnecessaryElseFormatter struct{}

func (f *UnnecessaryElseFormatter) Format(
	issue internal.Issue,
	snippet *internal.SourceCode,
) string {
	var result strings.Builder
	ifStartLine, elseEndLine := issue.Start.Line-2, issue.End.Line
	maxLineNumberStr := fmt.Sprintf("%d", elseEndLine)
	padding := strings.Repeat(" ", len(maxLineNumberStr)-1)

	result.WriteString(lineStyle.Sprintf("  %s|\n", padding))

	maxLen := 0
	for i := ifStartLine; i <= elseEndLine; i++ {
		if len(snippet.Lines[i-1]) > maxLen {
			maxLen = len(snippet.Lines[i-1])
		}
		line := expandTabs(snippet.Lines[i-1])
		lineNumberStr := fmt.Sprintf("%d", i)
		linePadding := strings.Repeat(" ", len(maxLineNumberStr)-len(lineNumberStr))
		result.WriteString(lineStyle.Sprintf("%s%s | ", linePadding, lineNumberStr))
		result.WriteString(line + "\n")
	}

	result.WriteString(lineStyle.Sprintf("  %s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n", strings.Repeat("~", maxLen)))
	result.WriteString(lineStyle.Sprintf("  %s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n\n", issue.Message))

	return result.String()
}
