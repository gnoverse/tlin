package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

type CyclomaticComplexityFormatter struct{}

func (f *CyclomaticComplexityFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	var result strings.Builder

	maxLineNumWidth := len(fmt.Sprintf("%d", len(snippet.Lines)))

	// add vertical line
	vl := fmt.Sprintf("%s |\n", strings.Repeat(" ", maxLineNumWidth))
	result.WriteString(lineStyle.Sprintf(vl))

	// function declaration
	functionDeclaration := snippet.Lines[issue.Start.Line-1]
	result.WriteString(formatLine(issue.Start.Line, maxLineNumWidth, functionDeclaration))

	// print complexity
	complexityInfo := fmt.Sprintf("Cyclomatic Complexity: %s", strings.TrimPrefix(issue.Message, "function "))
	result.WriteString(formatLine(0, maxLineNumWidth, complexityInfo))

	// print suggestion
	result.WriteString("\n")
	result.WriteString(suggestionStyle.Sprint("Suggestion: "))
	result.WriteString(issue.Suggestion)

	// print note
	result.WriteString("\n")
	result.WriteString(suggestionStyle.Sprint("Note: "))
	result.WriteString(issue.Note)

	result.WriteString("\n")

	return result.String()
}

func formatLine(lineNum, maxWidth int, content string) string {
	if lineNum > 0 {
		return lineStyle.Sprintf(fmt.Sprintf("%%%dd | ", maxWidth), lineNum) + content + "\n"
	}
	return lineStyle.Sprintf(fmt.Sprintf("%%%ds | ", maxWidth), "") + messageStyle.Sprint(content) + "\n"
}
