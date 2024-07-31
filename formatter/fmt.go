package formatter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

// rule set
const (
	UnnecessaryElse     = "unnecessary-else"
	UnnecessaryTypeConv = "unnecessary-type-conversion"
	SimplifySliceExpr   = "simplify-slice-range"
	CycloComplexity     = "high-cyclomatic-complexity"
	EmitFormat          = "emit-format"
)

// IssueFormatter is the interface that wraps the Format method.
// Implementations of this interface are responsible for formatting specific types of lint issues.
type IssueFormatter interface {
	Format(issue tt.Issue, snippet *internal.SourceCode) string
}

// FormatIssuesWithArrows formats a slice of issues into a human-readable string.
// It uses the appropriate formatter for each issue based on its rule.
func FormatIssuesWithArrows(issues []tt.Issue, snippet *internal.SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		builder.WriteString(formatIssueHeader(issue))
		formatter := getFormatter(issue.Rule)
		builder.WriteString(formatter.Format(issue, snippet))
	}
	return builder.String()
}

// getFormatter is a factory function that returns the appropriate IssueFormatter
// based on the given rule.
// If no specific formatter is found for the given rule, it returns a GeneralIssueFormatter.
func getFormatter(rule string) IssueFormatter {
	switch rule {
	case UnnecessaryElse:
		return &UnnecessaryElseFormatter{}
	case SimplifySliceExpr:
		return &SimplifySliceExpressionFormatter{}
	case UnnecessaryTypeConv:
		return &UnnecessaryTypeConversionFormatter{}
	case CycloComplexity:
		return &CyclomaticComplexityFormatter{}
	case EmitFormat:
		return &EmitFormatFormatter{}
	default:
		return &GeneralIssueFormatter{}
	}
}

// formatIssueHeader creates a formatted header string for a given issue.
// The header includes the rule and the filename. (e.g. "error: unused-variable\n --> test.go")
func formatIssueHeader(issue tt.Issue) string {
	return errorStyle.Sprint("error: ") + ruleStyle.Sprint(issue.Rule) + "\n" +
		lineStyle.Sprint(" --> ") + fileStyle.Sprint(issue.Filename) + "\n"
}

func buildSuggestion(result *strings.Builder, issue tt.Issue, lineStyle, suggestionStyle *color.Color, startLine int) {
	maxLineNumWidth := calculateMaxLineNumWidth(issue.End.Line)
	padding := strings.Repeat(" ", maxLineNumWidth)

	result.WriteString(suggestionStyle.Sprintf("Suggestion:\n"))
	for i, line := range strings.Split(issue.Suggestion, "\n") {
		lineNum := fmt.Sprintf("%d", startLine+i)

		if maxLineNumWidth < len(lineNum) {
			maxLineNumWidth = len(lineNum)
		}

		result.WriteString(lineStyle.Sprintf("%s%s | ", padding[:maxLineNumWidth-len(lineNum)], lineNum))
		result.WriteString(fmt.Sprintf("%s\n", line))
	}
	result.WriteString("\n")
}

func buildNote(result *strings.Builder, issue tt.Issue, suggestionStyle *color.Color) {
	result.WriteString(suggestionStyle.Sprint("Note: "))
	result.WriteString(fmt.Sprintf("%s\n", issue.Note))
	result.WriteString("\n")
}

func calculateMaxLineNumWidth(endLine int) int {
	return len(fmt.Sprintf("%d", endLine))
}

func calculateMaxLineLength(lines []string, start, end int) int {
	maxLen := 0
	for i := start - 1; i < end; i++ {
		if len(lines[i]) > maxLen {
			maxLen = len(lines[i])
		}
	}
	return maxLen
}
