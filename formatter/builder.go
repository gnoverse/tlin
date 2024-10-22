package formatter

import (
	"fmt"
	"strings"
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
	lineStyle       = color.New(color.FgBlue, color.Bold)
	messageStyle    = color.New(color.FgRed, color.Bold)
	suggestionStyle = color.New(color.FgGreen, color.Bold)
)

// issueFormatter is the interface that wraps the Format method.
// Implementations of this interface are responsible for formatting specific types of lint issues.
//
// ! TODO: Use template to format issue
type issueFormatter interface {
	Format(issue tt.Issue, snippet *internal.SourceCode) string
}

// GenerateFormattedIssue formats a slice of issues into a human-readable string.
// It uses the appropriate formatter for each issue based on its rule.
func GenerateFormattedIssue(issues []tt.Issue, snippet *internal.SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		formatter := getFormatter(issue.Rule)
		builder.WriteString(formatter.Format(issue, snippet))
	}
	return builder.String()
}

// getFormatter is a factory function that returns the appropriate IssueFormatter
// based on the given rule.
// If no specific formatter is found for the given rule, it returns a GeneralIssueFormatter.
func getFormatter(rule string) issueFormatter {
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

/***** Issue Formatter Builder *****/

type issueFormatterBuilder struct {
	snippet         *internal.SourceCode
	padding         string
	commonIndent    string
	result          strings.Builder
	issue           tt.Issue
	startLine       int
	endLine         int
	maxLineNumWidth int
}

func newIssueFormatterBuilder(issue tt.Issue, snippet *internal.SourceCode) *issueFormatterBuilder {
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

	return &issueFormatterBuilder{
		issue:           issue,
		snippet:         snippet,
		startLine:       startLine,
		endLine:         endLine,
		maxLineNumWidth: maxLineNumWidth,
		padding:         padding,
		commonIndent:    commonIndent,
	}
}

func (b *issueFormatterBuilder) AddHeader() *issueFormatterBuilder {
	// add header type and rule name
	switch b.issue.Severity {
	case tt.SeverityError:
		b.writeStyledLine(errorStyle, "error: ")
	case tt.SeverityWarning:
		b.writeStyledLine(warningStyle, "warning: ")
	case tt.SeverityInfo:
		b.writeStyledLine(messageStyle, "info: ")
	}

	b.writeStyledLine(ruleStyle, "%s\n", b.issue.Rule)

	// add file name
	padding := strings.Repeat(" ", b.maxLineNumWidth)
	b.writeStyledLine(lineStyle, "%s--> ", padding)
	b.writeStyledLine(fileStyle, "%s:%d:%d\n", b.issue.Filename, b.issue.Start.Line, b.issue.Start.Column)

	return b
}

func (b *issueFormatterBuilder) AddCodeSnippet() *issueFormatterBuilder {
	// add separator
	b.writeStyledLine(lineStyle, "%s|\n", b.padding)

	for i := b.startLine; i <= b.endLine; i++ {
		if i-1 < 0 || i-1 >= len(b.snippet.Lines) {
			continue
		}

		line := b.snippet.Lines[i-1]
		line = strings.TrimPrefix(line, b.commonIndent)
		lineNum := fmt.Sprintf("%*d", b.maxLineNumWidth, i)

		b.writeStyledLine(lineStyle, "%s | %s\n", lineNum, line)
	}

	return b
}

func (b *issueFormatterBuilder) AddUnderlineAndMessage() *issueFormatterBuilder {
	b.writeStyledLine(lineStyle, "%s| ", b.padding)

	if !b.isValidLineRange() {
		b.writeStyledLine(messageStyle, "%s\n\n", b.issue.Message)
		return b
	}

	commonIndentWidth := calculateVisualColumn(b.commonIndent, len(b.commonIndent)+1)

	// calculate underline start position
	underlineStart := calculateVisualColumn(b.snippet.Lines[b.startLine-1], b.issue.Start.Column) - commonIndentWidth
	if underlineStart < 0 {
		underlineStart = 0
	}

	// calculate underline end position
	underlineEnd := calculateVisualColumn(b.snippet.Lines[b.endLine-1], b.issue.End.Column) - commonIndentWidth
	underlineLength := underlineEnd - underlineStart + 1

	b.result.WriteString(strings.Repeat(" ", underlineStart))
	b.writeStyledLine(messageStyle, "%s\n", strings.Repeat("~", underlineLength))

	b.writeStyledLine(lineStyle, "%s= ", b.padding)
	b.writeStyledLine(messageStyle, "%s\n\n", b.issue.Message)

	return b
}

func (b *issueFormatterBuilder) AddMessage() *issueFormatterBuilder {
	b.writeStyledLine(messageStyle, "%s\n\n", b.issue.Message)

	return b
}

func (b *issueFormatterBuilder) AddSuggestion() *issueFormatterBuilder {
	if b.issue.Suggestion == "" {
		return b
	}

	b.writeStyledLine(suggestionStyle, "Suggestion:\n")
	b.writeStyledLine(lineStyle, "%s|\n", b.padding)

	suggestionLines := strings.Split(b.issue.Suggestion, "\n")
	for i, line := range suggestionLines {
		lineNum := fmt.Sprintf("%*d", b.maxLineNumWidth, b.issue.Start.Line+i)
		b.writeStyledLine(lineStyle, "%s | %s\n", lineNum, line)
	}

	b.writeStyledLine(lineStyle, "%s|\n\n", b.padding)

	return b
}

func (b *issueFormatterBuilder) AddNote() *issueFormatterBuilder {
	if b.issue.Note == "" {
		return b
	}

	b.result.WriteString(suggestionStyle.Sprint("Note: "))
	b.writeStyledLine(lineStyle, "%s\n\n", b.issue.Note)

	return b
}

func (b *issueFormatterBuilder) writeStyledLine(style *color.Color, format string, a ...interface{}) {
	b.result.WriteString(style.Sprintf(format, a...))
}

type BaseFormatter struct{}

func (b *issueFormatterBuilder) Build() string {
	return b.result.String()
}

func (b *issueFormatterBuilder) isValidLineRange() bool {
	return b.startLine > 0 &&
		b.endLine > 0 &&
		b.startLine <= b.endLine &&
		b.startLine <= len(b.snippet.Lines) &&
		b.endLine <= len(b.snippet.Lines)
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
