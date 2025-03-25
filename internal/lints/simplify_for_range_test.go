package lints

import (
	"go/parser"
	"go/token"
	"testing"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/require"
)

func TestSimplifyForRange(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		wantIssue bool
	}{
		{
			name: "trivial for loop",
			code: `
package main

func main() {
	for i := 0; i < 5; i++ {
		println(i)
	}
}`,
			wantIssue: true,
		},
		{
			name: "already using range",
			code: `
package main

func main() {
	for i := range 5 {
		println(i)
	}
}`,
			wantIssue: false,
		},
		{
			name: "different form of for loop",
			code: `
package main

func main() {
	for i := 0; i <= 5; i += 2 {
		println(i)
	}
}`,
			wantIssue: false,
		},
		{
			name: "variable has been modified inside the loop",
			code: `
package main

func main() {
	for i := 0; i < 5; i++ {
		if i > 3 {
			i++
		}
		println(i)
	}
}`,
			wantIssue: false,
		},
		{
			name: "initial value is not 0",
			code: `
package main

func main() {
	for i := 1; i < 5; i++ {
		println(i)
	}
}`,
			wantIssue: false,
		},
		{
			name: "multiple variables",
			code: `
package main

func main() {
	for i, j := 0, 5; i < j; i, j = i+1, j-1 {
		println(i, j)
	}
}`,
			wantIssue: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", test.code, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectSimplifiableForLoops("test.go", file, fset, tt.SeverityWarning)
			require.NoError(t, err)

			if test.wantIssue {
				require.Len(t, issues, 1)
				require.Equal(t, "counter-based for loop can be simplified to range-based loop", issues[0].Message)
			} else {
				require.Empty(t, issues)
			}
		})
	}
}
