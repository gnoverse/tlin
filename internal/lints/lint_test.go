package lints

import (
	"os"
	"path/filepath"
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

			issues, err := DetectUnnecessaryElse(tmpfile)
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

func TestDetectUnusedFunctions(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "No unused functions",
			code: `
package main

func main() {
    helper()
}

func helper() {
    println("do something")
}`,
			expected: 0,
		},
		{
			name: "One unused function",
			code: `
package main

func main() {
    println("1")
}

func unused() {
    println("do something")
}`,
			expected: 1,
		},
		{
			name: "Multiple unused functions",
			code: `
package main

func main() {
    used()
}

func used() {
    // this function is called
}

func unused1() {
    // this function is never called
}

func unused2() {
    // this function is also never called
}`,
			expected: 2,
		},
		// 		{
		// 			name: "Unused method",
		// 			code: `
		// package main

		// type MyStruct struct{}

		// func (m MyStruct) Used() {
		//     // this method is used
		// }

		// func (m MyStruct) Unused() {
		//     // this method is never used
		// }

		// func main() {
		//     m := MyStruct{}
		//     m.Used()
		// }`,
		// 			expected: 1,
		// 		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			issues, err := DetectUnusedFunctions(tmpfile)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues), "Number of detected unused functions doesn't match expected")

			if len(issues) > 0 {
				for _, issue := range issues {
					assert.Equal(t, "unused-function", issue.Rule)
					assert.Contains(t, issue.Message, "function", "is declared but not used")
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

			issues, err := DetectUnnecessarySliceLength(tmpfile)
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

			issues, err := DetectUnnecessaryConversions(tmpfile)
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
			err = os.WriteFile(tmpfile, []byte(tt.code), 0644)
			require.NoError(t, err)

			issues, err := DetectLoopAllocation(tmpfile)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues), "Number of issues does not match expected")

			for _, issue := range issues {
				assert.Contains(t, issue.Message, "Potential unnecessary allocation inside loop")
			}
		})
	}
}
