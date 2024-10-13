package lints

import (
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
	t.Skip("skipping test")
	tests := []struct {
		name     string
		code     string
		expected int // number of expected issues
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
			expected: 1,
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
			expected: 0,
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
			expected: 2, // One for the outer if-else, one for the inner
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
			expected: 1,
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
			expected: 3, // One for the outer if-else, two for the inner ones
		},
		{
			name: "Early return with break",
			code: `
package main

func example(x int) {
    for i := 0; i < 10; i++ {
        if x > i {
            doSomething()
            break
        } else {
            continue
        }
    }
}`,
			expected: 1,
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

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			issues, err := DetectEarlyReturnOpportunities(tmpfile, node, fset, types.SeverityError)
			require.NoError(t, err)

			// assert.Equal(t, tt.expected, len(issues), "Number of detected early return opportunities doesn't match expected")
			if len(issues) != tt.expected {
				for _, issue := range issues {
					t.Logf("Issue: %v", issue)
					t.Logf("suggestion: %v", issue.Suggestion)
				}
			}
			assert.Equal(t, tt.expected, len(issues), "Number of detected early return opportunities doesn't match expected")

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
