package fixer

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const confidenceThreshold = 0.8

func TestAutoFixer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		issues   []tt.Issue
		dryRun   bool
	}{
		{
			name: "Fix - Simple case",
			input: `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 24},
					Suggestion: "_ = slice[:]",
				},
			},
			expected: `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:]
}
`,
		},
		{
			name: "Fix - Multiple issues",
			input: `package main

func main() {
	slice1 := []int{1, 2, 3}
	_ = slice1[:len(slice1)]

	slice2 := []string{"a", "b", "c"}
	_ = slice2[:len(slice2)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 26},
					Suggestion: "_ = slice1[:]",
				},
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 8, Column: 5},
					End:        token.Position{Line: 8, Column: 26},
					Suggestion: "_ = slice2[:]",
				},
			},
			expected: `package main

func main() {
	slice1 := []int{1, 2, 3}
	_ = slice1[:]

	slice2 := []string{"a", "b", "c"}
	_ = slice2[:]
}
`,
		},
		{
			name: "Fix - Preserve indentation",
			input: `package main

func main() {
	if true {
		slice := []int{1, 2, 3}
		_ = slice[:len(slice)]
	}
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 6, Column: 3},
					End:        token.Position{Line: 6, Column: 22},
					Suggestion: "_ = slice[:]",
				},
			},
			expected: `package main

func main() {
	if true {
		slice := []int{1, 2, 3}
		_ = slice[:]
	}
}
`,
		},
		{
			name: "DryRun",
			input: `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 24},
					Suggestion: "_ = slice[:]",
				},
			},
			expected: `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`,
			dryRun: true,
		},
		{
			name: "FixIssues - Emit function formatting",
			input: `package main

import "runtime/chain"

func main() {
    newOwner := "Alice"
    oldOwner := "Bob"
    chain.Emit("OwnershipChange",
	"newOwner", newOwner, "oldOwner", oldOwner)
}`,
			issues: []tt.Issue{
				{
					Rule:    "emit-format",
					Message: "Consider formatting chain.Emit call for better readability",
					Start:   token.Position{Line: 8, Column: 5},
					End:     token.Position{Line: 9, Column: 44},
					Suggestion: `chain.Emit(
    "OwnershipChange",
    "newOwner", newOwner,
    "oldOwner", oldOwner,
)`,
				},
			},
			expected: `package main

import "runtime/chain"

func main() {
	newOwner := "Alice"
	oldOwner := "Bob"
	chain.Emit(
		"OwnershipChange",
		"newOwner", newOwner,
		"oldOwner", oldOwner,
	)
}
`,
		},
		{
			name: "Fix - format-without-verb Errorf with return",
			input: `package main

func example() error {
	return ufmt.Errorf("handler error")
}
`,
			issues: []tt.Issue{
				{
					Rule:       "format-without-verb",
					Message:    "format string has no verbs; use errors.New() instead",
					Start:      token.Position{Line: 4, Column: 9},
					End:        token.Position{Line: 4, Column: 38},
					Suggestion: `return errors.New("handler error")`,
				},
			},
			expected: `package main

import "errors"

func example() error {
	return errors.New("handler error")
}
`,
		},
		{
			name: "Fix - format-without-verb Sprintf to literal",
			input: `package main

func main() {
	msg := ufmt.Sprintf("hello world")
}
`,
			issues: []tt.Issue{
				{
					Rule:       "format-without-verb",
					Message:    "format string has no verbs; use a string literal directly",
					Start:      token.Position{Line: 4, Column: 9},
					End:        token.Position{Line: 4, Column: 37},
					Suggestion: `msg := "hello world"`,
				},
			},
			expected: `package main

func main() {
	msg := "hello world"
}
`,
		},
		{
			name: "Fix - format-without-verb Printf to print",
			input: `package main

func main() {
	ufmt.Printf("status ok")
}
`,
			issues: []tt.Issue{
				{
					Rule:       "format-without-verb",
					Message:    "format string has no verbs; use print() instead",
					Start:      token.Position{Line: 4, Column: 2},
					End:        token.Position{Line: 4, Column: 26},
					Suggestion: `print("status ok")`,
				},
			},
			expected: `package main

func main() {
	print("status ok")
}
`,
		},
		{
			name: "Fix - format-without-verb Errorf adds errors import",
			input: `package main

func example() error {
	return ufmt.Errorf("handler error")
}
`,
			issues: []tt.Issue{
				{
					Rule:            "format-without-verb",
					Message:         "format string has no verbs; use errors.New() instead",
					Start:           token.Position{Line: 4, Column: 9},
					End:             token.Position{Line: 4, Column: 38},
					Suggestion:      `return errors.New("handler error")`,
					RequiredImports: []string{"errors"},
				},
			},
			expected: `package main

import "errors"

func example() error {
	return errors.New("handler error")
}
`,
		},
		{
			name: "Fix - does not duplicate existing import",
			input: `package main

import "errors"

func example() error {
	return ufmt.Errorf("handler error")
}
`,
			issues: []tt.Issue{
				{
					Rule:            "format-without-verb",
					Message:         "format string has no verbs; use errors.New() instead",
					Start:           token.Position{Line: 6, Column: 9},
					End:             token.Position{Line: 6, Column: 38},
					Suggestion:      `return errors.New("handler error")`,
					RequiredImports: []string{"errors"},
				},
			},
			expected: `package main

import "errors"

func example() error {
	return errors.New("handler error")
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTestCase(t, tt.input, tt.issues, tt.expected, tt.dryRun)
		})
	}
}

func runTestCase(t *testing.T, input string, issues []tt.Issue, expected string, dryRun bool) {
	t.Helper()
	_, testFile, cleanup := setupTestFile(t, input)
	defer cleanup()

	for i := range issues {
		issues[i].Filename = testFile
	}

	fixer := New(dryRun, confidenceThreshold)
	err := fixer.Fix(testFile, issues)
	require.NoError(t, err)

	content, err := os.ReadFile(testFile)
	require.NoError(t, err)

	assert.Equal(t, expected, string(content))
}

func setupTestFile(t *testing.T, content string) (string, string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "autofixer-test")
	require.NoError(t, err)

	testFile := filepath.Join(tmpDir, "test.go")
	err = os.WriteFile(testFile, []byte(content), 0o644)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, testFile, cleanup
}

func BenchmarkFix(b *testing.B) {
	benchmarks := []struct {
		name   string
		input  string
		issues []tt.Issue
	}{
		{
			name: "Simple case",
			input: `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 24},
					Suggestion: "_ = slice[:]",
				},
			},
		},
		{
			name: "Multiple issues",
			input: `package main

func main() {
	slice1 := []int{1, 2, 3}
	_ = slice1[:len(slice1)]

	slice2 := []string{"a", "b", "c"}
	_ = slice2[:len(slice2)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 26},
					Suggestion: "_ = slice1[:]",
				},
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 8, Column: 5},
					End:        token.Position{Line: 8, Column: 26},
					Suggestion: "_ = slice2[:]",
				},
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			tmpDir, err := os.MkdirTemp("", "autofixer-benchmark")
			require.NoError(b, err)
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(testFile, []byte(bm.input), 0o644)
			require.NoError(b, err)

			for i := range bm.issues {
				bm.issues[i].Filename = testFile
			}

			fixer := New(false, confidenceThreshold)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err := fixer.Fix(testFile, bm.issues)
				require.NoError(b, err)

				// reset the file content for the next iteration
				err = os.WriteFile(testFile, []byte(bm.input), 0o644)
				require.NoError(b, err)
			}
		})
	}
}

func TestApplyIndent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		indent   string
		expected string
	}{
		{
			name:     "Single line with basic indent",
			content:  "line1",
			indent:   "    ",
			expected: "    line1",
		},
		{
			name:     "Multiple lines with basic indent",
			content:  "line1\nline2",
			indent:   "    ",
			expected: "    line1\n    line2",
		},
		{
			name:     "Empty content",
			content:  "",
			indent:   "    ",
			expected: "",
		},
		{
			name:     "Multiple lines with tab indent",
			content:  "line1\nline2",
			indent:   "\t",
			expected: "\tline1\n\tline2",
		},
		{
			name:     "Content with trailing newline",
			content:  "line1\nline2\n",
			indent:   "    ",
			expected: "    line1\n    line2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyIndent(tt.content, tt.indent)
			assert.Equal(t, tt.expected, result)
		})
	}
}
