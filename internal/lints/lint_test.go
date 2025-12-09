package lints

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectUnnecessarySliceLength(t *testing.T) {
	t.Parallel()
	baseMsg := "unnecessary use of len() in slice expression, can be simplified"
	tests := []struct {
		name     string
		code     string
		message  string
		expected int
	}{
		{
			name: "suggests to use slice[:]",
			code: `
package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:len(slice)]
}`,
			expected: 1,
			message:  baseMsg,
		},
		{
			name: "suggests to use slice[a:]",
			code: `
package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[1:len(slice)]
}`,
			expected: 1,
			message:  baseMsg,
		},
		{
			name: "Unnecessary slice length",
			code: `
package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:]
}`,
			expected: 0,
		},
		{
			name: "slice[a:len(slice)] -> slice[a:] (a: variable)",
			code: `
package main

func main() {
	slice := []int{1, 2, 3}
	a := 1
	_ = slice[a:len(slice)]
}`,
			expected: 1,
			message:  baseMsg,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			node, fset, err := ParseFile(tmpfile, nil)
			require.NoError(t, err)

			issues, err := DetectUnnecessarySliceLength(tmpfile, node, fset, types.SeverityError)
			require.NoError(t, err)

			assert.Equal(
				t,
				tt.expected,
				len(issues),
				"Number of detected unnecessary slice length doesn't match expected",
			)

			if len(issues) > 0 {
				for _, issue := range issues {
					assert.Equal(t, "simplify-slice-range", issue.Rule)
					assert.Equal(
						t,
						tt.message,
						issue.Message,
					)
				}
			}
		})
	}
}

func TestDetectUnnecessaryTypeConversion(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "Unnecessary conversion",
			code: `
package main

func example() {
    var x int = 5
    y := int(x)
    _ = y
}`,
			expected: 1,
		},
		{
			name: "Necessary conversion",
			code: `
package main

func example() {
    var x float64 = 5.0
    y := int(x)
    _ = y
}`,
			expected: 0,
		},
		{
			name: "Untyped constant conversion",
			code: `
package main

func example() {
    x := int(5)
    _ = x
}`,
			expected: 0,
		},
		{
			name: "Multiple unnecessary conversions",
			code: `
package main

func example() {
    var x int = 5
    var y int64 = 10
    a := int(x)
    b := int64(y)
    _, _ = a, b
}`,
			expected: 2,
		},
		{
			name: "No conversions",
			code: `
package main

func example() {
    x := 5
    y := 10
    _ = x + y
}`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			node, fset, err := ParseFile(tmpfile, nil)
			require.NoError(t, err)

			issues, err := DetectUnnecessaryConversions(tmpfile, node, fset, types.SeverityError)
			require.NoError(t, err)

			assert.Equal(
				t,
				tt.expected,
				len(issues),
				"Number of detected unnecessary type conversions doesn't match expected",
			)

			if len(issues) > 0 {
				for _, issue := range issues {
					assert.Equal(t, "unnecessary-type-conversion", issue.Rule)
					assert.Equal(t, "unnecessary type conversion", issue.Message)
				}
			}
		})
	}
}

func TestDetectEmitFormat(t *testing.T) {
	t.Parallel()
	_, current, _, ok := runtime.Caller(0)
	require.True(t, ok)
	testDir := filepath.Join(filepath.Dir(current), "..", "..", "testdata", "emit")

	tests := []struct {
		name     string
		filename string
		expected int
	}{
		{
			name:     "Emit with 3 arguments",
			filename: "emit0.gno",
			expected: 0,
		},
		{
			name:     "Emit with more than 3 arguments",
			filename: "emit1.gno",
			expected: 1,
		},
		{
			name:     "Emit with new line",
			filename: "emit2.gno",
			expected: 0,
		},
		{
			name:     "Emit with inconsistent new line",
			filename: "emit3.gno",
			expected: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(testDir, tt.filename)
			content, err := os.ReadFile(path)
			require.NoError(t, err)

			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, content, 0o644)
			require.NoError(t, err)

			node, fset, err := ParseFile(tmpfile, nil)
			require.NoError(t, err)

			issues, err := DetectEmitFormat(tmpfile, node, fset, types.SeverityError)
			require.NoError(t, err)

			assert.Equal(
				t,
				tt.expected,
				len(issues),
				fmt.Sprintf("Number of detected issues doesn't match expected for %s. %v", tt.filename, issues),
			)

			if len(issues) > 0 {
				assert.Equal(t, "emit-format", issues[0].Rule)
				assert.Contains(t, issues[0].Message, "consider formatting chain.Emit call for better readability")
			}
		})
	}
}

