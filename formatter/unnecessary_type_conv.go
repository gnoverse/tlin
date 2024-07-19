package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
)

// UnnecessaryConversionFormatter formats unnecessary conversion issues.
type UnnecessaryConversionFormatter struct{}

func (f *UnnecessaryConversionFormatter) Format(
	issue internal.Issue,
	snippet *internal.SourceCode,
) string {
	var result strings.Builder

	lineNumberStr := fmt.Sprintf("%d", issue.Start.Line)
	padding := strings.Repeat(" ", len(lineNumberStr)-1)
	result.WriteString(lineStyle.Sprintf("  %s|\n", padding))

	// Get the problematic line
	line := snippet.Lines[issue.Start.Line-1]
	
	// Write the line number and content
	result.WriteString(lineStyle.Sprintf("%d | ", issue.Start.Line))
	result.WriteString(line + "\n")

	// Calculate the position for the error indicator
	arrowPos := strings.Repeat(" ", issue.Start.Column-1)
	
	// Write the error indicator
	result.WriteString(lineStyle.Sprintf("  | "))
	result.WriteString(arrowPos)
	result.WriteString(messageStyle.Sprintf("^ unnecessary conversion\n"))

	// Add explanation and suggestion
	result.WriteString("\n")
	result.WriteString(suggestionStyle.Sprint("Suggestion: "))
	result.WriteString("Remove the unnecessary type conversion.\n")
	
	// Try to provide a code suggestion
	suggestion := f.generateSuggestion(line, issue.Start.Column-1, issue.End.Column-1)
	if suggestion != "" {
		result.WriteString("Consider changing the code to:\n")
		result.WriteString(suggestionStyle.Sprintf("%s\n", suggestion))
	}

	result.WriteString("\n")

	return result.String()
}

func (f *UnnecessaryConversionFormatter) generateSuggestion(line string, start, end int) string {
	// This is a simple suggestion generator and might not work for all cases
	// A more robust solution would involve parsing the AST
	before := line[:start]
	conversionPart := line[start:end]
	after := line[end:]

	// Try to remove the type conversion
	parts := strings.SplitN(conversionPart, "(", 2)
	if len(parts) != 2 {
		return ""
	}

	innerPart := strings.TrimRight(parts[1], ")")
	
	return before + innerPart + after
}
