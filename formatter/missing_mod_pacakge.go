package formatter

import (
	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
)

type MissingModPackageFormatter struct{}

func (f *MissingModPackageFormatter) Format(issue tt.Issue, snippet *internal.SourceCode) string {
	builder := NewIssueFormatterBuilder(issue, snippet)
	return builder.
		AddHeader().
		AddMessage().
		Build()
}
