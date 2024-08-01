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

	// 1. Calculate dimensions
	startLine := issue.Start.Line
	endLine := issue.Start.Line // Only one line for function declaration
	maxLineNumWidth := calculateMaxLineNumWidth(endLine)

	// 2. Write header
	padding := strings.Repeat(" ", maxLineNumWidth+1)
	result.WriteString(lineStyle.Sprintf("%s|\n", padding))

	// 3. Write code snippet (function declaration)
	line := expandTabs(snippet.Lines[startLine-1])
	lineNum := fmt.Sprintf("%*d", maxLineNumWidth, startLine)
	result.WriteString(lineStyle.Sprintf("%s | %s\n", lineNum, line))

	// 4. Write complexity info
	complexityInfo := fmt.Sprintf("Cyclomatic Complexity: %s", strings.TrimPrefix(issue.Message, "function "))
	result.WriteString(lineStyle.Sprintf("%s| ", padding))
	result.WriteString(messageStyle.Sprintf("%s\n\n", complexityInfo))

	// 5. Write suggestion (if available)
	if issue.Suggestion != "" {
		result.WriteString(suggestionStyle.Sprint("Suggestion: "))
		result.WriteString(issue.Suggestion + "\n\n")
	}

	// 6. Write note (if available)
	if issue.Note != "" {
		result.WriteString(suggestionStyle.Sprint("Note: "))
		result.WriteString(issue.Note + "\n\n")
	}

	return result.String()
}
