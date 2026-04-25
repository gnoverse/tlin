package rule

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Repeated TypesInfo() calls must return the same pointer so the
// type checker runs at most once per file regardless of how many
// rules ask for it.
func TestAnalysisContextTypesInfoSharedAcrossCalls(t *testing.T) {
	t.Parallel()

	src := `package demo

func sum(xs []int) int {
	total := 0
	for i := 0; i < len(xs); i++ {
		total += xs[i]
	}
	return total
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "demo.go", src, parser.ParseComments)
	require.NoError(t, err)

	ctx := &AnalysisContext{File: file, Fset: fset}

	first := ctx.TypesInfo()
	require.NotNil(t, first)
	second := ctx.TypesInfo()
	assert.Same(t, first, second)
}

// A context built without File/Fset must not panic inside
// types.Config.Check.
func TestAnalysisContextTypesInfoNilInputs(t *testing.T) {
	t.Parallel()

	ctx := &AnalysisContext{}
	assert.NotNil(t, ctx.TypesInfo())
}

func TestRemapFilename(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		workingPath  string
		originalPath string
		input        string
		want         string
	}{
		{"matches working path", "/tmp/temp_x.go", "/src/foo.gno", "/tmp/temp_x.go", "/src/foo.gno"},
		{"unrelated path passes through", "/tmp/temp_x.go", "/src/foo.gno", "/other/file.go", "/other/file.go"},
		{"empty working path disables remap", "", "/src/foo.gno", "/tmp/temp_x.go", "/tmp/temp_x.go"},
		{"identical paths unchanged", "/src/a.go", "/src/a.go", "/src/a.go", "/src/a.go"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := &AnalysisContext{WorkingPath: tc.workingPath, OriginalPath: tc.originalPath}
			assert.Equal(t, tc.want, ctx.RemapFilename(tc.input))
		})
	}
}

// NewIssue must remap Filename, Start.Filename, and End.Filename to
// OriginalPath so the temp .go path can't leak into emitted issues.
func TestNewIssueFillsCanonicalFields(t *testing.T) {
	t.Parallel()

	const src = "package demo\n\nfunc f() {}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "/tmp/temp_demo.go", src, parser.ParseComments)
	require.NoError(t, err)

	ctx := &AnalysisContext{
		OriginalPath: "/src/demo.gno",
		WorkingPath:  "/tmp/temp_demo.go",
		File:         file,
		Fset:         fset,
		Severity:     1, // arbitrary non-zero
	}

	issue := ctx.NewIssue("demo-rule", file.Pos(), file.End())
	assert.Equal(t, "demo-rule", issue.Rule)
	assert.Equal(t, "/src/demo.gno", issue.Filename, "top-level Filename must use OriginalPath")
	assert.Equal(t, "/src/demo.gno", issue.Start.Filename, "Start.Filename must be remapped")
	assert.Equal(t, "/src/demo.gno", issue.End.Filename, "End.Filename must be remapped")
	assert.Equal(t, ctx.Severity, issue.Severity)
}
