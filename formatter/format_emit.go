package formatter

import (
	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

type EmitFormatFormatter struct{}

func (f *EmitFormatFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader(errorHeader).
		AddCodeSnippet().
		AddUnderlineAndMessage().
		AddSuggestion().
		AddNote().
		Build()
}
