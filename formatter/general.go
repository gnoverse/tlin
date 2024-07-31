package formatter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

const tabWidth = 8

var (
	errorStyle      = color.New(color.FgRed, color.Bold)
	warningStyle    = color.New(color.FgHiYellow, color.Bold)
	ruleStyle       = color.New(color.FgYellow, color.Bold)
	fileStyle       = color.New(color.FgCyan, color.Bold)
	lineStyle       = color.New(color.FgBlue, color.Bold)
	messageStyle    = color.New(color.FgRed, color.Bold)
	suggestionStyle = color.New(color.FgGreen, color.Bold)
)

// GeneralIssueFormatter is a formatter for general lint issues.
type GeneralIssueFormatter struct{}

// Format formats a general lint issue into a human-readable string.
// It takes an Issue and a SourceCode snippet as input and returns a formatted string.
func (f *GeneralIssueFormatter) Format(
	issue tt.Issue,
	snippet *internal.SourceCode,
) string {
	var result strings.Builder

	lineIndex := issue.Start.Line - 1
	if lineIndex < 0 {
		lineIndex = 0
	}
	if lineIndex >= len(snippet.Lines) {
		lineIndex = len(snippet.Lines) - 1
	}

	lineNumberStr := fmt.Sprintf("%d", issue.Start.Line)
	padding := strings.Repeat(" ", len(lineNumberStr)-1)
	result.WriteString(lineStyle.Sprintf("  %s|\n", padding))

	if len(snippet.Lines) > 0 {
		line := expandTabs(snippet.Lines[lineIndex])
		result.WriteString(lineStyle.Sprintf("%d | ", issue.Start.Line))
		result.WriteString(line + "\n")

		visualColumn := calculateVisualColumn(line, issue.Start.Column)
		result.WriteString(lineStyle.Sprintf("  %s| ", padding))
		result.WriteString(strings.Repeat(" ", visualColumn))
		result.WriteString(messageStyle.Sprintf("^ %s\n\n", issue.Message))
	} else {
		result.WriteString(messageStyle.Sprintf("Unable to display line. File might be empty.\n"))
		result.WriteString(messageStyle.Sprintf("Issue: %s\n\n", issue.Message))
	}

	return result.String()
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
