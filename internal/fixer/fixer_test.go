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

func TestAutoFixer(t *testing.T) {
	t.Run("Fix issues", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp(os.TempDir(), "autofixer-test")
		require.NoError(t, err)

		testFile := filepath.Join(tmpDir, "test.go")
		initContent := `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:len(slice)]
}`
		err = os.WriteFile(testFile, []byte(initContent), 0644)
		require.NoError(t, err)

		issues := []tt.Issue{
			{
				Rule:       "simplify-slice-range",
				Filename:   testFile,
				Message:    "unnecessary use of len() in slice expression, can be simplified",
				Start:      token.Position{Line: 5, Column: 6},
				End:        token.Position{Line: 5, Column: 24},
				Suggestion: `_ = slice[:]`,
			},
		}

		autoFixer := New(false)
		autoFixer.autoConfirm = true

		err = autoFixer.Fix(testFile, issues)
		require.NoError(t, err)

		fixed, err := os.ReadFile(testFile)
		require.NoError(t, err)

		expectedContent := `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:]
}
`
		assert.Equal(t, expectedContent, string(fixed))
	})
}
