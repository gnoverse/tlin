package formatter

import (
	"github.com/gnoswap-labs/tlin/internal"
	tt "github.com/gnoswap-labs/tlin/internal/types"
)

type EarlyReturnOpportunityFormatter struct{}

func (f *EarlyReturnOpportunityFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader(warningHeader).
		AddCodeSnippet().
		AddUnderlineAndMessage().
		AddSuggestion().
		AddNote().
		Build()
}
