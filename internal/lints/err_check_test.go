package lints

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectErrCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		code     string
		expected int
		messages []string
	}{
		{
			name: "Simple error ignored",
			code: `
package main
import "os"
func main() {
    f, _ := os.Open("file.txt")
    defer f.Close()
}`,
			expected: 1,
			messages: []string{
				"error return value is ignored with blank identifier",
			},
		},
		{
			name: "Simple error ignored 2",
			code: `
package main

import "errors"

func foo() (int, error) {
	return 0, errors.New("test")
}

func main() {
    v, _ := foo()
	println(v)
}`,
			expected: 1,
			messages: []string{
				"error return value is ignored with blank identifier",
			},
		},
		{
			name: "Error not checked after assignment",
			code: `
package main
import "os"
func main() {
    var err error
    f, err := os.Open("file.txt")
    f.Write([]byte("test"))
    defer f.Close()
}`,
			expected: 1,
			messages: []string{
				"error-returning function call's result is ignored",
			},
		},
		{
			name: "Proper error handling",
			code: `
package main

import "errors"

func foo() (int, error) {
	return 0, errors.New("test")
}

func main() {
    v, err := foo()
    if err != nil {
		panic(err)
	}
	println(v)
}`,
			expected: 0,
			messages: []string{},
		},
		{
			name: "Multiple error ignoring patterns",
			code: `
package main
import (
    "os"
    "encoding/json"
)
func main() {
    f, _ := os.Open("file.txt")
    json.Unmarshal([]byte("{}"), &struct{}{})
    f.Write([]byte("test"))
}`,
			expected: 3,
			messages: []string{
				"error return value is ignored with blank identifier",
				"error-returning function call's result is ignored",
				"error-returning function call's result is ignored",
			},
		},
		{
			name: "Non-error returning functions",
			code: `
package main
func main() {
    println("test")
    x := len("test")
    _ = cap([]int{1,2,3})
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

			issues, err := DetectErrCheck(tmpfile, node, fset, types.SeverityError)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues),
				"Number of detected issues doesn't match expected")

			if len(issues) > 0 {
				for i, issue := range issues {
					assert.Equal(t, "error-check", issue.Rule)
					assert.Contains(t, issue.Message, tt.messages[i])
				}
			}
		})
	}
}
