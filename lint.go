package lint

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// Issue represents a lint issue found in the code base.
type Issue struct {
	Rule     string
	Filename string
	Line     int
	Column   int
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
				pos := fset.Position(node.Pos())
				issues = append(issues, Issue{
					Rule:     name,
					Filename: pos.Filename,
					Line:     pos.Line,
					Column:   pos.Column,
					Message:  message,
				})
			}
		}
		return true
	})

	return issues
}

// FormatIssues formats the issues into a string
func FormatIssues(issues []Issue) string {
	var builder strings.Builder
	for _, issue := range issues {
		builder.WriteString(issue.Filename)
		builder.WriteString(":")
		builder.WriteString(strconv.Itoa(issue.Line))
		builder.WriteString(":")
		builder.WriteString(strconv.Itoa(issue.Column))
		builder.WriteString(": ")
		builder.WriteString(issue.Rule)
		builder.WriteString(": ")
		builder.WriteString(issue.Message)
		builder.WriteString("\n")
	}

	return builder.String()
}
