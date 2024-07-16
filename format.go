package lint

import (
	"fmt"
	"strings"
)

// FormatIssuesWithArrows formats a list of issues with the corresponding lines of source code
// and points to the specific location of each issue using arrows. It considers the visual
// column of each issue, adjusting for tabs and line number padding.
//
// Parameters:
// - issues: A slice of Issue structs, each representing a linting issue.
// - sourceCode: A pointer to the SourceCode struct containing the lines of code.
//
// Returns:
// - A formatted string highlighting the issues with arrows pointing to their locations.
func FormatIssuesWithArrows(issues []Issue, sourceCode *SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		// calculate padding for line number
		lineNumberStr := fmt.Sprintf("%d", issue.Start.Line)
		lineNumberPadding := strings.Repeat(" ", len(lineNumberStr)-1)

		// write error header with adjusted indentation
		builder.WriteString(fmt.Sprintf("error: %s\n", issue.Rule))
		builder.WriteString(fmt.Sprintf(" --> %s\n", issue.Filename))
		builder.WriteString(fmt.Sprintf("  %s|\n", lineNumberPadding))

		// write the problematic line with line number
		line := sourceCode.Lines[issue.Start.Line-1]
		expandedLine := expandTabs(line)
		builder.WriteString(fmt.Sprintf("%d | %s%s\n", issue.Start.Line, lineNumberPadding, expandedLine))

		// calculate the visual column, considering expanded tabs and line number padding
		visualColumn := calculateVisualColumn(expandedLine, issue.Start.Column) + len(lineNumberPadding)

		// write the arrow pointing to the issue
		builder.WriteString(fmt.Sprintf("  %s| ", lineNumberPadding))
		builder.WriteString(strings.Repeat(" ", visualColumn))
		builder.WriteString("^ ")
		builder.WriteString(issue.Message)
		builder.WriteString("\n\n")
	}
	return builder.String()
}

// expandTabs replaces tab characters with spaces, considering a tab width of 8.
// It ensures that each tab character is expanded to the appropriate number of spaces
// based on its position in the line.
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

// calculateVisualColumn computes the visual column index for a given source code line,
// considering tabs and their expanded width.
//
// Parameters:
// - line: A string representing a single line of source code.
// - column: The 1-based column index of the issue in the source code.
//
// Returns:
// - The 0-based visual column index accounting for tab expansion.
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
