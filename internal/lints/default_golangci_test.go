package lints

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeGolangciOutput(t *testing.T) {
	t.Parallel()

	t.Run("empty input yields no issues", func(t *testing.T) {
		t.Parallel()
		for _, in := range [][]byte{nil, []byte(""), []byte("   \n\t  ")} {
			got, err := decodeGolangciOutput(in)
			require.NoError(t, err)
			assert.Empty(t, got.Issues)
		}
	})

	t.Run("valid JSON populates issues", func(t *testing.T) {
		t.Parallel()
		in := []byte(`{"Issues":[{"FromLinter":"errcheck","Text":"unchecked error","Pos":{"Filename":"a.go","Line":12,"Column":3}}]}`)

		got, err := decodeGolangciOutput(in)
		require.NoError(t, err)
		require.Len(t, got.Issues, 1)
		assert.Equal(t, "errcheck", got.Issues[0].FromLinter)
		assert.Equal(t, "unchecked error", got.Issues[0].Text)
		assert.Equal(t, "a.go", got.Issues[0].Pos.Filename)
		assert.Equal(t, 12, got.Issues[0].Pos.Line)
		assert.Equal(t, 3, got.Issues[0].Pos.Column)
	})

	t.Run("malformed input returns decode error", func(t *testing.T) {
		t.Parallel()
		got, err := decodeGolangciOutput([]byte("not json"))
		require.Error(t, err)
		assert.Empty(t, got.Issues)
	})
}

func TestSnippet(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []byte
		n    int
		want string
	}{
		{"empty", nil, 10, ""},
		{"whitespace only", []byte("  \n\t"), 10, ""},
		{"under limit", []byte("hello"), 10, "hello"},
		{"trimmed under limit", []byte("  hello\n"), 10, "hello"},
		{"over limit truncates after trim", []byte("  abcdefghij"), 5, "abcde"},
		{"exactly at limit", []byte("abcde"), 5, "abcde"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := snippet(tc.in, tc.n)
			assert.Equal(t, tc.want, got)
			assert.LessOrEqual(t, len(got), tc.n)
			assert.Equal(t, strings.TrimSpace(got), got)
		})
	}
}
