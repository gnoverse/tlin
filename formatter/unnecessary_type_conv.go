package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

// type UnnecessaryTypeConversionFormatter struct{}

// func (f *UnnecessaryTypeConversionFormatter) Format(
// 	issue tt.Issue,
// 	snippet *internal.SourceCode,
// ) string {
// 	var result strings.Builder

// 	lineNumberStr := fmt.Sprintf("%d", issue.Start.Line)
// 	padding := strings.Repeat(" ", len(lineNumberStr)-1)
// 	result.WriteString(lineStyle.Sprintf("  %s|\n", padding))

// 	line := expandTabs(snippet.Lines[issue.Start.Line-1])
// 	result.WriteString(lineStyle.Sprintf("%d | ", issue.Start.Line))
// 	result.WriteString(line + "\n")

// 	visualColumn := calculateVisualColumn(line, issue.Start.Column)
// 	result.WriteString(lineStyle.Sprintf("  %s| ", padding))
// 	result.WriteString(strings.Repeat(" ", visualColumn))
// 	result.WriteString(messageStyle.Sprintf("^ %s\n\n", issue.Message))

// 	buildSuggestion(&result, issue, lineStyle, suggestionStyle, issue.Start.Line)
// 	buildNote(&result, issue, suggestionStyle)

// 	return result.String()
// }

type UnnecessaryTypeConversionFormatter struct{}

func (f *UnnecessaryTypeConversionFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	var result strings.Builder

	// 1. Calculate dimensions
	startLine := issue.Start.Line
	endLine := issue.Start.Line // Only one line for type conversion
	maxLineNumWidth := calculateMaxLineNumWidth(endLine)

	// 2. Write header
	padding := strings.Repeat(" ", maxLineNumWidth+1)
	result.WriteString(lineStyle.Sprintf("%s|\n", padding))

	// 3. Write code snippet
	line := expandTabs(snippet.Lines[startLine-1])
	lineNum := fmt.Sprintf("%*d", maxLineNumWidth, startLine)
	result.WriteString(lineStyle.Sprintf("%s | %s\n", lineNum, line))

	// 4. Write underline and message
	visualColumn := calculateVisualColumn(line, issue.Start.Column)
	result.WriteString(lineStyle.Sprintf("%s| ", padding))
	result.WriteString(strings.Repeat(" ", visualColumn))
	result.WriteString(messageStyle.Sprintf("^ %s\n\n", issue.Message))

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
