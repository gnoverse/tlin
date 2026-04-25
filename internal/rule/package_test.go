package rule

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageContextRemapFilename(t *testing.T) {
	t.Parallel()

	pctx := &PackageContext{
		Dir:           "/src/demo",
		OriginalPaths: []string{"/src/demo/a.gno", "/src/demo/b.go"},
		WorkingPaths:  []string{"/src/demo/temp_a.go", "/src/demo/b.go"},
	}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"working path remaps to original", "/src/demo/temp_a.go", "/src/demo/a.gno"},
		{"plain .go path passes through", "/src/demo/b.go", "/src/demo/b.go"},
		{"unrelated path passes through", "/other/file.go", "/other/file.go"},
		{"empty input passes through", "", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, pctx.RemapFilename(tc.in))
		})
	}
}

func TestPackageContextInScope(t *testing.T) {
	t.Parallel()

	pctx := &PackageContext{
		WorkingPaths: []string{"/src/demo/temp_a.go", "/src/demo/b.go"},
	}
	assert.True(t, pctx.InScope("/src/demo/temp_a.go"))
	assert.True(t, pctx.InScope("/src/demo/b.go"))
	assert.False(t, pctx.InScope("/src/demo/other.go"))
	assert.False(t, pctx.InScope(""))
}

// Dir must come from WorkingPath so loaders scan the temp .go that
// the engine actually parsed, not the .gno sibling.
func TestAnalysisContextSinglePackage(t *testing.T) {
	t.Parallel()

	ctx := &AnalysisContext{
		OriginalPath: "/src/demo/a.gno",
		WorkingPath:  "/tmp/demo/temp_a.go",
		Severity:     1,
	}
	pctx := ctx.SinglePackage()

	assert.Equal(t, filepath.Dir(ctx.WorkingPath), pctx.Dir,
		"Dir derives from WorkingPath so loaders see temp .go")
	assert.Equal(t, []string{ctx.OriginalPath}, pctx.OriginalPaths)
	assert.Equal(t, []string{ctx.WorkingPath}, pctx.WorkingPaths)
	assert.Equal(t, ctx.Severity, pctx.Severity)
}

// SinglePackage on an empty AnalysisContext (RunSource path) must
// not panic and produces an empty-but-usable PackageContext.
func TestAnalysisContextSinglePackageEmptyWorking(t *testing.T) {
	t.Parallel()

	ctx := &AnalysisContext{}
	pctx := ctx.SinglePackage()
	assert.Equal(t, "", pctx.Dir)
	assert.Equal(t, []string{""}, pctx.OriginalPaths)
	assert.Equal(t, []string{""}, pctx.WorkingPaths)
}
