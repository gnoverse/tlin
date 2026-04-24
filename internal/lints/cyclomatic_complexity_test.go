package lints

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectHighCyclomaticComplexity(t *testing.T) {
	t.Parallel()
	_, current, _, ok := runtime.Caller(0)
	require.True(t, ok)
	testDir := filepath.Join(filepath.Dir(current), "..", "..", "testdata", "complexity")

	tests := []struct {
		name      string
		filename  string
		threshold int
		expected  int
	}{
		{
			name:      "low complexity below default threshold",
			filename:  "low.gno",
			threshold: 10,
			expected:  0,
		},
		{
			name:      "medium complexity below default threshold",
			filename:  "medium.gno",
			threshold: 10,
			expected:  0,
		},
		{
			name:      "medium complexity flagged at lower threshold",
			filename:  "medium.gno",
			threshold: 3,
			expected:  2,
		},
		{
			name:      "high complexity flagged at default threshold",
			filename:  "high.gno",
			threshold: 10,
			expected:  2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(testDir, tt.filename)

			issues, err := DetectHighCyclomaticComplexity(path, tt.threshold, types.SeverityError)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues),
				"Number of detected high-complexity functions doesn't match expected for %s at threshold %d",
				tt.filename, tt.threshold)

			for _, issue := range issues {
				assert.Equal(t, "high-cyclomatic-complexity", issue.Rule)
				assert.Contains(t, issue.Message, "cyclomatic complexity")
			}
		})
	}
}
