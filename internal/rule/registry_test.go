package rule

import (
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

