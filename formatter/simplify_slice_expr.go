package formatter

import (
	"github.com/gnoswap-labs/tlin/internal"
	tt "github.com/gnoswap-labs/tlin/internal/types"
)

type SimplifySliceExpressionFormatter struct{}

func (f *SimplifySliceExpressionFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader(errorHeader).
		AddCodeSnippet().
		AddUnderlineAndMessage().
		AddSuggestion().
		AddNote().
		Build()
}
