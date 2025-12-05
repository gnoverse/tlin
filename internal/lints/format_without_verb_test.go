package lints

import (
	"path/filepath"
	"testing"

	"go/parser"
	"go/token"

	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectFormatWithoutVerb(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		code     string
		expected int
	}{
		{
			name:     "ufmt sprintf literal",
			filename: "file.gno",
			code: `
package main

import "gno.land/p/nt/ufmt"

func main() {
	_ = ufmt.Sprintf("literal")
}
`,
			expected: 1,
		},
		{
			name:     "ufmt sprintf literal with args",
			filename: "file.gno",
			code: `
package main

import "gno.land/p/nt/ufmt"

func main() {
	_ = ufmt.Sprintf("literal", 1)
}
`,
			expected: 1,
		},
		{
			name:     "ufmt printf literal",
			filename: "file.gno",
			code: `
package main

import "gno.land/p/nt/ufmt"

func main() {
	ufmt.Printf("status ok")
}
`,
			expected: 1,
		},
		{
			name:     "ufmt fprintf literal",
			filename: "file.gno",
			code: `
package main

import (
	"gno.land/p/nt/ufmt"
)

func main() {
	var w any
	ufmt.Fprintf(w, "ready")
}
`,
			expected: 1,
		},
		{
			name:     "fmt sprintf allowed in test files",
			filename: "file_test.gno",
			code: `
package main

import "fmt"

func main() {
	_ = fmt.Sprintf("literal")
}
`,
			expected: 1,
		},
		{
			name:     "respects verb presence",
			filename: "file.gno",
			code: `
package main

import "gno.land/p/nt/ufmt"

func main() {
	_ = ufmt.Sprintf("value: %s", "x")
}
`,
			expected: 0,
		},
		{
			name:     "skips non-constant format",
			filename: "file.gno",
			code: `
package main

import "gno.land/p/nt/ufmt"

func main() {
	msg := "status ok"
	_ = ufmt.Sprintf(msg, "x")
}
`,
			expected: 0,
		},
		{
			name:     "handles alias imports",
			filename: "file.gno",
			code: `
package main

import u "gno.land/p/nt/ufmt"

func main() {
	_ = u.Sprintf("literal")
}
`,
			expected: 1,
		},
		{
			name:     "detects percent escapes only",
			filename: "file.gno",
			code: `
package main

import "gno.land/p/nt/ufmt"

func main() {
	var w any
	ufmt.Fprintf(w, "percent: %%d")
}
`,
			expected: 1,
		},
		{
			name:     "uses string constants",
			filename: "file.gno",
			code: `
package main

import "gno.land/p/nt/ufmt"

const msg = "literal"

func main() {
	_ = ufmt.Sprintf(msg)
}
`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, filepath.ToSlash(tt.filename), tt.code, parser.ParseComments)
			require.NoError(t, err)

			issues, err := DetectFormatWithoutVerb(tt.filename, node, fset, types.SeverityWarning)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, len(issues), "unexpected issue count")
			for _, issue := range issues {
				assert.Equal(t, "format-without-verb", issue.Rule)
				assert.NotEmpty(t, issue.Message)
			}
		})
	}
}
