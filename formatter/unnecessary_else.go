package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

type UnnecessaryElseFormatter struct{}

func (f *UnnecessaryElseFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	var result strings.Builder

	// 1. Calculate dimensions
	startLine := issue.Start.Line - 2 // Include the 'if' line
	endLine := issue.End.Line
	maxLineNumWidth := calculateMaxLineNumWidth(endLine)
	maxLineLength := calculateMaxLineLength(snippet.Lines, startLine, endLine)

	// 2. Write header
	padding := strings.Repeat(" ", maxLineNumWidth+1)
	result.WriteString(lineStyle.Sprintf("%s|\n", padding))

	// 3. Write code snippet
	for i := startLine; i <= endLine; i++ {
		line := expandTabs(snippet.Lines[i-1])
		lineNum := fmt.Sprintf("%*d", maxLineNumWidth, i)
		result.WriteString(lineStyle.Sprintf("%s | %s\n", lineNum, line))
	}

	// 4. Write underline and message
	result.WriteString(lineStyle.Sprintf("%s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n", strings.Repeat("~", maxLineLength)))
	result.WriteString(lineStyle.Sprintf("%s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n\n", issue.Message))

	// 5. Write suggestion
	code := strings.Join(snippet.Lines, "\n")
	problemSnippet := internal.ExtractSnippet(issue, code, startLine-1, endLine-1)
	suggestion, err := internal.RemoveUnnecessaryElse(problemSnippet)
	if err != nil {
		suggestion = problemSnippet
	}

	result.WriteString(suggestionStyle.Sprint("Suggestion:\n"))
	result.WriteString(lineStyle.Sprintf("%s|\n", padding))
	suggestionLines := strings.Split(suggestion, "\n")
	for i, line := range suggestionLines {
		lineNum := fmt.Sprintf("%*d", maxLineNumWidth, startLine+i)
		result.WriteString(lineStyle.Sprintf("%s | %s\n", lineNum, line))
	}
	result.WriteString(lineStyle.Sprintf("%s|", padding))
	result.WriteString("\n")

	// 6. Write note
	result.WriteString(suggestionStyle.Sprint("Note: "))
	result.WriteString("Unnecessary 'else' block removed.\n")
	result.WriteString("The code inside the 'else' block has been moved outside, as it will only be executed when the 'if' condition is false.\n\n")

	return result.String()
}
