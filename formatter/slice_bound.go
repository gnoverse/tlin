package formatter

type SliceBoundsCheckFormatter struct{}

func (f *SliceBoundsCheckFormatter) IssueTemplate() string {
	return `{{header .Rule .Severity .MaxLineNumWidth .Filename .StartLine .StartColumn -}}
{{snippet .SnippetLines .StartLine .EndLine .MaxLineNumWidth .CommonIndent .Padding -}}
{{underlineAndMessage .Message .Padding .StartLine .EndLine .StartColumn .EndColumn .SnippetLines .CommonIndent .Note}}
{{warning .Category -}}
`
}

// TODO: make this as a note
func warning(category string) string {
	var endString string
	endString = warningStyle.Sprint("warning: ")
	if category == "index-access" {
		endString += "Index access without bounds checking can lead to runtime panics.\n\n"
	} else if category == "slice-expression" {
		endString += "Slice expressions without proper length checks may cause unexpected behavior.\n\n"
	}

	return endString
}
