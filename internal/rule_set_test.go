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