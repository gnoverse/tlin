package internal

import (
	"fmt"
	"strings"
)

const (
	greenColor = "\033[32m"
	resetColor = "\033[0m"
	tabWidth   = 8
)

func FormatIssuesWithArrows(issues []Issue, sourceCode *SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		builder.WriteString(formatIssueHeader(issue))
		if issue.Rule == "unnecessary-else" {
			builder.WriteString(formatUnnecessaryElse(issue, sourceCode))
		} else {
			builder.WriteString(formatGeneralIssue(issue, sourceCode))
		}
	}
	return builder.String()
}

func formatIssueHeader(issue Issue) string {
	return fmt.Sprintf("error: %s\n --> %s\n", issue.Rule, issue.Filename)
}

func formatUnnecessaryElse(issue Issue, sourceCode *SourceCode) string {
	var result strings.Builder
	ifStartLine, elseEndLine := issue.Start.Line-2, issue.End.Line
	maxLineNumberStr := fmt.Sprintf("%d", elseEndLine)
	padding := strings.Repeat(" ", len(maxLineNumberStr)-1)

	result.WriteString(fmt.Sprintf("  %s|\n", padding))

	for i := ifStartLine; i <= elseEndLine; i++ {
		line := expandTabs(sourceCode.Lines[i-1])
		lineNumberStr := fmt.Sprintf("%d", i)
		linePadding := strings.Repeat(" ", len(maxLineNumberStr)-len(lineNumberStr))
		result.WriteString(fmt.Sprintf("%s%s | %s\n", linePadding, lineNumberStr, line))
	}

	result.WriteString(fmt.Sprintf("  %s| %s\n", padding, strings.Repeat("~", len(sourceCode.Lines[elseEndLine-1])-1)))
	result.WriteString(fmt.Sprintf("  %s| %s\n\n", padding, issue.Message))
	return result.String()
}

func formatGeneralIssue(issue Issue, sourceCode *SourceCode) string {
	var result strings.Builder
	lineNumberStr := fmt.Sprintf("%d", issue.Start.Line)
	padding := strings.Repeat(" ", len(lineNumberStr)-1)
	result.WriteString(fmt.Sprintf("  %s|\n", padding))
	line := expandTabs(sourceCode.Lines[issue.Start.Line-1])
	result.WriteString(fmt.Sprintf("%d | %s\n", issue.Start.Line, line))
	visualColumn := calculateVisualColumn(line, issue.Start.Column)
	result.WriteString(fmt.Sprintf("  %s| %s^ %s\n\n", padding, strings.Repeat(" ", visualColumn), issue.Message))
	return result.String()
}

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
