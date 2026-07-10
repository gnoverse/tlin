package lints

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectStoredRealm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "package-level realm var",
			code: `
package vault

var savedRealm realm
`,
			expected: 1,
		},
		{
			name: "struct field of type realm",
			code: `
package vault

type record struct {
	owner realm
	name  string
}
`,
			expected: 1,
		},
		{
			name: "storing address string is fine",
			code: `
package vault

var savedAddr address

func Save(cur realm) {
	savedAddr = cur.Previous().Address()
}
`,
			expected: 0,
		},
		{
			name: "local realm variable is fine",
			code: `
package vault

func Save(cur realm) {
	prev := cur.Previous()
	_ = prev
}
`,
			expected: 0,
		},
		{
			name: "grouped var block with realm entry",
			code: `
package vault

var (
	name  string
	saved realm
)
`,
			expected: 1,
		},
		{
			name: "realm containers are flagged",
			code: `
package vault

var (
	realms  []realm
	byName  map[string]realm
	pointer *realm
)

type record struct {
	owners map[realm]string
}
`,
			expected: 4,
		},
		{
			name: "user-defined realm-named struct type is still flagged by name",
			code: `
package vault

type box struct {
	r realm
}

var b box
`,
			expected: 1,
		},
	}

	t.Run("plain go file is not checked", func(t *testing.T) {
		t.Parallel()

		code := `
package game

type realm struct{ name string }

var current realm
`
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, "file.go", code, parser.ParseComments)
		require.NoError(t, err)

		issues, err := DetectStoredRealm(newTestContext("file.go", node, fset, []byte(code)))
		require.NoError(t, err)
		assert.Empty(t, issues)
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "file.gno", tc.code, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectStoredRealm(newTestContext("file.gno", node, fset, []byte(tc.code)))
			require.NoError(t, err)

			assert.Equal(t, tc.expected, len(issues), "unexpected issue count")
			for _, issue := range issues {
				assert.Equal(t, "stored-realm", issue.Rule)
				assert.NotEmpty(t, issue.Message)
			}
		})
	}
}
