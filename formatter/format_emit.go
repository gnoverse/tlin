package formatter

import (
	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
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
