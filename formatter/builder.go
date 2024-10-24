package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"unicode"

	"github.com/fatih/color"
	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
)

const tabWidth = 8

// rule set
const (
	CycloComplexity   = "high-cyclomatic-complexity"
	SliceBound        = "slice-bounds-check"
	MissingModPackage = "gno-mod-tidy"
	DeprecatedFunc    = "deprecated"
)

var (
	errorStyle      = color.New(color.FgRed, color.Bold)
	warningStyle    = color.New(color.FgHiYellow, color.Bold)
	ruleStyle       = color.New(color.FgYellow, color.Bold)
	fileStyle       = color.New(color.FgCyan, color.Bold)
	lineStyle       = color.New(color.FgHiBlue, color.Bold)
	messageStyle    = color.New(color.FgRed, color.Bold)
	suggestionStyle = color.New(color.FgGreen, color.Bold)
	noStyle         = color.New(color.FgWhite)
)

// issueFormatter is the interface that wraps the issueTemplate method.
// Implementations of this interface are responsible for formatting specific types of lint issues.
type issueFormatter interface {
	IssueTemplate() string
}

// getIssueFormatter is a factory function that returns the appropriate IssueFormatter
// based on the given rule.
// If no specific formatter is found for the given rule, it returns a GeneralIssueFormatter.
func getIssueFormatter(rule string) issueFormatter {
	switch rule {
	case CycloComplexity:
		return &CyclomaticComplexityFormatter{}
	case SliceBound:
		return &SliceBoundsCheckFormatter{}
	case MissingModPackage:
		return &MissingModPackageFormatter{}
	default:
		return &GeneralIssueFormatter{}
	}
}

// GenerateFormattedIssue formats a slice of issues into a human-readable string.
// It uses the appropriate formatter for each issue based on its rule.
func GenerateFormattedIssue(issues []tt.Issue, snippet *internal.SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		formatter := getIssueFormatter(issue.Rule)
		formattedIssue := buildIssue(issue, snippet, formatter)
		builder.WriteString(formattedIssue)
	}
	return builder.String()
}

/***** Issue Formatter Builder *****/

type IssueData struct {
	Category        string
	Severity        string
	Rule            string
	Filename        string
	Padding         string
	StartLine       int
	StartColumn     int
	EndLine         int
	EndColumn       int
	MaxLineNumWidth int
	Message         string
	Suggestion      string
	Note            string
	SnippetLines    []string
	CommonIndent    string
}

