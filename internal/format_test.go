package internal

import (
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatIssuesWithArrows(t *testing.T) {
	sourceCode := &SourceCode{
		Lines: []string{
			"package main",
			"",
			"func main() {",
			"    x := 1",
			"    if true {}",
			"}",
		},
	}

	issues := []Issue{
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
	sourceCodeWithTabs := &SourceCode{
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
	sourceCode := &SourceCode{
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

	issues := []Issue{
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
10 |      println("end")
   |      ^ example issue

`

	result := FormatIssuesWithArrows(issues, sourceCode)

	assert.Equal(t, expected, result, "Formatted output with multiple digit line numbers does not match expected")
}
