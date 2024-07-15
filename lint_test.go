package lint

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLintRule(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		ruleName string
		expected bool
	}{
		{
			name:     "Detect empty if statement",
			code:     "package main\n\nfunc main() {\n\tif true {}\n}",
			ruleName: "no-empty-if",
			expected: true,
		},
		{
			name:     "No empty if statement",
			code:     "package main\n\nfunc main() {\n\tif true { println(\"Hello\") }\n}",
			ruleName: "no-empty-if",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("failed to parse code: %v", err)
			}

			engine := NewEngine()
			engine.AddRule(tt.ruleName, NoEmptyIfRule{})

			issues := engine.Run(fset, f)

			if tt.expected && len(issues) == 0 {
				t.Errorf("expected to find issues, but found none")
			}
			if !tt.expected && len(issues) > 0 {
				t.Errorf("expected to find no issues, but found %d", len(issues))
			}

			if len(issues) > 0 {
				sourceCode := &SourceCode{Lines: strings.Split(tt.code, "\n")}
				formattedIssues := FormatIssuesWithArrows(issues, sourceCode)
				t.Logf("Found issues with arrows:\n%s", formattedIssues)

				assert.Contains(t, formattedIssues, "no-empty-if: empty if statement")
				assert.Contains(t, formattedIssues, "if true {}")
				assert.Contains(t, formattedIssues, "^^^^^^^^^^")
			}
		})
	}
}