func TestFormatEmitCall(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "Simple Emit call",
			input: `chain.Emit("OwnershipChange", "newOwner", newOwner.String())`,
			expected: `chain.Emit(
    "OwnershipChange",
    "newOwner", newOwner.String(),
)`,
		},
		{
			name:  "Emit call with multiple key-value pairs",
			input: `chain.Emit("OwnershipChange", "newOwner", newOwner.String(), "oldOwner", oldOwner.String())`,
			expected: `chain.Emit(
    "OwnershipChange",
    "newOwner", newOwner.String(),
    "oldOwner", oldOwner.String(),
)`,
		},
		{
			name:  "Emit call with function calls as values",
			input: `chain.Emit("Transfer", "from", sender.Address(), "to", recipient.Address(), "amount", token.Format(amount))`,
			expected: `chain.Emit(
    "Transfer",
    "from", sender.Address(),
    "to", recipient.Address(),
    "amount", token.Format(amount),
)`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			expr, err := parser.ParseExpr(tt.input)
			assert.NoError(t, err)

			callExpr, ok := expr.(*ast.CallExpr)
			assert.True(t, ok)

			result := formatEmitCall(callExpr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectUselessBreak(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "No useless break",
			code: `
package main

func main() {
	switch x := 1; x {
	case 1:
		println("one")
	case 2:
		println("two")
	default:
		println("other")
	}
}`,
			expected: 0,
		},
		{
			name: "Useless break in switch",
			code: `
package main

func main() {
	switch x := 1; x {
	case 1:
		println("one")
		break
	case 2:
		println("two")
	default:
		println("other")
		break
	}
}`,
			expected: 2,
		},
		{
			name: "Useless break in select",
			code: `
package main

func main() {
	select {
	case <-ch1:
		println("received from ch1")
		break
	case <-ch2:
		println("received from ch2")
	default:
		println("no communication")
		break
	}
}`,
			expected: 2,
		},
		{
			name: "Labeled break (not useless)",
			code: `
package main

func main() {
outer:
	for {
		switch x := 1; x {
		case 1:
			println("one")
			break outer
		case 2:
			println("two")
		}
	}
}`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectUselessBreak("test.go", node, fset, types.SeverityError)
			require.NoError(t, err)

			assert.Equal(
				t,
				tt.expected,
				len(issues),
				"Number of detected useless break statements doesn't match expected",
			)

			if len(issues) > 0 {
				for _, issue := range issues {
					assert.Equal(t, "useless-break", issue.Rule)
					assert.Contains(t, issue.Message, "useless break statement")
				}
			}
		})
	}
}

func TestDetectConstErrorDeclaration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "Constant error declaration",
			code: `
package main

import "errors"

const err = errors.New("error")
`,
			expected: 1,
		},
		{
			name: "Constant error declaration with multiple errors",
			code: `
package main

import "errors"

const (
	err1 = errors.New("error1")
	err2 = errors.New("error2")
)
`,
			expected: 1,
		},
		{
			name: "Variable error declaration",
			code: `
package main

import "errors"

var err = errors.New("error")
`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			node, fset, err := ParseFile(tmpfile, nil)
			require.NoError(t, err)

			issues, err := DetectConstErrorDeclaration(tmpfile, node, fset, types.SeverityError)
			require.NoError(t, err)

			assert.Equal(
				t,
				tt.expected,
				len(issues),
				"Number of detected constant error declarations doesn't match expected",
			)

			for _, issue := range issues {
				assert.Equal(t, "const-error-declaration", issue.Rule)
				assert.Contains(t, issue.Message, "avoid declaring constant errors")
				assert.Contains(t, issue.Suggestion, "var")
			}
		})
	}
}
