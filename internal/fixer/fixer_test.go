package fixer

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	tt "github.com/gnoswap-labs/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const confidenceThreshold = 0.8

func TestAutoFixer(t *testing.T) {
	t.Run("Fix - Simple case", func(t *testing.T) {
		_, testFile, cleanup := setupTestFile(t, `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`)
		defer cleanup()

		issues := []tt.Issue{
			{
				Rule:       "simplify-slice-range",
				Filename:   testFile,
				Message:    "unnecessary use of len() in slice expression, can be simplified",
				Start:      token.Position{Line: 5, Column: 5},
				End:        token.Position{Line: 5, Column: 24},
				Suggestion: "_ = slice[:]",
				Confidence: 0.9,
			},
		}

		fixer := New(false, confidenceThreshold)
		fixer.autoConfirm = true
		err := fixer.Fix(testFile, issues)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		expected := `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:]
}
`
		assert.Equal(t, expected, string(content))
	})

	t.Run("Don't Fix - Not enough confidence", func(t *testing.T) {
		_, testFile, cleanup := setupTestFile(t, `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`)
		defer cleanup()

		issues := []tt.Issue{
			{
				Rule:       "simplify-slice-range",
				Filename:   testFile,
				Message:    "unnecessary use of len() in slice expression, can be simplified",
				Start:      token.Position{Line: 5, Column: 5},
				End:        token.Position{Line: 5, Column: 24},
				Suggestion: "_ = slice[:]",
				Confidence: 0.3,
			},
		}

		fixer := New(false, confidenceThreshold)
		fixer.autoConfirm = true
		err := fixer.Fix(testFile, issues)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		expected := `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:len(slice)]
}
`
		assert.Equal(t, expected, string(content))
	})

	t.Run("Fix - Multiple issues", func(t *testing.T) {
		_, testFile, cleanup := setupTestFile(t, `package main

func main() {
	slice1 := []int{1, 2, 3}
	_ = slice1[:len(slice1)]

	slice2 := []string{"a", "b", "c"}
	_ = slice2[:len(slice2)]
}`)
		defer cleanup()

		issues := []tt.Issue{
			{
				Rule:       "simplify-slice-range",
				Filename:   testFile,
				Message:    "unnecessary use of len() in slice expression, can be simplified",
				Start:      token.Position{Line: 5, Column: 5},
				End:        token.Position{Line: 5, Column: 26},
				Suggestion: "_ = slice1[:]",
				Confidence: 0.9,
			},
			{
				Rule:       "simplify-slice-range",
				Filename:   testFile,
				Message:    "unnecessary use of len() in slice expression, can be simplified",
				Start:      token.Position{Line: 8, Column: 5},
				End:        token.Position{Line: 8, Column: 26},
				Suggestion: "_ = slice2[:]",
				Confidence: 0.9,
			},
		}

		fixer := New(false, confidenceThreshold)
		fixer.autoConfirm = true
		err := fixer.Fix(testFile, issues)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		expected := `package main

func main() {
	slice1 := []int{1, 2, 3}
	_ = slice1[:]

	slice2 := []string{"a", "b", "c"}
	_ = slice2[:]
}
`
		assert.Equal(t, expected, string(content))
	})

	t.Run("Fix - Preserve indentation", func(t *testing.T) {
		_, testFile, cleanup := setupTestFile(t, `package main

func main() {
	if true {
		slice := []int{1, 2, 3}
		_ = slice[:len(slice)]
	}
}`)
		defer cleanup()

		issues := []tt.Issue{
			{
				Rule:       "simplify-slice-range",
				Filename:   testFile,
				Message:    "unnecessary use of len() in slice expression, can be simplified",
				Start:      token.Position{Line: 6, Column: 3},
				End:        token.Position{Line: 6, Column: 22},
				Suggestion: "_ = slice[:]",
				Confidence: 0.9,
			},
		}

		fixer := New(false, confidenceThreshold)
		fixer.autoConfirm = true
		err := fixer.Fix(testFile, issues)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		expected := `package main

func main() {
	if true {
		slice := []int{1, 2, 3}
		_ = slice[:]
	}
}
`
		assert.Equal(t, expected, string(content))
	})

	t.Run("DryRun", func(t *testing.T) {
		_, testFile, cleanup := setupTestFile(t, `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`)
		defer cleanup()

		issues := []tt.Issue{
			{
				Rule:       "simplify-slice-range",
				Filename:   testFile,
				Message:    "unnecessary use of len() in slice expression, can be simplified",
				Start:      token.Position{Line: 5, Column: 5},
				End:        token.Position{Line: 5, Column: 24},
				Suggestion: "_ = slice[:]",
				Confidence: 0.9,
			},
		}

		fixer := New(true, confidenceThreshold) // dry-run mode
		fixer.autoConfirm = true
		err := fixer.Fix(testFile, issues)
		require.NoError(t, err)

		content, err := os.ReadFile(testFile)
		require.NoError(t, err)

		// Content should remain unchanged in dry-run mode
		expected := `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`
		assert.Equal(t, expected, string(content))
	})
}

func setupTestFile(t *testing.T, content string) (string, string, func()) {
	tmpDir, err := os.MkdirTemp("", "autofixer-test")
	require.NoError(t, err)

	testFile := filepath.Join(tmpDir, "test.go")
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, testFile, cleanup
}