func buildIssue(issue tt.Issue, snippet *internal.SourceCode, formatter issueFormatter) string {
	startLine := issue.Start.Line
	endLine := issue.End.Line
	maxLineNumWidth := calculateMaxLineNumWidth(endLine)
	padding := strings.Repeat(" ", maxLineNumWidth+1)

	var commonIndent string
	if startLine-1 < 0 || endLine > len(snippet.Lines) || startLine > endLine {
		commonIndent = ""
	} else {
		commonIndent = findCommonIndent(snippet.Lines[startLine-1 : endLine])
	}

	data := IssueData{
		Severity:        issue.Severity.String(),
		Category:        issue.Category,
		Rule:            issue.Rule,
		Filename:        issue.Filename,
		StartLine:       issue.Start.Line,
		StartColumn:     issue.Start.Column,
		EndLine:         issue.End.Line,
		EndColumn:       issue.End.Column,
		Message:         issue.Message,
		Suggestion:      issue.Suggestion,
		Note:            issue.Note,
		MaxLineNumWidth: maxLineNumWidth,
		Padding:         padding,
		CommonIndent:    commonIndent,
		SnippetLines:    snippet.Lines,
	}

	funcMap := template.FuncMap{
		"header":              header,
		"suggestion":          suggestion,
		"note":                note,
		"snippet":             codeSnippet,
		"underlineAndMessage": underlineAndMessage,
		"message":             message,
		"warning":             warning,
		"complexityInfo":      complexityInfo,
	}

	issueTemplate := formatter.IssueTemplate()
	tmpl := template.Must(template.New("issue").Funcs(funcMap).Parse(issueTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error formatting issue: %v", err)
	}
	return buf.String()
}

// utils functions used in the text templates

func header(rule string, severity string, maxLineNumWidth int, filename string, startLine int, startColumn int) string {
	var endString string
	switch severity {
	case "ERROR":
		endString = errorStyle.Sprintf("error: ")
	case "WARNING":
		endString = warningStyle.Sprintf("warning: ")
	case "INFO":
		endString = messageStyle.Sprintf("info: ")
	}

	endString += ruleStyle.Sprintf("%s\n", rule)

	padding := strings.Repeat(" ", maxLineNumWidth)
	endString += lineStyle.Sprintf("%s--> ", padding)
	endString += fileStyle.Sprintf("%s:%d:%d", filename, startLine, startColumn)

	return endString
}

func codeSnippet(snippetLines []string, startLine int, endLine int, maxLineNumWidth int, commonIndent string, padding string) string {
	var endString string
	endString = lineStyle.Sprintf("%s|\n", padding)

	for i := startLine; i <= endLine; i++ {
		if i-1 < 0 || i-1 >= len(snippetLines) {
			continue
		}

		line := snippetLines[i-1]
		line = strings.TrimPrefix(line, commonIndent)
		lineNum := fmt.Sprintf("%*d", maxLineNumWidth, i)

		endString += lineStyle.Sprintf("%s | %s", lineNum, line)
	}

	return endString
}

func underlineAndMessage(message string, padding string, startLine int, endLine int, startColumn int, endColumn int, snippetLines []string, commonIndent string) string {
	var endString string
	endString = lineStyle.Sprintf("%s| ", padding)

	if !isValidLineRange(startLine, endLine, snippetLines) {
		endString += messageStyle.Sprintf("%s\n", message)
		return endString
	}

	commonIndentWidth := calculateVisualColumn(commonIndent, len(commonIndent)+1)

	// calculate underline start position
	underlineStart := calculateVisualColumn(snippetLines[startLine-1], startColumn) - commonIndentWidth
	if underlineStart < 0 {
		underlineStart = 0
	}

	// calculate underline end position
	underlineEnd := calculateVisualColumn(snippetLines[endLine-1], endColumn) - commonIndentWidth
	underlineLength := underlineEnd - underlineStart + 1

	endString += fmt.Sprint(strings.Repeat(" ", underlineStart))
	endString += messageStyle.Sprintf("%s\n", strings.Repeat("~", underlineLength))

	endString += lineStyle.Sprintf("%s= ", padding)
	endString += messageStyle.Sprintf("%s\n", message)

	return endString
}

func suggestion(suggestion string, padding string, maxLineNumWidth int, startLine int) string {
	if suggestion == "" {
		return ""
	}

	var endString string
	endString = suggestionStyle.Sprintf("Suggestion:\n")
	endString += lineStyle.Sprintf("%s|\n", padding)

	suggestionLines := strings.Split(suggestion, "\n")
	for i, line := range suggestionLines {
		lineNum := fmt.Sprintf("%*d", maxLineNumWidth, startLine+i)
		endString += lineStyle.Sprintf("%s | %s\n", lineNum, line)
	}

	endString += lineStyle.Sprintf("%s|\n", padding)
	return endString
}

func note(note string) string {
	if note == "" {
		return ""
	}

	var endString string
	endString = suggestionStyle.Sprint("Note: ")
	endString += lineStyle.Sprintf("%s\n", note)
	return endString
}

func isValidLineRange(startLine int, endLine int, snippetLines []string) bool {
	return startLine > 0 &&
		endLine > 0 &&
		startLine <= endLine &&
		startLine <= len(snippetLines) &&
		endLine <= len(snippetLines)
}

func calculateMaxLineNumWidth(endLine int) int {
	return len(fmt.Sprintf("%d", endLine))
}

// calculateVisualColumn calculates the visual column position
// in a string. taking into account tab characters.
func calculateVisualColumn(line string, column int) int {
	if column < 0 {
		return 0
	}
	visualColumn := 0
	for i, ch := range line {
		if i+1 == column {
			break
		}
		if ch == '\t' {
			visualColumn += tabWidth - (visualColumn % tabWidth)
		} else {
			visualColumn++
		}
	}
	return visualColumn
}

// findCommonIndent finds the common indent in the code snippet.
func findCommonIndent(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	// find first non-empty line's indent
	firstIndent := make([]rune, 0)
	for _, line := range lines {
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		if trimmed != "" {
			firstIndent = []rune(line[:len(line)-len(trimmed)])
			break
		}
	}

	if len(firstIndent) == 0 {
		return ""
	}

	// search common indent for all non-empty lines
	for _, line := range lines {
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		if trimmed == "" {
			continue
		}

		currentIndent := []rune(line[:len(line)-len(trimmed)])
		firstIndent = commonPrefix(firstIndent, currentIndent)

		if len(firstIndent) == 0 {
			break
		}
	}

	return string(firstIndent)
}

// commonPrefix finds the common prefix of two strings.
func commonPrefix(a, b []rune) []rune {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:minLen]
}
