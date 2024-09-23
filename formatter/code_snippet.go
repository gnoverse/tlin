package formatter

import (
	"strings"

	"github.com/gnoswap-labs/tlin/internal"
	tt "github.com/gnoswap-labs/tlin/internal/types"
)

func GetCodeSnippet(issue tt.Issue, snippet *internal.SourceCode) string {
	startLine := issue.Start.Line - 1
	endLine := issue.End.Line
	if endLine > len(snippet.Lines) {
		endLine = len(snippet.Lines)
	}
	return strings.Join(snippet.Lines[startLine:endLine], "\n")
}
