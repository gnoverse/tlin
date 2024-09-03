package lints

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeferChecker(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name: "defer panic",
			code: `
package main

func main() {
	defer panic("this is bad")
}
`,
			expected: []string{"defer-panic"},
		},
		{
			name: "defer in loop",
			code: `
package main

func main() {
	for i := 0; i < 10; i++ {
		defer println(i)
	}
}
`,
			expected: []string{"defer-in-loop"},
		},
		{
			name: "defer nil func",
			code: `
package main

func main() {
	var f func()
	defer f()
}
`,
			expected: []string{"defer-nil-func"},
		},
		{
			name: "multiple issues",
			code: `
		package main
		
		func main() {
			var f func()
			for i := 0; i < 10; i++ {
				defer panic("oops")
			}
			defer f()
		}
		`,
			expected: []string{"defer-in-loop", "defer-panic", "defer-nil-func"},
		},
		{
			name: "no issues",
			code: `
package main

func main() {
	defer println("clean up")
}
`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			require.NoError(t, err)

			checker := NewDeferChecker("test.go", fset)
			issues := checker.Check(f)

			assert.Len(t, issues, len(tt.expected), "Unexpected number of issues")

			for i, exp := range tt.expected {
				if i < len(issues) {
					assert.Equal(t, exp, issues[i].Rule, "Unexpected rule detected")
				}
			}
		})
	}
}

func TestDeferChecker_ReturnInDefer(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool // true -> lint issue
	}{
		{
			name: "return in defer",
			code: `
package main

func foo() {
	defer func() {
		return // This return is useless
	}()
}
`,
			expected: true,
		},
		{
			name: "return in defer with named return value",
			code: `
package main

func foo() (result int) {
	defer func() {
		result = 42 // This is fine
		return // This return is unnecessary but not incorrect
	}()
	return 0
}
`,
			expected: true,
		},
		{
			name: "no return in defer",
			code: `
package main

func foo() {
	defer func() {
		println("cleanup")
	}()
}
`,
			expected: false,
		},
		{
			name: "return in regular function literal",
			code: `
package main

func foo() {
	f := func() {
		return // This is fine
	}
	f()
}
`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			require.NoError(t, err)

			checker := NewDeferChecker("test.go", fset)
			issues := checker.Check(f)

			hasReturnInDeferIssue := false
			for _, issue := range issues {
				if issue.Rule == "return-in-defer" {
					hasReturnInDeferIssue = true
					break
				}
			}

			assert.Equal(t, tt.expected, hasReturnInDeferIssue, "Unexpected result for return in defer check")
		})
	}
}
