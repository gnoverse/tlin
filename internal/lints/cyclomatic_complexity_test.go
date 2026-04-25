package lints

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gnolang/tlin/internal/rule"
	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCyclomaticComplexityRule_Check(t *testing.T) {
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
			content, err := os.ReadFile(path)
			require.NoError(t, err)

			file, fset, err := ParseFile(path, content)
			require.NoError(t, err)

			r := &cyclomaticComplexityRule{threshold: tt.threshold}
			issues, err := r.Check(&rule.AnalysisContext{
				OriginalPath: path,
				WorkingPath:  path,
				File:         file,
				Fset:         fset,
				Severity:     types.SeverityError,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues),
				"Number of detected high-complexity functions doesn't match expected for %s at threshold %d",
				tt.filename, tt.threshold)

			for _, issue := range issues {
				assert.Equal(t, "high-cyclomatic-complexity", issue.Rule)
				assert.Contains(t, issue.Message, "cyclomatic complexity")
				assert.Equal(t, types.SeverityError, issue.Severity,
					"Issue severity must reflect ctx.Severity, not the rule's DefaultSeverity")
			}
		})
	}
}

func TestCyclomaticComplexityRule_ParseConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		raw         any
		startThres  int
		wantErr     bool
		wantThres   int // expected threshold after ParseConfig (regardless of err)
	}{
		{"int threshold", map[string]any{"threshold": 5}, 10, false, 5},
		{"int64 threshold", map[string]any{"threshold": int64(7)}, 10, false, 7},
		{"float64 threshold (yaml decoded)", map[string]any{"threshold": 8.0}, 10, false, 8},
		{"missing threshold keeps default", map[string]any{}, 10, false, 10},
		{"non-map raw", "ten", 10, true, 10},
		{"non-numeric threshold", map[string]any{"threshold": "ten"}, 10, true, 10},
		{"zero threshold rejected", map[string]any{"threshold": 0}, 10, true, 10},
		{"negative threshold rejected", map[string]any{"threshold": -1}, 10, true, 10},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := &cyclomaticComplexityRule{threshold: tc.startThres}
			err := r.ParseConfig(tc.raw)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantThres, r.threshold,
				"threshold field should be %d after ParseConfig (errors must not corrupt state)",
				tc.wantThres)
		})
	}
}
