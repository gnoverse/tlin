package formatter

import (
	"go/token"
	"testing"

	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestFormatIssuesWithArrows(t *testing.T) {
	t.Parallel()
	code := &internal.SourceCode{
		Lines: []string{
			"package main",
			"",
			"func main() {",
			"    x := 1",
			"    if true {}",
			"}",
		},
	}

	issues := []tt.Issue{
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
 --> test.go:4:5
  |
4 | x := 1
  | ^^
  |
  = x declared but not used

error: empty-if
 --> test.go:5:5
  |
5 | if true {}
  | ^^^^^^^^^
  |
  = empty branch

`

	result := GenerateFormattedIssue(issues, code)

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
 --> test.go:4:5
  |
4 | x := 1
  | ^^
  |
  = x declared but not used

error: empty-if
 --> test.go:5:5
  |
5 | if true {}
  | ^^^^^^^^^
  |
  = empty branch

`

	resultWithTabs := GenerateFormattedIssue(issues, sourceCodeWithTabs)

	assert.Equal(t, expectedWithTabs, resultWithTabs, "Formatted output with tabs does not match expected")
}

func TestFormatIssuesWithArrows_MultipleDigitsLineNumbers(t *testing.T) {
	t.Parallel()
	code := &internal.SourceCode{
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

	issues := []tt.Issue{
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
 --> test.go:4:5
  |
4 | x := 1  // unused variable
  | ^^
  |
  = x declared but not used

error: empty-if
 --> test.go:5:5
  |
5 | if true {}  // empty if statement
  | ^^^^^^^^^
  |
  = empty branch

error: example
  --> test.go:10:5
   |
10 | println("end")
   | ^^^^^^^^
   |
   = example issue

`

	result := GenerateFormattedIssue(issues, code)

	assert.Equal(t, expected, result, "Formatted output with multiple digit line numbers does not match expected")
}

func TestUnnecessaryTypeConversionFormatter(t *testing.T) {
	t.Parallel()

	issue := tt.Issue{
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

	expected := `error: unnecessary-type-conversion
 --> test.go:5:10
  |
5 | result := int(myInt)
  |      ^^^^^^^^^^^
  |
  = unnecessary type conversion
  = note: Unnecessary type conversions can make the code less readable and may slightly impact performance. They are safe to remove when the expression already has the desired type.
suggestion:
  |
5 | Remove the type conversion. Change ` + "`int(myInt)`" + ` to just ` + "`myInt`" + `.
  |

`

	result := GenerateFormattedIssue([]tt.Issue{issue}, snippet)

	assert.Equal(t, expected, result, "Formatted output should match expected output")
}

// TestFormatIssuesWithInconsistentIndentation tests the case where an issue
// spans multiple lines with inconsistent indentation (first line has no indent,
// subsequent lines are indented). This was causing a panic due to negative
// Repeat count in strings.Repeat.
func TestFormatIssuesWithInconsistentIndentation(t *testing.T) {
	t.Parallel()

	// Reproduces the bug: first line has no indentation,
	// but the rest of the lines are indented.
	code := &internal.SourceCode{
		Lines: []string{
			"impl, ok := result.(IMyType)",
			"    if !ok {",
			`        return errors.New("impl is not an IMyType")`,
			"    } else {",
			"        implementation = impl",
			"        return nil",
			"    }",
		},
	}

	// Issue spans from line 2 to line 7, simulating early-return-opportunity detection
	issues := []tt.Issue{
		{
			Rule:     "early-return-opportunity",
			Filename: "test.go",
			Start:    token.Position{Line: 2, Column: 5},
			End:      token.Position{Line: 7, Column: 5},
			Message:  "this if-else block can be simplified using early return",
		},
	}

	// Should not panic
	result := GenerateFormattedIssue(issues, code)

	// Verify the output contains expected elements
	assert.Contains(t, result, "early-return-opportunity")
	assert.Contains(t, result, "test.go:2:5")
	assert.Contains(t, result, "this if-else block can be simplified using early return")
}

func TestFindCommonIndent(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		lines    []string
	}{
		{
			name: "whitespace indent",
			lines: []string{
				"    if foo {",
				"        println()",
				"    }",
			},
			expected: "    ",
		},
		{
			name: "tab indent",
			lines: []string{
				"	if foo {",
				"		println()",
				"	}",
			},
			expected: "\t",
		},
		{
			name: "mixed indent (space and tab)",
			lines: []string{
				"\t    if foo {",
				"\t    \tprintln()",
				"\t    }",
			},
			expected: "\t    ",
		},
		{
			name: "no indent",
			lines: []string{
				"if foo {",
				"println()",
				"}",
			},
			expected: "",
		},
		{
			name: "empty line",
			lines: []string{
				"    if foo {",
				"",
				"        println()",
				"    }",
			},
			expected: "    ",
		},
		{
			name: "various indent levels",
			lines: []string{
				"    if foo {",
				"      bar()",
				"        baz()",
				"    }",
			},
			expected: "    ",
		},
		{
			name:     "empty input",
			lines:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findCommonIndent(tt.lines)
			if result != tt.expected {
				t.Errorf("findCommonIndent() = %q, want %q", result, tt.expected)
			}
		})
	}
}
