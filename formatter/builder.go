package formatter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/fatih/color"
	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
)

// rule set
const (
	EarlyReturn         = "early-return"
	UnnecessaryTypeConv = "unnecessary-type-conversion"
	SimplifySliceExpr   = "simplify-slice-range"
	CycloComplexity     = "high-cyclomatic-complexity"
	EmitFormat          = "emit-format"
	SliceBound          = "slice-bounds-check"
	Defers              = "defer-issues"
	MissingModPackage   = "gno-mod-tidy"
	DeprecatedFunc      = "deprecated"
)

const tabWidth = 8

var (
	errorStyle      = color.New(color.FgRed, color.Bold)
	warningStyle    = color.New(color.FgHiYellow, color.Bold)
	ruleStyle       = color.New(color.FgYellow, color.Bold)
	fileStyle       = color.New(color.FgCyan, color.Bold)
	lineStyle       = color.New(color.FgBlue, color.Bold)
	messageStyle    = color.New(color.FgRed, color.Bold)
	suggestionStyle = color.New(color.FgGreen, color.Bold)
)

// IssueFormatter is the interface that wraps the Format method.
// Implementations of this interface are responsible for formatting specific types of lint issues.
//
// ! TODO: Use template to format issue
type IssueFormatter interface {
	Format(issue tt.Issue, snippet *internal.SourceCode) string
}

// GenerateFormattedIssue formats a slice of issues into a human-readable string.
// It uses the appropriate formatter for each issue based on its rule.
func GenerateFormattedIssue(issues []tt.Issue, snippet *internal.SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		// builder.WriteString(formatIssueHeader(issue))
		formatter := getFormatter(issue.Rule)
		builder.WriteString(formatter.Format(issue, snippet))
	}
	return builder.String()
}

// getFormatter is a factory function that returns the appropriate IssueFormatter
// based on the given rule.
// If no specific formatter is found for the given rule, it returns a GeneralIssueFormatter.
func getFormatter(rule string) IssueFormatter {
	switch rule {
	case DeprecatedFunc:
		return &DeprecatedFuncFormatter{}
	case EarlyReturn:
		return &EarlyReturnOpportunityFormatter{}
	case SimplifySliceExpr:
		return &SimplifySliceExpressionFormatter{}
	case UnnecessaryTypeConv:
		return &UnnecessaryTypeConversionFormatter{}
	case CycloComplexity:
		return &CyclomaticComplexityFormatter{}
	case EmitFormat:
		return &EmitFormatFormatter{}
	case SliceBound:
		return &SliceBoundsCheckFormatter{}
	case Defers:
		return &DefersFormatter{}
	case MissingModPackage:
		return &MissingModPackageFormatter{}
	default:
		return &GeneralIssueFormatter{}
	}
}

/***** Issue Formatter Builder *****/

type IssueFormatterBuilder struct {
	snippet         *internal.SourceCode
	padding         string
	commonIndent    string
	result          strings.Builder
	issue           tt.Issue
	startLine       int
	endLine         int
	maxLineNumWidth int
}

func NewIssueFormatterBuilder(issue tt.Issue, snippet *internal.SourceCode) *IssueFormatterBuilder {
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

	return &IssueFormatterBuilder{
		issue:           issue,
		snippet:         snippet,
		startLine:       startLine,
		endLine:         endLine,
		maxLineNumWidth: maxLineNumWidth,
		padding:         padding,
		commonIndent:    commonIndent,
	}
}

// headerType represents the type of header to be added to the formatted issue.
// The header can be either a warning or an error.
type headerType int

const (
	warningHeader headerType = iota
	errorHeader
)

func (b *IssueFormatterBuilder) AddHeader(kind headerType) *IssueFormatterBuilder {
	// add header type and rule name
	switch kind {
	case errorHeader:
		b.result.WriteString(errorStyle.Sprint("error: "))
	case warningHeader:
		b.result.WriteString(warningStyle.Sprint("warning: "))
	}

	b.result.WriteString(ruleStyle.Sprintln(b.issue.Rule))

	// add file name
	padding := strings.Repeat(" ", b.maxLineNumWidth)
	b.result.WriteString(lineStyle.Sprint(fmt.Sprintf("%s--> ", padding)))
	b.result.WriteString(fileStyle.Sprintf("%s:%d:%d\n", b.issue.Filename, b.issue.Start.Line, b.issue.Start.Column))

	return b
}

func (b *IssueFormatterBuilder) AddCodeSnippet() *IssueFormatterBuilder {
	// add separator
	b.result.WriteString(lineStyle.Sprintf("%s|\n", b.padding))

	for i := b.startLine; i <= b.endLine; i++ {
		if i-1 < 0 || i-1 >= len(b.snippet.Lines) {
			continue
		}

		line := b.snippet.Lines[i-1]
		line = strings.TrimPrefix(line, b.commonIndent)
		lineNum := fmt.Sprintf("%*d", b.maxLineNumWidth, i)

		b.result.WriteString(lineStyle.Sprintf("%s | ", lineNum))
		b.result.WriteString(line + "\n")
	}

	return b
}

func (b *IssueFormatterBuilder) AddUnderlineAndMessage() *IssueFormatterBuilder {
	b.result.WriteString(lineStyle.Sprintf("%s| ", b.padding))

	if !b.isValidLineRange() {
		b.result.WriteString(messageStyle.Sprintf("%s\n\n", b.issue.Message))
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
	b.result.WriteString(messageStyle.Sprintf("%s\n", strings.Repeat("~", underlineLength)))

	b.result.WriteString(lineStyle.Sprintf("%s= ", b.padding))
	b.result.WriteString(messageStyle.Sprintf("%s\n\n", b.issue.Message))

	return b
}

func (b *IssueFormatterBuilder) AddMessage() *IssueFormatterBuilder {
	b.result.WriteString(messageStyle.Sprint(b.issue.Message))
	b.result.WriteString("\n\n")

	return b
}

func (b *IssueFormatterBuilder) AddSuggestion() *IssueFormatterBuilder {
	if b.issue.Suggestion == "" {
		return b
	}

	b.result.WriteString(suggestionStyle.Sprint("Suggestion:\n"))
	b.result.WriteString(lineStyle.Sprintf("%s|\n", b.padding))

	suggestionLines := strings.Split(b.issue.Suggestion, "\n")
	for i, line := range suggestionLines {
		lineNum := fmt.Sprintf("%*d", b.maxLineNumWidth, b.issue.Start.Line+i)
		b.result.WriteString(lineStyle.Sprintf("%s | ", lineNum))
		b.result.WriteString(line + "\n")
	}

	b.result.WriteString(lineStyle.Sprintf("%s|\n", b.padding))
	b.result.WriteString("\n")

	return b
}

func (b *IssueFormatterBuilder) AddNote() *IssueFormatterBuilder {
	if b.issue.Note == "" {
		return b
	}

	b.result.WriteString(suggestionStyle.Sprint("Note: "))
	b.result.WriteString(b.issue.Note)
	b.result.WriteString("\n\n")

	return b
}

type BaseFormatter struct{}

func (b *IssueFormatterBuilder) Build() string {
	return b.result.String()
}

func (b *IssueFormatterBuilder) isValidLineRange() bool {
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
	var firstIndent []rune
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
