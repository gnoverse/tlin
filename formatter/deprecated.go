package formatter

import (
	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
)

type DeprecatedFuncFormatter struct{}

func (f *DeprecatedFuncFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader().
		AddCodeSnippet().
		AddUnderlineAndMessage().
		AddNote().
		Build()
}
