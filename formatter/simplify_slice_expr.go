package formatter

import (
    "fmt"
    "strings"

    "github.com/gnoswap-labs/lint/internal"
)

type SimplifySliceExpressionFormatter struct{}

func (f *SimplifySliceExpressionFormatter) Format(
    issue internal.Issue,
    snippet *internal.SourceCode,
) string {
    var result strings.Builder

    lineNumberStr := fmt.Sprintf("%d", issue.Start.Line)
    padding := strings.Repeat(" ", len(lineNumberStr)-1)
    result.WriteString(lineStyle.Sprintf("  %s|\n", padding))

    line := expandTabs(snippet.Lines[issue.Start.Line-1])
    result.WriteString(lineStyle.Sprintf("%d | ", issue.Start.Line))
    result.WriteString(line + "\n")

    visualColumn := calculateVisualColumn(line, issue.Start.Column)
    result.WriteString(lineStyle.Sprintf("  %s| ", padding))
    result.WriteString(strings.Repeat(" ", visualColumn))
    result.WriteString(messageStyle.Sprintf("^ %s\n\n", issue.Message))

    result.WriteString(suggestionStyle.Sprintf("Suggestion:\n"))
    result.WriteString(lineStyle.Sprintf("%d | ", issue.Start.Line))
    result.WriteString(fmt.Sprintf("%s\n", issue.Suggestion))

    result.WriteString(suggestionStyle.Sprint("Note: "))
    result.WriteString(fmt.Sprintf("%s\n", issue.Message))

    return result.String()
}
