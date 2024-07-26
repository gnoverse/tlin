package lints

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLinter(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	require.True(t, ok)

	testDir := filepath.Join(filepath.Dir(current), "..", "..", "testdata", "pkg")

	tests := []struct {
		filename       string
		expectedIssues []struct {
			rule    string
			message string
		}
		expectedDeps map[string]struct {
			isGno     bool
			isUsed    bool
			isIgnored bool
		}
	}{
		{
			filename: filepath.Join(testDir, "pkg0.gno"),
			expectedIssues: []struct {
				rule    string
				message string
			}{
				{"unused-import", "unused import: strings"},
			},
			expectedDeps: map[string]struct {
				isGno     bool
				isUsed    bool
				isIgnored bool
			}{
				"fmt":                  {false, true, false},
				"gno.land/p/demo/ufmt": {true, true, false},
				"strings":              {false, false, false},
				"std":                  {true, true, false},
				"gno.land/p/demo/diff": {true, false, true},
			},
		},
		{
			filename: filepath.Join(testDir, "pkg1.gno"),
			expectedIssues: []struct {
				rule    string
				message string
			}{},
			expectedDeps: map[string]struct {
				isGno     bool
				isUsed    bool
				isIgnored bool
			}{
				"strings": {false, true, false},
			},
		},
	}

	for _, tc := range tests {
		t.Run(filepath.Base(tc.filename), func(t *testing.T) {
			file, deps, err := analyzeFile(tc.filename)
			require.NoError(t, err)
			require.NotNil(t, file)

			issues := runGnoPackageLinter(file, deps)

			assert.Equal(t, len(tc.expectedIssues), len(issues), "Number of issues doesn't match expected for %s", tc.filename)

			for i, expected := range tc.expectedIssues {
				assert.Equal(t, expected.rule, issues[i].Rule, "Rule doesn't match for issue %d in %s", i, tc.filename)
				assert.Contains(t, issues[i].Message, expected.message, "Message doesn't match for issue %d in %s", i, tc.filename)
			}

			for importPath, expected := range tc.expectedDeps {
				dep, exists := deps[importPath]
				assert.True(t, exists, "Dependency %s not found in %s", importPath, tc.filename)
				if exists {
					assert.Equal(t, expected.isGno, dep.IsGno, "IsGno mismatch for %s in %s", importPath, tc.filename)
					assert.Equal(t, expected.isUsed, dep.IsUsed, "IsUsed mismatch for %s in %s", importPath, tc.filename)
					assert.Equal(t, expected.isIgnored, dep.IsIgnored, "IsIgnored mismatch for %s in %s", importPath, tc.filename)
				}
			}
		})
	}
}
