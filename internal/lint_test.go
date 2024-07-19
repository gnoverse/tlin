package internal

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

			engine := &Engine{}
			issues, err := engine.detectUnnecessaryElse(tmpfile)
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

			engine := &Engine{}
			issues, err := engine.detectUnusedFunctions(tmpfile)
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
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "No unnecessary slice length",
			code: `
package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:len(slice)]
}`,
			expected: 1,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "lint-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			engine := &Engine{}
			issues, err := engine.detectUnnecessarySliceLength(tmpfile)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues), "Number of detected unnecessary slice length doesn't match expected")

			if len(issues) > 0 {
				for _, issue := range issues {
					assert.Equal(t, "unnecessary-slice-length", issue.Rule)
					assert.Equal(
						t,
						"unnecessary use of len() in slice expression, can be simplified to a[b:]",
						issue.Message,
					)
				}
			}
		})
	}
}
