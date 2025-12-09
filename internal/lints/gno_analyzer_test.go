package lints

import (
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLinter(t *testing.T) {
	t.Parallel()
	_, current, _, ok := runtime.Caller(0)
	require.True(t, ok)

	testDir := filepath.Join(filepath.Dir(current), "..", "..", "testdata", "pkg")

	tests := []struct {
		expectedDeps map[string]struct {
			isGno     bool
			isUsed    bool
			isIgnored bool
		}
		filename       string
		expectedIssues []struct {
			rule    string
			message string
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
				"runtime/chain":        {false, true, false},
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

	for _, tt := range tests {
		tt := tt
		t.Run(filepath.Base(tt.filename), func(t *testing.T) {
			t.Parallel()
			fset := token.NewFileSet()
			file, deps, err := analyzeFile(tt.filename)
			require.NoError(t, err)
			require.NotNil(t, file)

			issues := runGnoPackageLinter(file, fset, deps, types.SeverityError)

			assert.Equal(t, len(tt.expectedIssues), len(issues), "Number of issues doesn't match expected for %s", tt.filename)

			for i, expected := range tt.expectedIssues {
				assert.Equal(t, expected.rule, issues[i].Rule, "Rule doesn't match for issue %d in %s", i, tt.filename)
				assert.Contains(t, issues[i].Message, expected.message, "Message doesn't match for issue %d in %s", i, tt.filename)
			}

			for importPath, expected := range tt.expectedDeps {
				dep, exists := deps[importPath]
				assert.True(t, exists, "Dependency %s not found in %s", importPath, tt.filename)
				if exists {
					assert.Equal(t, expected.isGno, dep.IsGno, "IsGno mismatch for %s in %s", importPath, tt.filename)
					assert.Equal(t, expected.isUsed, dep.IsUsed, "IsUsed mismatch for %s in %s", importPath, tt.filename)
					assert.Equal(t, expected.isIgnored, dep.IsIgnored, "IsIgnored mismatch for %s in %s", importPath, tt.filename)
				}
			}
		})
	}
}
