package lints

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentGuard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		code     string
		expected int
	}{
		{
			name:     "IsUser guard next to OriginSend",
			filename: "file.gno",
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
			name:     "IsUserCall guard is correct",
			filename: "file.gno",
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
			name:     "IsUser without OriginSend in package is not flagged",
			filename: "file.gno",
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
			name:     "IsUser in sibling function of OriginSend user",
			filename: "file.gno",
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
		{
			name:     "plain go file is not checked",
			filename: "file.go",
			code: `
package payments

func Process(u user, b banker) {
	if u.IsUser() {
		b.OriginSend()
	}
}
`,
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pctx := writeGnoPackage(t, map[string]string{tc.filename: tc.code})
			issues, err := paymentGuardRule{}.CheckPackage(context.Background(), pctx)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, len(issues), "unexpected issue count")
			for _, issue := range issues {
				assert.Equal(t, "payment-guard", issue.Rule)
				assert.NotEmpty(t, issue.Message)
			}
		})
	}

	t.Run("guard and OriginSend split across files", func(t *testing.T) {
		t.Parallel()

		pctx := writeGnoPackage(t, map[string]string{
			"guards.gno": `
package vault

func guard(cur realm) {
	if !cur.Previous().IsUser() {
		panic("not a user")
	}
}
`,
			"deposit.gno": `
package vault

func Deposit(cur realm) {
	guard(cur)
	_ = getBanker().OriginSend()
}

func getBanker() banker { return banker{} }

type banker struct{}

func (banker) OriginSend() int { return 0 }
`,
		})

		issues, err := paymentGuardRule{}.CheckPackage(context.Background(), pctx)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		assert.Equal(t, "payment-guard", issues[0].Rule)
		assert.Contains(t, issues[0].Filename, "guards.gno")
	})

	t.Run("different packages in one dir are not conflated", func(t *testing.T) {
		t.Parallel()

		pctx := writeGnoPackage(t, map[string]string{
			"guards.gno": `
package profile

func Show(cur realm) bool { return cur.Previous().IsUser() }
`,
			"deposit.gno": `
package vault

func Deposit(b banker) { _ = b.OriginSend() }

type banker struct{}

func (banker) OriginSend() int { return 0 }
`,
		})

		issues, err := paymentGuardRule{}.CheckPackage(context.Background(), pctx)
		require.NoError(t, err)
		assert.Empty(t, issues)
	})
}
