package formatter

func init() {
	Register("gno-mod-tidy", &MissingModPackageFormatter{})
}

type MissingModPackageFormatter struct{}

func (f *MissingModPackageFormatter) IssueTemplate() string {
	return `{{header .Rule .Severity .MaxLineNumWidth .Filename .StartLine .StartColumn -}}
{{message .Message}}
`
}

func message(message string) string {
	return messageStyle.Sprintf("%s\n", message)
}
