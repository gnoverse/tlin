package lints

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectEarlyReturnOpportunities(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		totalIssues int
	}{
		{
			name: "Simple early return opportunity",
			code: `
package main

func example(x int) string {
    if x > 10 {
        return "greater"
    } else {
        return "less or equal"
    }
}`,
			totalIssues: 1,
		},
		{
			name: "No early return opportunity",
			code: `
package main

func example(x int) string {
    if x > 10 {
        return "greater"
    }
    return "less or equal"
}`,
			totalIssues: 0,
		},
		{
			name: "Nested if with early return opportunity",
			code: `
package main

func example(x, y int) string {
    if x > 10 {
        if y > 20 {
            return "x > 10, y > 20"
        } else {
            return "x > 10, y <= 20"
        }
    } else {
        return "x <= 10"
    }
}`,
			totalIssues: 1,
		},
		{
			name: "Early return with additional logic",
			code: `
package main

func example(x int) string {
	if x > 10 {
		doSomething()
		return "greater"
	} else {
		doSomethingElse()
		return "less or equal"
	}
}`,
			totalIssues: 1,
		},
		{
			name: "Multiple early return opportunities",
			code: `
package main

func example(x, y int) string {
	if x > 10 {
		if y > 20 {
			return "x > 10, y > 20"
		} else {
			return "x > 10, y <= 20"
		}
	} else {
		if y > 20 {
			return "x <= 10, y > 20"
		} else {
			return "x <= 10, y <= 20"
		}
	}
}`,
			totalIssues: 1,
		},
		{
			name: "No early return with single branch",
			code: `
package main

func example(x int) {
	if x > 10 {
		doSomething()
	}
	doSomethingElse()
}`,
			totalIssues: 0,
		},
		{
			name: "No early return when else uses init vars",
			code: `
package main

func example() int {
	if v, ok := get(); !ok {
		return 0
	} else {
		return v
	}
}`,
			totalIssues: 0,
		},
		{
			name: "No early return when else-if uses init vars",
			code: `
package main

func example() int {
	if v, ok := get(); !ok {
		return 0
	} else if v > 0 {
		return v
	} else {
		return 1
	}
}`,
			totalIssues: 0,
		},
		{
			name: "No early return when else shadows init vars",
			code: `
package main

func example() int {
	if v, ok := get(); !ok {
		return 0
	} else {
		v := other()
		return v
	}
}`,
			totalIssues: 0,
		},
		{
			name: "No early return when if body does not terminate",
			code: `
package main

func example(x int) int {
	if x > 0 {
		doSomething()
	} else {
		return 1
	}
	return 2
}`,
			totalIssues: 0,
		},
		{
			name: "No early return when if body panics",
			code: `
package main

func example(x int) int {
	if x > 0 {
		panic("boom")
	} else {
		return 1
	}
	return 2
}`,
			totalIssues: 0,
		},
		{
			name: "Early return with multi-statement else",
			code: `
package main

func example(x int) int {
	if x > 0 {
		return 1
	} else {
		log()
		return 2
	}
}`,
			totalIssues: 1,
		},
		{
			name: "No early return when else-if does not terminate",
			code: `
package main

func example(x, y int) int {
	if x > 0 {
		return 1
	} else if y > 0 {
		doSomething()
	} else {
		return 2
	}
	return 3
}`,
			totalIssues: 0,
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

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			issues, err := DetectEarlyReturnOpportunities(tmpfile, node, fset, types.SeverityError)
			require.NoError(t, err)

			if len(issues) != tt.totalIssues {
				for _, issue := range issues {
					t.Logf("Issue: %v", issue)
					t.Logf("suggestion: %v", issue.Suggestion)
				}
			}
			assert.Equal(t, tt.totalIssues, len(issues), "Number of detected early return opportunities doesn't match expected")

			if len(issues) > 0 {
				for _, issue := range issues {
					assert.Equal(t, "early-return", issue.Rule)
					assert.Contains(t, issue.Message, "can be simplified using early returns")
				}
			}
		})
	}
}

