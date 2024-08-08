package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

type CyclomaticComplexityFormatter struct{}

func (f *CyclomaticComplexityFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader().
		AddCodeSnippet().
		AddComplexityInfo().
		AddSuggestion().
		AddNote().
		Build()
}

func (b *IssueFormatterBuilder) AddComplexityInfo() *IssueFormatterBuilder {
	maxLineNumWidth := calculateMaxLineNumWidth(b.issue.End.Line)
	padding := strings.Repeat(" ", maxLineNumWidth+1)

	complexityInfo := fmt.Sprintf("Cyclomatic Complexity: %s", strings.TrimPrefix(b.issue.Message, "function "))
	b.result.WriteString(lineStyle.Sprintf("%s| ", padding))
	b.result.WriteString(messageStyle.Sprintf("%s\n\n", complexityInfo))

	return b
}
