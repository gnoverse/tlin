package formatter

import (
	"fmt"
	"strings"

	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
)

type CyclomaticComplexityFormatter struct{}

func (f *CyclomaticComplexityFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	builder := newIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader().
		AddCodeSnippet().
		AddComplexityInfo().
		AddSuggestion().
		AddNote().
		Build()
}

func (b *issueFormatterBuilder) AddComplexityInfo() *issueFormatterBuilder {
	maxLineNumWidth := calculateMaxLineNumWidth(b.issue.End.Line)
	padding := strings.Repeat(" ", maxLineNumWidth+1)

	complexityInfo := fmt.Sprintf("Cyclomatic Complexity: %s", strings.TrimPrefix(b.issue.Message, "function "))
	b.result.WriteString(lineStyle.Sprintf("%s| ", padding))
	b.result.WriteString(messageStyle.Sprintf("%s\n\n", complexityInfo))

	return b
}
