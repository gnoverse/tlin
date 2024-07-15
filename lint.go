package lint

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strings"
)

// SourceCode stores the content of a source code file.
type SourceCode struct {
	Lines []string
}

// ReadSourceFile reads the content of a file and returns it as a `SourceCode` struct.
func ReadSourceCode(filename string) (*SourceCode, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	return &SourceCode{Lines: lines}, nil
}

// Issue represents a lint issue found in the code base.
type Issue struct {
	Rule     string
	Filename string
	Start    token.Position
	End      token.Position
	Message  string
}

// Rule is the interface that wraps the basic Check method.
//
// Check examines an AST node and returns true if the rule is violated,
// along with a message describing the issue.
type Rule interface {
	Check(fset *token.FileSet, node ast.Node) (bool, string)
}

// Engine manages a set of lint rules and runs them on AST nodes.
type Engine struct {
	rules map[string]Rule
}

// NewEngine manages a new lint engine with an empty set of rules.
func NewEngine() *Engine {
	return &Engine{
		rules: make(map[string]Rule),
	}
}

// AddRule adds a new rule to the engine with the given name.
func (e *Engine) AddRule(name string, rule Rule) {
	e.rules[name] = rule
}

// Run applies all the rules to the given AST file and returns a slice of issues.
func (e *Engine) Run(fset *token.FileSet, f *ast.File) []Issue {
	var issues []Issue

	ast.Inspect(f, func(node ast.Node) bool {
		for name, rule := range e.rules {
			if ok, message := rule.Check(fset, node); ok {
				start := fset.Position(node.Pos())
				end := fset.Position(node.End())
				issues = append(issues, Issue{
					Rule:     name,
					Filename: start.Filename,
					Start:    start,
					End:      end,
					Message:  message,
				})
			}
		}
		return true
	})

	return issues
}

func FormatIssuesWithArrows(issues []Issue, sourceCode *SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		// Write issue location and message
		builder.WriteString(fmt.Sprintf("%s:%d:%d: %s: %s\n",
			issue.Filename, issue.Start.Line, issue.Start.Column, issue.Rule, issue.Message))

		// Write the problematic line
		line := sourceCode.Lines[issue.Start.Line-1]
		builder.WriteString(line + "\n")

		// Calculate the visual column, considering tabs
		visualStartColumn := calculateVisualColumn(line, issue.Start.Column)
		visualEndColumn := calculateVisualColumn(line, issue.End.Column)

		// Write the arrow pointing to the issue
		builder.WriteString(strings.Repeat(" ", visualStartColumn-1))
		arrowLength := visualEndColumn - visualStartColumn
		if arrowLength < 1 {
			arrowLength = 1
		}
		builder.WriteString(strings.Repeat("^", arrowLength))
		builder.WriteString("\n")
	}
	return builder.String()
}

func calculateVisualColumn(line string, column int) int {
	visualColumn := 0
	for i, ch := range line {
		if i+1 >= column {
			break
		}
		if ch == '\t' {
			visualColumn += 8 - (visualColumn % 8)
		} else {
			visualColumn++
		}
	}
	return visualColumn
}
