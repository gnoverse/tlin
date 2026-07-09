package lints

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectUnsafePreviousRealm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "unsafe.PreviousRealm inside crossing function",
			code: `
package vault

import "chain/runtime/unsafe"

func Set(cur realm, key, value string) {
	caller := unsafe.PreviousRealm().Address()
	_ = caller
}
`,
			expected: 1,
		},
		{
			name: "aliased import",
			code: `
package vault

import u "chain/runtime/unsafe"

func Set(cur realm, key string) {
	_ = u.PreviousRealm()
}
`,
			expected: 1,
		},
		{
			name: "import without direct call still flagged",
			code: `
package vault

import "chain/runtime/unsafe"

var f = unsafe.PreviousRealm

func Set(cur realm, key string) {}
`,
			expected: 1,
		},
		{
			name: "no crossing function: import allowed",
			code: `
package helper

import "chain/runtime/unsafe"

func Caller() string {
	return unsafe.PreviousRealm().Address().String()
}
`,
			expected: 0,
		},
		{
			name: "crossing function using cur.Previous is fine",
			code: `
package vault

func Set(cur realm, key, value string) {
	if !cur.IsCurrent() {
		panic("spoofed realm")
	}
	caller := cur.Previous().Address()
	_ = caller
}
`,
			expected: 0,
		},
		{
			name: "stdlib unsafe import is not confused",
			code: `
package vault

import "unsafe"

func Set(cur realm, p unsafe.Pointer) {}
`,
			expected: 0,
		},
		{
			name: "multiple calls each flagged",
			code: `
package vault

import "chain/runtime/unsafe"

func Set(cur realm) {
	_ = unsafe.PreviousRealm()
	_ = unsafe.PreviousRealm()
}
`,
			expected: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "file.gno", tc.code, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectUnsafePreviousRealm(newTestContext("file.gno", node, fset, []byte(tc.code)))
			require.NoError(t, err)

			assert.Equal(t, tc.expected, len(issues), "unexpected issue count")
			for _, issue := range issues {
				assert.Equal(t, "unsafe-previous-realm", issue.Rule)
				assert.NotEmpty(t, issue.Message)
			}
		})
	}
}
