package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

// UnnecessaryElseFormatter is a formatter specifically designed for the "unnecessary-else" rule.
type UnnecessaryElseFormatter struct{}

func (f *UnnecessaryElseFormatter) Format(
	issue tt.Issue,
	snippet *internal.SourceCode,
) string {
	var result strings.Builder
	ifStartLine, elseEndLine := issue.Start.Line-2, issue.End.Line

	code := strings.Join(snippet.Lines, "\n")
	problemSnippet := internal.ExtractSnippet(issue, code, ifStartLine-1, elseEndLine-1)
	suggestion, err := internal.RemoveUnnecessaryElse(problemSnippet)
	if err != nil {
		suggestion = problemSnippet
	}

	maxLineNumWidth := calculateMaxLineNumWidth(elseEndLine)
	padding := strings.Repeat(" ", maxLineNumWidth-1)
	result.WriteString(lineStyle.Sprintf("  %s|\n", padding))

	maxLen := calculateMaxLineLength(snippet.Lines, ifStartLine, elseEndLine)
	for i := ifStartLine; i <= elseEndLine; i++ {
		line := expandTabs(snippet.Lines[i-1])
		lineNumberStr := fmt.Sprintf("%*d", maxLineNumWidth, i)
		result.WriteString(lineStyle.Sprintf("%s | ", lineNumberStr))
		result.WriteString(line + "\n")
	}

	result.WriteString(lineStyle.Sprintf("  %s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n", strings.Repeat("~", maxLen)))
	result.WriteString(lineStyle.Sprintf("  %s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n\n", issue.Message))

	result.WriteString(formatSuggestion(issue, suggestion, ifStartLine))
	result.WriteString("\n")

	return result.String()
}

func formatSuggestion(issue tt.Issue, improvedSnippet string, startLine int) string {
	var result strings.Builder
	lines := strings.Split(improvedSnippet, "\n")
	maxLineNumWidth := calculateMaxLineNumWidth(issue.End.Line)

	result.WriteString(suggestionStyle.Sprint("Suggestion:\n"))

	for i, line := range lines {
		lineNum := fmt.Sprintf("%*d", maxLineNumWidth, startLine+i)
		result.WriteString(lineStyle.Sprintf("%s | ", lineNum))
		result.WriteString(fmt.Sprintln(line))
	}

	// Add a note explaining the improvement
	result.WriteString("\n")
	result.WriteString(suggestionStyle.Sprint("Note: "))
	result.WriteString("Unnecessary 'else' block removed.\n")
	result.WriteString("The code inside the 'else' block has been moved outside, as it will only be executed when the 'if' condition is false.\n")

	return result.String()
}
