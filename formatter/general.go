package formatter

import (
	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
)

// GeneralIssueFormatter is a formatter for general lint issues.
type GeneralIssueFormatter struct{}

// Format formats a general lint issue into a human-readable string.
// It takes an Issue and a SourceCode snippet as input and returns a formatted string.
func (f *GeneralIssueFormatter) Format(
	issue tt.Issue,
	snippet *internal.SourceCode,
) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader().
		AddCodeSnippet().
		AddUnderlineAndMessage().
		Build()
}
