package internal

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

const (
	tabWidth = 8
)

var (
	errorStyle   = color.New(color.FgRed, color.Bold)
	ruleStyle    = color.New(color.FgYellow, color.Bold)
	fileStyle    = color.New(color.FgCyan, color.Bold)
	lineStyle    = color.New(color.FgBlue, color.Bold)
	messageStyle = color.New(color.FgRed, color.Bold)
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
	return errorStyle.Sprint("error: ") + ruleStyle.Sprint(issue.Rule) + "\n" +
		lineStyle.Sprint(" --> ") + fileStyle.Sprint(issue.Filename) + "\n"
}

func formatUnnecessaryElse(issue Issue, sourceCode *SourceCode) string {
	var result strings.Builder
	ifStartLine, elseEndLine := issue.Start.Line-2, issue.End.Line
	maxLineNumberStr := fmt.Sprintf("%d", elseEndLine)
	padding := strings.Repeat(" ", len(maxLineNumberStr)-1)

	result.WriteString(lineStyle.Sprintf("  %s|\n", padding))

	maxLen := 0
	for i := ifStartLine; i <= elseEndLine; i++ {
		if len(sourceCode.Lines[i-1]) > maxLen {
			maxLen = len(sourceCode.Lines[i-1])
		}
		line := expandTabs(sourceCode.Lines[i-1])
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

func formatGeneralIssue(issue Issue, sourceCode *SourceCode) string {
	var result strings.Builder

	lineNumberStr := fmt.Sprintf("%d", issue.Start.Line)
	padding := strings.Repeat(" ", len(lineNumberStr)-1)
	result.WriteString(lineStyle.Sprintf("  %s|\n", padding))

	line := expandTabs(sourceCode.Lines[issue.Start.Line-1])
	result.WriteString(lineStyle.Sprintf("%d | ", issue.Start.Line))
	result.WriteString(line + "\n")

	visualColumn := calculateVisualColumn(line, issue.Start.Column)
	result.WriteString(lineStyle.Sprintf("  %s| ", padding))
	result.WriteString(strings.Repeat(" ", visualColumn))
	result.WriteString(messageStyle.Sprintf("^ %s\n\n", issue.Message))

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
