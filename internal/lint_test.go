package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegratedLintEngine(t *testing.T) {
	t.Skip("skipping integrated lint engine test")
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
    x := 1
    fmt.Println("Hello")
}
`,
			expected: []string{
				"x declared and not used",
			},
		},
		{
			name: "Detect multiple issues",
			code: `
package main

import (
    "fmt"
    "strings"
)

func foo() {
	println("unused")
}

func main() {
    x := 1
    y := "unused"
    fmt.Println("Hello")
}
`,
			expected: []string{
				"x declared and not used",
				"y declared and not used",
				`"strings" imported and not used`,
				"function foo is declared and not used",
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

			rootDir := "."
			engine, err := NewEngine(rootDir)
			if err != nil {
				t.Fatalf("unexpected error initializing lint engine: %v", err)
			}

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
