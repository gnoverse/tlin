package formatter

import (
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gnoswap-labs/lint/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatIssuesWithArrows(t *testing.T) {
	sourceCode := &internal.SourceCode{
		Lines: []string{
			"package main",
			"",
			"func main() {",
			"    x := 1",
			"    if true {}",
			"}",
		},
	}

	issues := []internal.Issue{
		{
			Rule:     "unused-variable",
			Filename: "test.go",
			Start:    token.Position{Line: 4, Column: 5},
			End:      token.Position{Line: 4, Column: 6},
			Message:  "x declared but not used",
		},
		{
			Rule:     "empty-if",
			Filename: "test.go",
			Start:    token.Position{Line: 5, Column: 5},
			End:      token.Position{Line: 5, Column: 13},
			Message:  "empty branch",
		},
	}

	expected := `error: unused-variable
 --> test.go
  |
4 |     x := 1
  |     ^ x declared but not used

error: empty-if
 --> test.go
  |
5 |     if true {}
  |     ^ empty branch

`

	result := FormatIssuesWithArrows(issues, sourceCode)

	assert.Equal(t, expected, result, "Formatted output does not match expected")

	// Test with tab characters
	sourceCodeWithTabs := &internal.SourceCode{
		Lines: []string{
			"package main",
			"",
			"func main() {",
			"    x := 1",
			"    if true {}",
			"}",
		},
	}

	expectedWithTabs := `error: unused-variable
 --> test.go
  |
4 |     x := 1
  |     ^ x declared but not used

error: empty-if
 --> test.go
  |
5 |     if true {}
  |     ^ empty branch

`

	resultWithTabs := FormatIssuesWithArrows(issues, sourceCodeWithTabs)

	assert.Equal(t, expectedWithTabs, resultWithTabs, "Formatted output with tabs does not match expected")
}

func TestFormatIssuesWithArrows_MultipleDigitsLineNumbers(t *testing.T) {
	sourceCode := &internal.SourceCode{
		Lines: []string{
			"package main",
			"",
			"func main() {",
			"    x := 1  // unused variable",
			"    if true {}  // empty if statement",
			"    println(\"hello\")",
			"    println(\"world\")",
			"    println(\"test\")",
			"    println(\"example\")",
			"    println(\"end\")",
		},
	}

	issues := []internal.Issue{
		{
			Rule:     "unused-variable",
			Filename: "test.go",
			Start:    token.Position{Line: 4, Column: 5},
			End:      token.Position{Line: 4, Column: 6},
			Message:  "x declared but not used",
		},
		{
			Rule:     "empty-if",
			Filename: "test.go",
			Start:    token.Position{Line: 5, Column: 5},
			End:      token.Position{Line: 5, Column: 13},
			Message:  "empty branch",
		},
		{
			Rule:     "example",
			Filename: "test.go",
			Start:    token.Position{Line: 10, Column: 5},
			End:      token.Position{Line: 10, Column: 12},
			Message:  "example issue",
		},
	}

	expected := `error: unused-variable
 --> test.go
  |
4 |     x := 1  // unused variable
  |     ^ x declared but not used

error: empty-if
 --> test.go
  |
5 |     if true {}  // empty if statement
  |     ^ empty branch

error: example
 --> test.go
   |
10 |     println("end")
   |     ^ example issue

`

	result := FormatIssuesWithArrows(issues, sourceCode)

	assert.Equal(t, expected, result, "Formatted output with multiple digit line numbers does not match expected")
}

func TestFormatIssuesWithArrows_UnnecessaryElse(t *testing.T) {
	sourceCode := &internal.SourceCode{
		Lines: []string{
			"package main",
			"",
			"func unnecessaryElse() bool {",
			"    if condition {",
			"        return true",
			"    } else {",
			"        return false",
			"    }",
			"}",
		},
	}

	issues := []internal.Issue{
		{
			Rule:     "unnecessary-else",
			Filename: "test.go",
			Start:    token.Position{Line: 6, Column: 5},
			End:      token.Position{Line: 8, Column: 5},
			Message:  "unnecessary else block",
		},
	}

	expected := `error: unnecessary-else
 --> test.go
  |
4 |     if condition {
5 |         return true
6 |     } else {
7 |         return false
8 |     }
  | ~~~~~~~~~~~~~~~~~~~~
  | unnecessary else block

Suggestion:
4 | if condition {
5 | 	return true
6 | }
7 | return false

Note: Unnecessary 'else' block removed.
The code inside the 'else' block has been moved outside, as it will only be executed when the 'if' condition is false.

`

	result := FormatIssuesWithArrows(issues, sourceCode)
	assert.Equal(t, expected, result, "Formatted output does not match expected for unnecessary else")
}

func TestIntegratedLintEngine(t *testing.T) {
	t.Skip("skipping integrated lint engine test")
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name: "Detect unused issues",
			code: `
package main

import (
    "fmt"
)

func main() {
    x := 1
    fmt.Println("Hello")
}
`,
			expected: []string{
				"x declared and not used",
			},
		},
		{
			name: "Detect multiple issues",
			code: `
package main

import (
    "fmt"
    "strings"
)

func main() {
    x := 1
    y := "unused"
    fmt.Println("Hello")
}
`,
			expected: []string{
				"x declared and not used",
				"y declared and not used",
				`"strings" imported and not used`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			rootDir := "."
			engine, err := internal.NewEngine(rootDir)
			if err != nil {
				t.Fatalf("unexpected error initializing lint engine: %v", err)
			}

			issues, err := engine.Run(tmpfile)
			require.NoError(t, err)

			assert.Equal(t, len(tt.expected), len(issues), "Number of issues doesn't match")

			for _, exp := range tt.expected {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue.Message, exp) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected issue not found: "+exp)
			}

			if len(issues) > 0 {
				sourceCode, err := internal.ReadSourceCode(tmpfile)
				require.NoError(t, err)
				formattedIssues := FormatIssuesWithArrows(issues, sourceCode)
				t.Logf("Found issues with arrows:\n%s", formattedIssues)
			}
		})
	}
}

func TestUnnecessaryTypeConversionFormatter(t *testing.T) {
	formatter := &UnnecessaryTypeConversionFormatter{}
	
	issue := internal.Issue{
		Rule:       "unnecessary-type-conversion",
		Filename:   "test.go",
		Start:      token.Position{Line: 5, Column: 10},
		End:        token.Position{Line: 5, Column: 20},
		Message:    "unnecessary type conversion",
		Suggestion: "Remove the type conversion. Change `int(myInt)` to just `myInt`.",
		Note:       "Unnecessary type conversions can make the code less readable and may slightly impact performance. They are safe to remove when the expression already has the desired type.",
	}

	snippet := &internal.SourceCode{
		Lines: []string{
			"package main",
			"",
			"func main() {",
			"    myInt := 42",
			"    result := int(myInt)",
			"}",
		},
	}

	expected := `  |
5 |     result := int(myInt)
  |          ^ unnecessary type conversion

Suggestion:
5 | Remove the type conversion. Change ` + "`int(myInt)`" + ` to just ` + "`myInt`" + `.

Note: Unnecessary type conversions can make the code less readable and may slightly impact performance. They are safe to remove when the expression already has the desired type.

`

	result := formatter.Format(issue, snippet)

	assert.Equal(t, expected, result, "Formatted output should match expected output")
}
