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

func TestDetectRepeatedRegexCompilation(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "No repeated compilation",
			code: `
package main

import "regexp"

func noRepeat() {
	r1 := regexp.MustCompile("pattern1")
	r2 := regexp.MustCompile("pattern2")
	_ = r1
	_ = r2
}
`,
			expected: 0,
		},
		{
			name: "Repeated compilation",
			code: `
package main

import "regexp"

func withRepeat() {
	r1 := regexp.MustCompile("pattern")
	r2 := regexp.MustCompile("pattern")
	_ = r1
	_ = r2
}
`,
			expected: 1,
		},
		{
			name: "Multiple repeated compilations",
			code: `
package main

import "regexp"

func multipleRepeats() {
	r1 := regexp.MustCompile("pattern1")
	r2 := regexp.MustCompile("pattern1")
	r3 := regexp.MustCompile("pattern2")
	r4 := regexp.MustCompile("pattern2")
	_ = r1
	_ = r2
	_ = r3
	_ = r4
}
`,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "regex-test")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			tempFile := filepath.Join(tempDir, "test.go")
			err = os.WriteFile(tempFile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tempFile, nil, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectRepeatedRegexCompilation(tempFile, node, fset, types.SeverityError)
			require.NoError(t, err)

			assert.Len(t, issues, tt.expected)

			if tt.expected > 0 {
				for _, issue := range issues {
					assert.Equal(t, "repeatedregexcompilation", issue.Rule)
					assert.Contains(t, issue.Message, "regexp.Compile called with same pattern more than once")
				}
			}
		})
	}
}
