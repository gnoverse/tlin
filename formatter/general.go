package formatter

type GeneralIssueFormatter struct{}

func (f *GeneralIssueFormatter) IssueTemplate() string {
	return `{{header .Rule .Severity .MaxLineNumWidth .Filename .StartLine .StartColumn -}}
{{snippet .SnippetLines .StartLine .EndLine .MaxLineNumWidth .CommonIndent .Padding -}}
{{underlineAndMessage .Message .Padding .StartLine .EndLine .StartColumn .EndColumn .SnippetLines .CommonIndent}}

{{- if .Suggestion }}
{{suggestion .Suggestion .Padding .MaxLineNumWidth .StartLine}}
{{- end }}

{{- if .Note }}
{{note .Note}}
{{- end }}
`
}
