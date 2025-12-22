package lints

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectDivisionByZero(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		code          string
		expectedCount int
		expectedLevel types.Severity
		expectedText  string
	}{
		{
			name: "definitely zero divisor",
			code: `
package main

func main() {
	y := 0
	_ = 10 / y
}`,
			expectedCount: 1,
			expectedLevel: types.SeverityError,
			expectedText:  "divisor is definitely zero",
		},
		{
			name: "maybe zero divisor",
			code: `
package main

func main() {
	var y int
	_ = 10 / y
}`,
			expectedCount: 1,
			expectedLevel: types.SeverityWarning,
			expectedText:  "divisor may be zero",
		},
		{
			name: "guarded non-zero",
			code: `
package main

func main() {
	y := 0
	if y != 0 {
		_ = 10 / y
	}
}`,
			expectedCount: 0,
		},
		{
			name: "division-like call",
			code: `
package main

func main() {
	var z any
	var y int
	_ = z
	z.Div(1, y)
}`,
			expectedCount: 1,
			expectedLevel: types.SeverityWarning,
			expectedText:  "divisor may be zero",
		},
		{
			name: "compound assignment division",
			code: `
package main

func main() {
	x := 10
	y := 0
	x /= y
}`,
			expectedCount: 1,
			expectedLevel: types.SeverityError,
			expectedText:  "divisor is definitely zero",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tmpDir, err := os.MkdirTemp("", "lint-divzero-test")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			tmpfile := filepath.Join(tmpDir, "test.go")
			err = os.WriteFile(tmpfile, []byte(tt.code), 0o644)
			require.NoError(t, err)

			node, fset, err := ParseFile(tmpfile, nil)
			require.NoError(t, err)

			issues, err := DetectDivisionByZero(tmpfile, node, fset, types.SeverityWarning)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedCount, len(issues))
			if tt.expectedCount == 0 {
				return
			}

			for _, issue := range issues {
				assert.Equal(t, "division-by-zero", issue.Rule)
				assert.Equal(t, tt.expectedLevel, issue.Severity)
				assert.Contains(t, issue.Message, tt.expectedText)
			}
		})
	}
}
