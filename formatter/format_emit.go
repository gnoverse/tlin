package formatter

import (
	"fmt"
	"strings"

	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

type EmitFormatFormatter struct{}

func (f *EmitFormatFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader().
		AddCodeSnippet().
		AddUnderlineAndMessage().
		AddEmitFormatSuggestion().
		AddNote().
		Build()
}

func (b *IssueFormatterBuilder) AddEmitFormatSuggestion() *IssueFormatterBuilder {
	if b.issue.Suggestion == "" {
		return b
	}

	maxLineNumWidth := calculateMaxLineNumWidth(b.issue.End.Line)
	padding := strings.Repeat(" ", maxLineNumWidth+1)

	b.result.WriteString(suggestionStyle.Sprint("Suggestion:\n"))
	b.result.WriteString(lineStyle.Sprintf("%s|\n", padding))

	suggestionLines := strings.Split(b.issue.Suggestion, "\n")
	for i, line := range suggestionLines {
		lineNum := fmt.Sprintf("%*d", maxLineNumWidth, b.issue.Start.Line+i)
		b.result.WriteString(lineStyle.Sprintf("%s | %s\n", lineNum, line))
	}

	b.result.WriteString(lineStyle.Sprintf("%s|\n", padding))

	return b
}
