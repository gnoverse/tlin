package formatter

import (
	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
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
