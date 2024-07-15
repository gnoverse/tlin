package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegratedLintEngine(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{
			name: "Detect unused issues",
			code: `
package main

import (
    "fmt"
)

func main() {
    x := 1      // unused variable
    fmt.Println("Hello")
}
`,
			expected: []string{
				"x declared and not used",
			},
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

			engine := NewEngine()

			issues, err := engine.Run(tmpfile)
			require.NoError(t, err)

			assert.Equal(t, len(tt.expected), len(issues), "Number of issues doesn't match")

			for _, exp := range tt.expected {
				found := false
				for _, issue := range issues {
					if strings.Contains(issue.Message, exp) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected issue not found: "+exp)
			}

			if len(issues) > 0 {
				sourceCode, err := ReadSourceCode(tmpfile)
				require.NoError(t, err)
				formattedIssues := FormatIssuesWithArrows(issues, sourceCode)
				t.Logf("Found issues with arrows:\n%s", formattedIssues)
			}
		})
	}
}
