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
