package rule

import (
	"go/ast"
	"go/token"
	"testing"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRule struct {
	name string
	sev  tt.Severity
}

func (f *fakeRule) Name() string                                { return f.name }
func (f *fakeRule) DefaultSeverity() tt.Severity                { return f.sev }
func (f *fakeRule) Check(*AnalysisContext) ([]tt.Issue, error)  { return nil, nil }

func TestRegistryRegisterAndAll(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(&fakeRule{name: "a", sev: tt.SeverityWarning})
	r.Register(&fakeRule{name: "b", sev: tt.SeverityError})

	got := r.All()
	require.Len(t, got, 2)
	assert.Equal(t, "a", got["a"].Name())
	assert.Equal(t, tt.SeverityError, got["b"].DefaultSeverity())
}

func TestRegistryDuplicatePanics(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(&fakeRule{name: "x"})
	assert.PanicsWithValue(t, `rule.Register: duplicate name "x"`, func() {
		r.Register(&fakeRule{name: "x"})
	})
}

func TestRegistryEmptyNamePanics(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	assert.PanicsWithValue(t, "rule.Register: empty Name()", func() {
		r.Register(&fakeRule{name: ""})
	})
}

func TestRegistryAllReturnsCopy(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register(&fakeRule{name: "a"})

	got := r.All()
	delete(got, "a")

	again := r.All()
	assert.Len(t, again, 1,
		"All() must return a copy; mutating the returned map must not affect the registry")
}

// TestLegacyAdapterPropagatesContextSeverity guards a subtle
// invariant: ctx.Severity (engine-resolved) is what reaches the inner
// check function, not r.DefaultSeverity(). Engine relies on this when
// applying config overrides.
func TestLegacyAdapterPropagatesContextSeverity(t *testing.T) {
	t.Parallel()
	var captured tt.Severity
	check := func(filename string, _ *ast.File, _ *token.FileSet, sev tt.Severity) ([]tt.Issue, error) {
		captured = sev
		return []tt.Issue{{Rule: "x", Filename: filename}}, nil
	}

	r := NewLegacy("x", tt.SeverityInfo, check)
	assert.Equal(t, "x", r.Name())
	assert.Equal(t, tt.SeverityInfo, r.DefaultSeverity())

	ctx := &AnalysisContext{
		WorkingPath: "/tmp/foo.go",
		Severity:    tt.SeverityWarning, // overridden by engine, ≠ DefaultSeverity
	}
	issues, err := r.Check(ctx)
	require.NoError(t, err)
	assert.Equal(t, tt.SeverityWarning, captured,
		"legacy adapter must pass ctx.Severity, not DefaultSeverity, to the inner check")
	require.Len(t, issues, 1)
	assert.Equal(t, "/tmp/foo.go", issues[0].Filename)
}
