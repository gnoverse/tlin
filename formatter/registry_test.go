package formatter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeFormatter struct{ template string }

func (f *fakeFormatter) IssueTemplate() string { return f.template }

func TestRegistryRegisterAndGet(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register("rule-a", &fakeFormatter{template: "tpl-a"})

	got := r.Get("rule-a")
	require.NotNil(t, got)
	assert.Equal(t, "tpl-a", got.IssueTemplate())
}

func TestRegistryDuplicatePanics(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register("rule-x", &fakeFormatter{})
	assert.PanicsWithValue(t, `formatter.Register: duplicate rule "rule-x"`, func() {
		r.Register("rule-x", &fakeFormatter{})
	})
}

func TestRegistryEmptyNamePanics(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	assert.PanicsWithValue(t, "formatter.Register: empty rule name", func() {
		r.Register("", &fakeFormatter{})
	})
}

// TestRegistryFallback pins the contract that Get always returns a
// usable formatter — unregistered rule names fall back to
// GeneralIssueFormatter rather than nil. The general formatter is
// what most rules use, so callers should never need a nil-check.
func TestRegistryFallback(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	got := r.Get("unregistered-rule")
	require.NotNil(t, got)
	_, isGeneral := got.(*GeneralIssueFormatter)
	assert.True(t, isGeneral, "unregistered rule must fall back to GeneralIssueFormatter, got %T", got)
}

// TestDefaultRegistryHasInitFormatters pins that the three per-file
// init() registrations actually populated the package-level default.
// If a future change misses an init(), Get() silently falls back to
// general — this test catches that drift.
func TestDefaultRegistryHasInitFormatters(t *testing.T) {
	t.Parallel()
	cases := []struct {
		rule string
		want any // type instance for the desired formatter
	}{
		{"high-cyclomatic-complexity", &CyclomaticComplexityFormatter{}},
		{"slice-bounds-check", &SliceBoundsCheckFormatter{}},
		{"gno-mod-tidy", &MissingModPackageFormatter{}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.rule, func(t *testing.T) {
			t.Parallel()
			got := Get(tc.rule)
			assert.IsType(t, tc.want, got,
				"default registry must serve %s through its init() registration", tc.rule)
		})
	}
}
