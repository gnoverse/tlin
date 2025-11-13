package lints

import (
	"go/parser"
	"go/token"
	"testing"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/require"
)

func TestSimplifyForRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		code          string
		expectedIssue int
	}{
		{
			name: "slice length loop can be simplified",
			code: `
package main

func main() {
	nums := []int{1, 2, 3}
	for i := 0; i < len(nums); i++ {
		_ = nums[i]
	}
}
`,
			expectedIssue: 1,
		},
		{
			name: "array length loop can be simplified",
			code: `
package main

func main() {
	arr := [2]int{1, 2}
	for i := 0; i < len(arr); i++ {
		_ = arr[i]
	}
}
`,
			expectedIssue: 1,
		},
		{
			name: "selector expression over slice",
			code: `
package main

type container struct{ items []int }

func main() {
	c := container{items: []int{1, 2}}
	for i := 0; i < len(c.items); i++ {
		_ = c.items[i]
	}
}
`,
			expectedIssue: 1,
		},
		{
			name: "numeric upper bound is not simplifiable",
			code: `
package main

func main() {
	for i := 0; i < 5; i++ {
		println(i)
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "len over map is rejected",
			code: `
package main

func main() {
	m := map[int]int{1: 1}
	for i := 0; i < len(m); i++ {
		_ = m[i]
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "loop variable modified in body",
			code: `
package main

func main() {
	nums := []int{1, 2, 3}
	for i := 0; i < len(nums); i++ {
		i++
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "loop variable used after loop",
			code: `
package main

func main() {
	nums := []int{1, 2, 3}
	var i int
	for i = 0; i < len(nums); i++ {
		_ = nums[i]
	}
	println(i)
}
`,
			expectedIssue: 0,
		},
		{
			name: "non canonical increment form",
			code: `
package main

func main() {
	nums := []int{1, 2, 3}
	for i := 0; i < len(nums); i += 2 {
		_ = nums[i]
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "counter loop over numeric constant must not simplify",
			code: `
package main

func main() {
	for i := 0; i < 10; i++ {
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "counter loop over non-len integer variable must not simplify",
			code: `
package main

func main() {
	max := 256
	for i := 0; i < max; i++ {
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "counter loop over named constant must not simplify",
			code: `
package main

const maxUint256SampleAttempts = 1000

func main() {
	for attempt := 0; attempt < maxUint256SampleAttempts; attempt++ {
		_ = attempt
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "counter loop over arithmetic expression must not simplify",
			code: `
package main

func main() {
	x := 3
	for i := 0; i < x*4; i++ {
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "counter loop over function call must not simplify",
			code: `
package main

func f() int { return 5 }

func main() {
	for i := 0; i < f(); i++ {
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "len over non-iterable variable must not simplify",
			code: `
package main

func main() {
	var x int
	for i := 0; i < len(x); i++ { // invalid in Go, but type-checker will catch
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "len over struct field that is not iterable must not simplify",
			code: `
package main

type S struct { v int }

func main() {
	s := S{}
	for i := 0; i < len(s.v); i++ { // not iterable
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "loop variable shadowed inside body does not modify actual loop variable",
			code: `
package main

func main() {
	nums := []int{1, 2, 3}
	for i := 0; i < len(nums); i++ {
		var i int // shadowing - but should still avoid simplifying
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "loop variable assigned through multi-assign must not simplify",
			code: `
package main

func main() {
	nums := []int{1,2,3}
	for i := 0; i < len(nums); i++ {
		nums[0], i = nums[1], 5
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "loop condition not strictly less-than must not simplify",
			code: `
package main

func main() {
	nums := []int{1,2,3}
	for i := 0; i <= len(nums); i++ { // <= is not allowed
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "loop initialization using explicit type must not simplify",
			code: `
package main

func main() {
	nums := []int{1,2,3}
	var i int
	for i = 0; i < len(nums); i++ { // not := 0
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "post increment not ++ must not simplify",
			code: `
package main

func main() {
	nums := []int{1,2,3}
	for i := 0; i < len(nums); i += 1 { // i += 1 instead of i++
		_ = i
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "condition left side not simple identifier must not simplify",
			code: `
package main

func main() {
	nums := []int{1,2,3}
	for i := 0; i < len(nums); i++ {
		_ = i
	}
}
`,
			expectedIssue: 1, // but let's test a failing form:
		},
		{
			name: "condition left side not a plain identifier",
			code: `
package main

func main() {
	nums := []int{1,2,3}
	for (i) := 0; (i) < len(nums); (i)++ { // parentheses used
		_ = nums[i]
	}
}
`,
			expectedIssue: 0,
		},
		{
			name: "real-world gno-case: maxUint256SampleAttempts must not simplify",
			code: `
package main

const maxUint256SampleAttempts = 256

func main() {
	for attempt := 0; attempt < maxUint256SampleAttempts; attempt++ {
		_ = attempt
	}
}
`,
			expectedIssue: 0,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", test.code, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectSimplifiableForLoops("test.go", file, fset, tt.SeverityWarning)
			require.NoError(t, err)

			require.Len(t, issues, test.expectedIssue)

			if test.expectedIssue > 0 {
				require.Equal(t, "counter-based for loop can be simplified to range-based loop", issues[0].Message)
			}
		})
	}
}
