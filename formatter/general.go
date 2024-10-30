package formatter

type GeneralIssueFormatter struct{}

func (f *GeneralIssueFormatter) IssueTemplate() string {
	return `{{header .Rule .Severity .MaxLineNumWidth .Filename .StartLine .StartColumn -}}
{{snippet .SnippetLines .StartLine .EndLine .MaxLineNumWidth .CommonIndent .Padding -}}
{{underlineAndMessage .Message .Padding .StartLine .EndLine .StartColumn .EndColumn .SnippetLines .CommonIndent .Note}}

{{- if .Note }}
{{note .Note .Padding .Suggestion}}
{{- end }}

{{- if .Suggestion }}
{{suggestion .Suggestion .Padding .MaxLineNumWidth .StartLine}}
{{- end }}
`
}
