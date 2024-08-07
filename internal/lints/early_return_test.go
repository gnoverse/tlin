package lints

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectEarlyReturnOpportunities(t *testing.T) {
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
            err = os.WriteFile(tmpfile, []byte(tt.code), 0644)
            require.NoError(t, err)

            fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

            issues, err := DetectEarlyReturnOpportunities(tmpfile, node, fset)
            require.NoError(t, err)

            // assert.Equal(t, tt.expected, len(issues), "Number of detected early return opportunities doesn't match expected")
            if len(issues) != tt.expected {
                for _, issue := range issues {
                    t.Logf("Issue: %v", issue)
                }
            }
            assert.Equal(t, tt.expected, len(issues), "Number of detected early return opportunities doesn't match expected")

            if len(issues) > 0 {
                for _, issue := range issues {
                    assert.Equal(t, "early-return-opportunity", issue.Rule)
                    assert.Contains(t, issue.Message, "can be simplified using early returns")
                }
            }
        })
    }
}
