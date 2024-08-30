package formatter

import (
	"github.com/gnoswap-labs/tlin/internal"
	tt "github.com/gnoswap-labs/tlin/internal/types"
)

type SliceBoundsCheckFormatter struct{}

func (f *SliceBoundsCheckFormatter) Format(
	issue tt.Issue,
	snippet *internal.SourceCode,
) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader(errorHeader).
		AddCodeSnippet().
		AddUnderlineAndMessage().
		AddWarning().
		Build()
}

func (b *IssueFormatterBuilder) AddWarning() *IssueFormatterBuilder {
	b.result.WriteString(warningStyle.Sprint("warning: "))
	if b.issue.Category == "index-access" {
		b.result.WriteString("Index access without bounds checking can lead to runtime panics.\n")
	} else if b.issue.Category == "slice-expression" {
		b.result.WriteString("Slice expressions without proper length checks may cause unexpected behavior.\n\n")
	}

	return b
}
