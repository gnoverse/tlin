package lints

import (
	"fmt"
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectUnnecessaryElse(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "Unnecessary else after return",
			code: `
package main

func example() bool {
    if condition {
        return true
    } else {
        return false
    }
}`,
			expected: 1,
		},
		{
			name: "No unnecessary else",
			code: `
package main

func example() {
    if condition {
        doSomething()
    } else {
        doSomethingElse()
    }
}`,
			expected: 0,
		},
		{
			name: "Multiple unnecessary else",
			code: `
package main

func example1() bool {
    if condition1 {
        return true
    } else {
        return false
    }
}

func example2() int {
    if condition2 {
        return 1
    } else {
        return 2
    }
}`,
			expected: 2,
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

			node, fset, err := ParseFile(tmpfile)

			issues, err := DetectUnnecessaryElse(tmpfile, node, fset)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues), "Number of detected unnecessary else statements doesn't match expected")

			if len(issues) > 0 {
				for _, issue := range issues {
					assert.Equal(t, "unnecessary-else", issue.Rule)
					assert.Equal(t, "unnecessary else block", issue.Message)
				}
			}
		})
	}
}

func TestDetectUnnecessarySliceLength(t *testing.T) {
	baseMsg := "unnecessary use of len() in slice expression, can be simplified"
	tests := []struct {
		name     string
		code     string
		expected int
		message  string
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
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			node, fset, err := ParseFile(tmpfile)
			require.NoError(t, err)

			issues, err := DetectUnnecessarySliceLength(tmpfile, node, fset)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues), "Number of detected unnecessary slice length doesn't match expected")

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
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			node, fset, err := ParseFile(tmpfile)
			require.NoError(t, err)

			issues, err := DetectUnnecessaryConversions(tmpfile, node, fset)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues), "Number of detected unnecessary type conversions doesn't match expected")

			if len(issues) > 0 {
				for _, issue := range issues {
					assert.Equal(t, "unnecessary-type-conversion", issue.Rule)
					assert.Equal(t, "unnecessary type conversion", issue.Message)
				}
			}
		})
	}
}

func TestDetectLoopAllocation(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "Allocation in for loop",
			code: `
package main

func main() {
	for i := 0; i < 10; i++ {
		_ = make([]int, 10)
	}
}`,
			expected: 1,
		},
		{
			name: "Allocation in range loop",
			code: `
package main

func main() {
	slice := []int{1, 2, 3}
	for _, v := range slice {
		_ = new(int)
		_ = v
	}
}`,
			expected: 1,
		},
		{
			name: "Multiple allocations in loop",
			code: `
package main

func main() {
	for i := 0; i < 10; i++ {
		_ = make([]int, 10)
		_ = new(string)
	}
}`,
			expected: 2,
		},
		// 		{
		// 			name: "Variable allocation in loop",
		// 			// ref: https://stackoverflow.com/questions/77180437/understanding-short-variable-declaration-in-loop-resulting-unnecessary-memory-a
		// 			code: `
		// 			package main

		// import "fmt"

		// func main() {
		//     for i:=0; i<10; i++ {
		//         a:=i+1 // BAD!: allocates memory in every iteration
		//         fmt.Printf("i-val: %d, i-addr: %p, a-val: %d, a-addr: %p\n", i, &i, a, &a)
		//     }
		// }`,
		// 			expected: 1,
		// 		},
		{
			name: "No allocation in loop",
			code: `
package main

func main() {
	slice := make([]int, 10)
	for i := 0; i < 10; i++ {
		slice[i] = i
	}
}`,
			expected: 0,
		},
		{
			name: "Allocation outside loop",
			code: `
package main

func main() {
	_ = make([]int, 10)
	for i := 0; i < 10; i++ {
		// Do something
	}
}`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			node, fset, err := ParseFile(tmpfile)
			require.NoError(t, err)

			issues, err := DetectLoopAllocation(tmpfile, node, fset)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues), "Number of issues does not match expected")

			for _, issue := range issues {
				assert.Contains(t, issue.Message, "Potential unnecessary allocation inside loop")
			}
		})
	}
}

func TestDetectEmitFormat(t *testing.T) {
	_, current, _, _ := runtime.Caller(0)
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
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(testDir, tt.filename)
			content, err := os.ReadFile(path)
			require.NoError(t, err)

			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, content, 0o644)
			require.NoError(t, err)

			node, fset, err := ParseFile(tmpfile)
			require.NoError(t, err)

			issues, err := DetectEmitFormat(tmpfile, node, fset)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues), fmt.Sprintf("Number of detected issues doesn't match expected for %s. %v", tt.filename, issues))

			if len(issues) > 0 {
				assert.Equal(t, "emit-format", issues[0].Rule)
				assert.Contains(t, issues[0].Message, "Consider formatting std.Emit call for better readability")
			}
		})
	}
}

func TestFormatEmitCall(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "Simple Emit call",
			input: `std.Emit("OwnershipChange", "newOwner", newOwner.String())`,
			expected: `std.Emit(
    "OwnershipChange",
    "newOwner", newOwner.String(),
)`,
		},
		{
			name:  "Emit call with multiple key-value pairs",
			input: `std.Emit("OwnershipChange", "newOwner", newOwner.String(), "oldOwner", oldOwner.String())`,
			expected: `std.Emit(
    "OwnershipChange",
    "newOwner", newOwner.String(),
    "oldOwner", oldOwner.String(),
)`,
		},
		{
			name:  "Emit call with function calls as values",
			input: `std.Emit("Transfer", "from", sender.Address(), "to", recipient.Address(), "amount", token.Format(amount))`,
			expected: `std.Emit(
    "Transfer",
    "from", sender.Address(),
    "to", recipient.Address(),
    "amount", token.Format(amount),
)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.input)
			assert.NoError(t, err)

			callExpr, ok := expr.(*ast.CallExpr)
			assert.True(t, ok)

			result := formatEmitCall(callExpr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectSliceBoundsCheck(t *testing.T) {
	code := `
package main

type Item struct {
    Name  string
    Value int
}

func main() {
    sourceItems := []*Item{
        {"item1", 10},
        {"item2", 20},
        {"item3", 30},
    }

    destinationItems := make([]*Item, 0, len(sourceItems))

    i := 0
    for _, item := range sourceItems {
        destinationItems[i] = item // Expect out-of-range linter warning here
        i++
    }
}
`

	tmpfile, err := os.CreateTemp("", "example.*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(code)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	node, fset, err := ParseFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	issues, err := DetectSliceBoundCheck(tmpfile.Name(), node, fset)
	if err != nil {
		t.Fatal(err)
	}

	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, but got %d", len(issues))
	}

	if len(issues) > 0 {
		if issues[0].Rule != "slice-bounds-check" {
			t.Errorf("Expected rule to be 'slice-bounds-check', but got '%s'", issues[0].Rule)
		}
		if !strings.Contains(issues[0].Message, "Potential slice bounds check failure") {
			t.Errorf("Unexpected message: %s", issues[0].Message)
		}
	}
}
