package formatter

import (
	"go/token"
	"testing"

	"github.com/gnoswap-labs/tlin/internal"
	tt "github.com/gnoswap-labs/tlin/internal/types"
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
 --> test.go
  |
4 |     x := 1
  |     ~~
  | x declared but not used

error: empty-if
 --> test.go
  |
5 |     if true {}
  |     ~~~~~~~~~
  | empty branch

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
 --> test.go
  |
4 |     x := 1
  |     ~~
  | x declared but not used

error: empty-if
 --> test.go
  |
5 |     if true {}
  |     ~~~~~~~~~
  | empty branch

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
 --> test.go
  |
4 |     x := 1  // unused variable
  |     ~~
  | x declared but not used

error: empty-if
 --> test.go
  |
5 |     if true {}  // empty if statement
  |     ~~~~~~~~~
  | empty branch

error: example
  --> test.go
   |
10 |     println("end")
   |     ~~~~~~~~
   | example issue

`

	result := GenerateFormattedIssue(issues, code)

	assert.Equal(t, expected, result, "Formatted output with multiple digit line numbers does not match expected")
}

func TestFormatIssuesWithArrows_UnnecessaryElse(t *testing.T) {
	t.Skip()
	t.Parallel()
	code := &internal.SourceCode{
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

	issues := []tt.Issue{
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
  |
4 | if condition {
5 | 	return true
6 | }
7 | return false
  |
Note: Unnecessary 'else' block removed.
The code inside the 'else' block has been moved outside, as it will only be executed when the 'if' condition is false.

`

	result := GenerateFormattedIssue(issues, code)
	t.Logf("result: %s", result)
	assert.Equal(t, expected, result, "Formatted output does not match expected for unnecessary else")
}

func TestUnnecessaryTypeConversionFormatter(t *testing.T) {
	t.Parallel()
	formatter := &UnnecessaryTypeConversionFormatter{}

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
 --> test.go
  |
5 |     result := int(myInt)
  |          ~~~~~~~~~~~
  | unnecessary type conversion

Suggestion:
  |
5 | Remove the type conversion. Change ` + "`int(myInt)`" + ` to just ` + "`myInt`" + `.
  |

Note: Unnecessary type conversions can make the code less readable and may slightly impact performance. They are safe to remove when the expression already has the desired type.

`

	result := formatter.Format(issue, snippet)

	assert.Equal(t, expected, result, "Formatted output should match expected output")
}
