package formatter

import (
	"strings"

	"github.com/gnoswap-labs/lint/internal"
)

// rule set
const (
	UnnecessaryElse = "unnecessary-else"
	SimplifySliceExpr = "simplify-slice-range"
)

// IssueFormatter is the interface that wraps the Format method.
// Implementations of this interface are responsible for formatting specific types of lint issues.
type IssueFormatter interface {
	Format(issue internal.Issue, snippet *internal.SourceCode) string
}

// FormatIssuesWithArrows formats a slice of issues into a human-readable string.
// It uses the appropriate formatter for each issue based on its rule.
func FormatIssuesWithArrows(issues []internal.Issue, snippet *internal.SourceCode) string {
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
	default:
		return &GeneralIssueFormatter{}
	}
}

// formatIssueHeader creates a formatted header string for a given issue.
// The header includes the rule and the filename. (e.g. "error: unused-variable\n --> test.go")
func formatIssueHeader(issue internal.Issue) string {
	return errorStyle.Sprint("error: ") + ruleStyle.Sprint(issue.Rule) + "\n" +
		lineStyle.Sprint(" --> ") + fileStyle.Sprint(issue.Filename) + "\n"
}