func TestRemoveUnnecessaryElse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "don't need to modify",
			input: `if x {
	println("x")
} else {
	println("hello")
}`,
			expected: `if x {
	println("x")
} else {
	println("hello")
}`,
		},
		{
			name: "remove unnecessary else",
			input: `if x {
	return 1
} else {
	return 2
}`,
			expected: `if x {
	return 1
}
return 2`,
		},
		{
			name: "nested if else",
			input: `if x {
	return 1
}
if z {
	println("x")
} else {
	if y {
		return 2
	} else {
		return 3
	}
}
`,
			expected: `if x {
	return 1
}
if z {
	println("x")
} else {
	if y {
		return 2
	}
	return 3

}`,
		},
		{
			name: "Partially returning nested if-else",
			input: `if x {
	if y {
		return 1
	} else {
		doSomething()
	}
} else {
	return 2
}`,
			expected: `if x {
	if y {
		return 1
	}
	doSomething()

} else {
	return 2
}`,
		},
		{
			name: "Loop control statements",
			input: `if x {
	break
} else {
	continue
}`,
			expected: `if x {
	break
}
continue`,
		},
		{
			name: "Keep else when using init vars",
			input: `if v, ok := get(); !ok {
	return 0
} else {
	return v
}`,
			expected: `if v, ok := get(); !ok {
	return 0
} else {
	return v
}`,
		},
		{
			name: "Keep else when else-if uses init vars",
			input: `if v, ok := get(); !ok {
	return 0
} else if v > 0 {
	return v
} else {
	return 1
}`,
			expected: `if v, ok := get(); !ok {
	return 0
} else if v > 0 {
	return v
} else {
	return 1
}`,
		},
		{
			name: "Keep else when else shadows init vars",
			input: `if v, ok := get(); !ok {
	return 0
} else {
	v := other()
	return v
}`,
			expected: `if v, ok := get(); !ok {
	return 0
} else {
	v := other()
	return v
}`,
		},
		{
			name: "Keep else when if body does not terminate",
			input: `if x > 0 {
	doSomething()
} else {
	return 1
}
return 2`,
			expected: `if x > 0 {
	doSomething()
} else {
	return 1
}
return 2`,
		},
		{
			name: "Keep else when if body panics",
			input: `if x > 0 {
	panic("boom")
} else {
	return 1
}
return 2`,
			expected: `if x > 0 {
	panic("boom")
} else {
	return 1
}
return 2`,
		},
		{
			name: "Flatten else with multiple statements",
			input: `if x > 0 {
	return 1
} else {
	log()
	return 2
}`,
			expected: `if x > 0 {
	return 1
}
log()
return 2`,
		},
		{
			name: "Keep else when else-if does not terminate",
			input: `if x > 0 {
	return 1
} else if y > 0 {
	doSomething()
} else {
	return 2
}
return 3`,
			expected: `if x > 0 {
	return 1
} else if y > 0 {
	doSomething()
} else {
	return 2
}
return 3`,
		},
		{
			name: "Flatten chain when all bodies terminate",
			input: `if x > 0 {
	return 1
} else if y > 0 {
	return 2
} else {
	return 3
}`,
			expected: `if x > 0 {
	return 1
}
if y > 0 {
	return 2
}
return 3`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			improved, err := RemoveUnnecessaryElse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, improved, "Improved code does not match expected output")
		})
	}
}

func TestExtractSnippetPreservesIndentation(t *testing.T) {
	code := "package main\n\nfunc example() {\n\tif x {\n\t\treturn 1\n\t} else {\n\t\treturn 2\n\t}\n}\n"

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", code, 0)
	require.NoError(t, err)

	var ifStmt *ast.IfStmt
	ast.Inspect(node, func(n ast.Node) bool {
		if s, ok := n.(*ast.IfStmt); ok {
			ifStmt = s
			return false
		}
		return true
	})
	require.NotNil(t, ifStmt)

	snippet := extractSnippet(ifStmt, fset, []byte(code))
	expected := "\tif x {\n\t\treturn 1\n\t} else {\n\t\treturn 2\n\t}"
	assert.Equal(t, expected, snippet)
}
