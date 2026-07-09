package lints

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectPaymentGuard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "IsUser guard next to OriginSend",
			code: `
package vault

func Deposit(cur realm) {
	if !cur.Previous().IsUser() {
		panic("not a user")
	}
	banker := getBanker()
	coins := banker.OriginSend()
	_ = coins
}

func getBanker() banker { return banker{} }

type banker struct{}

func (banker) OriginSend() int { return 0 }
`,
			expected: 1,
		},
		{
			name: "IsUserCall guard is correct",
			code: `
package vault

func Deposit(cur realm) {
	if !cur.Previous().IsUserCall() {
		panic("not a direct user call")
	}
	coins := getBanker().OriginSend()
	_ = coins
}

func getBanker() banker { return banker{} }

type banker struct{}

func (banker) OriginSend() int { return 0 }
`,
			expected: 0,
		},
		{
			name: "IsUser without OriginSend in file is not flagged",
			code: `
package profile

func Show(cur realm) string {
	if cur.Previous().IsUser() {
		return "user"
	}
	return "contract"
}
`,
			expected: 0,
		},
		{
			name: "IsUser in sibling function of OriginSend user",
			code: `
package vault

func guard(cur realm) {
	if !cur.Previous().IsUser() {
		panic("not a user")
	}
}

func Deposit(cur realm) {
	guard(cur)
	_ = getBanker().OriginSend()
}

func getBanker() banker { return banker{} }

type banker struct{}

func (banker) OriginSend() int { return 0 }
`,
			expected: 1,
		},
	}

	t.Run("plain go file is not checked", func(t *testing.T) {
		t.Parallel()

		code := `
package payments

func Process(u user, b banker) {
	if u.IsUser() {
		b.OriginSend()
	}
}
`
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, "file.go", code, parser.ParseComments)
		require.NoError(t, err)

		issues, err := DetectPaymentGuard(newTestContext("file.go", node, fset, []byte(code)))
		require.NoError(t, err)
		assert.Empty(t, issues)
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "file.gno", tc.code, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectPaymentGuard(newTestContext("file.gno", node, fset, []byte(tc.code)))
			require.NoError(t, err)

			assert.Equal(t, tc.expected, len(issues), "unexpected issue count")
			for _, issue := range issues {
				assert.Equal(t, "payment-guard", issue.Rule)
				assert.NotEmpty(t, issue.Message)
			}
		})
	}
}
