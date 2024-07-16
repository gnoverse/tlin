package lint

import (
	"fmt"
	"strings"
)

// expandTabs replaces tab characters with spaces, considering a tab width of 8
func expandTabs(line string) string {
	var expanded strings.Builder
	column := 0
	for _, ch := range line {
		if ch == '\t' {
			// Add the appropriate number of spaces for a tab character
			spaceCount := 8 - (column % 8)
			for i := 0; i < spaceCount; i++ {
				expanded.WriteByte(' ')
				column++
			}
		} else {
			expanded.WriteRune(ch)
			column++
		}
	}
	return expanded.String()
}

func FormatIssuesWithArrows(issues []Issue, sourceCode *SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		// Write error header
		builder.WriteString(fmt.Sprintf("error: %s\n", issue.Rule))
		builder.WriteString(fmt.Sprintf(" --> %s\n", issue.Filename))
		builder.WriteString("  |\n")

		// Write the problematic line with line number
		line := sourceCode.Lines[issue.Start.Line-1]
		expandedLine := expandTabs(line)
		builder.WriteString(fmt.Sprintf("%d | %s\n", issue.Start.Line, expandedLine))

		// Calculate the visual column, considering expanded tabs
		visualColumn := calculateVisualColumn(expandedLine, issue.Start.Column)

		// Write the arrow pointing to the issue
		builder.WriteString("  | ")
		builder.WriteString(strings.Repeat(" ", visualColumn))
		builder.WriteString("^ ")
		builder.WriteString(issue.Message)
		builder.WriteString("\n\n")
	}
	return builder.String()
}

func calculateVisualColumn(line string, column int) int {
	visualColumn := 0
	for i, ch := range line {
		if i+1 >= column {
			break
		}
		if ch == '\t' {
			visualColumn += 8 - (visualColumn % 8)
		} else {
			visualColumn++
		}
	}
	return visualColumn
}
