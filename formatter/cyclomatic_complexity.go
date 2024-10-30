package formatter

import (
	"fmt"
	"strings"
)

type CyclomaticComplexityFormatter struct{}

func (f *CyclomaticComplexityFormatter) IssueTemplate() string {
	return `{{header .Rule .Severity .MaxLineNumWidth .Filename .StartLine .StartColumn -}}
{{snippet .SnippetLines .StartLine .EndLine .MaxLineNumWidth .CommonIndent .Padding -}}
{{underlineAndMessage .Message .Padding .StartLine .EndLine .StartColumn .EndColumn .SnippetLines .CommonIndent .Note -}}
{{complexityInfo .Padding .Message }}

{{- if .Note }}
{{note .Note .Padding .Suggestion}}
{{- end }}

{{- if .Suggestion }}
{{suggestion .Suggestion .Padding .MaxLineNumWidth .StartLine}}
{{- end }}
`
}

func complexityInfo(padding string, message string) string {
	var endString string
	complexityInfo := fmt.Sprintf("Cyclomatic Complexity: %s", strings.TrimPrefix(message, "function "))
	endString = lineStyle.Sprintf("%s| ", padding)
	endString += messageStyle.Sprintf("%s\n", complexityInfo)

	return endString
}
