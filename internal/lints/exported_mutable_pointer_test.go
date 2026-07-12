package lints

import (
	"context"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectExportedMutablePointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "returns package-level pointer var",
			code: `
package vault

type Tree struct{ size int }

var store = &Tree{}

func GetStore() *Tree { return store }
`,
			expected: 1,
		},
		{
			name: "returns address of package-level var",
			code: `
package vault

type Account struct{ balance int }

var gAccount Account

func GetAccount() *Account { return &gAccount }
`,
			expected: 1,
		},
		{
			name: "returns pointer field of package-level var",
			code: `
package vault

type Tree struct{ size int }

type Container struct{ Items *Tree }

var c = &Container{Items: &Tree{}}

func GetItems() *Tree { return c.Items }
`,
			expected: 1,
		},
		{
			name: "value copy return is fine",
			code: `
package vault

type Account struct{ balance int }

var gAccount Account

func GetBalance() int { return gAccount.balance }

func GetAccount() Account { return gAccount }
`,
			expected: 0,
		},
		{
			name: "constructor returning fresh pointer is fine",
			code: `
package vault

type Tree struct{ size int }

func NewTree() *Tree { return &Tree{} }
`,
			expected: 0,
		},
		{
			name: "unexported function is not flagged",
			code: `
package vault

type Tree struct{ size int }

var store = &Tree{}

func getStore() *Tree { return store }
`,
			expected: 0,
		},
		{
			name: "local variable pointer return is fine",
			code: `
package vault

type Tree struct{ size int }

func Build() *Tree {
	t := &Tree{}
	return t
}
`,
			expected: 0,
		},
		{
			name: "multi-result with leaking pointer position",
			code: `
package vault

type Tree struct{ size int }

var store = &Tree{}

func GetStore() (*Tree, bool) { return store, true }
`,
			expected: 1,
		},
		{
			name: "local alias of package var is caught",
			code: `
package vault

type Tree struct{ size int }

var store = &Tree{}

func GetStore() *Tree {
	s := store
	return s
}
`,
			expected: 1,
		},
		{
			name: "chained alias is caught",
			code: `
package vault

type Tree struct{ size int }

var store = &Tree{}

func GetStore() *Tree {
	s := store
	t := s
	return t
}
`,
			expected: 1,
		},
		{
			name: "named result with bare return is caught",
			code: `
package vault

type Tree struct{ size int }

var store = &Tree{}

func GetStore() (s *Tree) {
	s = store
	return
}
`,
			expected: 1,
		},
		{
			name: "nested func literal returns are ignored",
			code: `
package vault

type Tree struct{ size int }

var store = &Tree{}

func Walk() *Tree {
	f := func() *Tree { return store }
	_ = f
	return nil
}
`,
			expected: 0,
		},
	}

	t.Run("state and getter split across files", func(t *testing.T) {
		t.Parallel()

		pctx := writeGnoPackage(t, map[string]string{
			"state.gno": `
package vault

type Tree struct{ size int }

var store = &Tree{}
`,
			"api.gno": `
package vault

func GetStore() *Tree { return store }
`,
		})

		issues, err := exportedMutablePointerRule{}.CheckPackage(context.Background(), pctx)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		assert.Equal(t, "exported-mutable-pointer", issues[0].Rule)
		assert.Contains(t, issues[0].Filename, "api.gno")
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "file.gno", tc.code, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectExportedMutablePointer(newTestContext("file.gno", node, fset, []byte(tc.code)))
			require.NoError(t, err)

			assert.Equal(t, tc.expected, len(issues), "unexpected issue count")
			for _, issue := range issues {
				assert.Equal(t, "exported-mutable-pointer", issue.Rule)
				assert.NotEmpty(t, issue.Message)
			}
		})
	}
}
