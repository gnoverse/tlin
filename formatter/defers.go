package formatter

import (
	"github.com/gnoswap-labs/tlin/internal"
	tt "github.com/gnoswap-labs/tlin/internal/types"
)

type DefersFormatter struct{}

func (f *DefersFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader(errorHeader).
		AddCodeSnippet().
		AddUnderlineAndMessage().
		AddNote().
		Build()
}
