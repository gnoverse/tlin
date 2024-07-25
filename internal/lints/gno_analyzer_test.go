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

	pkg, deps, err := analyzePackage(testDir)
	require.NoError(t, err)
	require.NotNil(t, pkg)
	require.NotNil(t, deps)

	issues := runGnoPackageLinter(pkg, deps)

	expectedIssues := []struct {
		rule    string
		message string
	}{
		{"unused-import", "unused import: strings"},
	}

	assert.Equal(t, len(expectedIssues), len(issues), "Number of issues doesn't match expected")

	for i, expected := range expectedIssues {
		assert.Equal(t, expected.rule, issues[i].Rule, "Rule doesn't match for issue %d", i)
		assert.Contains(t, issues[i].Message, expected.message, "Message doesn't match for issue %d", i)
	}

	expectedDeps := map[string]struct {
		isGno  bool
		isUsed bool
	}{
		"fmt":                  {false, true},
		"gno.land/p/demo/ufmt": {true, true},
		"strings":              {false, false},
		"std":                  {true, true},
	}

	for importPath, expected := range expectedDeps {
		dep, exists := deps[importPath]
		assert.True(t, exists, "Dependency %s not found", importPath)
		if exists {
			assert.Equal(t, expected.isGno, dep.IsGno, "IsGno mismatch for %s", importPath)
			assert.Equal(t, expected.isUsed, dep.IsUsed, "IsUsed mismatch for %s", importPath)
		}
	}
}
