package fixer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessImports(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		src      string
		expected string
	}{
		{
			name:     "add missing import",
			filename: "test.go",
			src: `package main

func main() {
	_ = errors.New("test")
}
`,
			expected: `package main

import "errors"

func main() {
	_ = errors.New("test")
}
`,
		},
		{
			name:     "remove unused import",
			filename: "test.go",
			src: `package main

import (
	"errors"
	"fmt"
)

func main() {
	_ = errors.New("test")
}
`,
			expected: `package main

import (
	"errors"
)

func main() {
	_ = errors.New("test")
}
`,
		},
		{
			name:     "add and remove imports simultaneously",
			filename: "test.go",
			src: `package main

import "fmt"

func main() {
	_ = errors.New("test")
}
`,
			expected: `package main

import "errors"

func main() {
	_ = errors.New("test")
}
`,
		},
		{
			name:     "gno file processed as go",
			filename: "test.gno",
			src: `package main

func main() {
	_ = errors.New("test")
}
`,
			expected: `package main

import "errors"

func main() {
	_ = errors.New("test")
}
`,
		},
		{
			name:     "preserve gno imports that are used",
			filename: "test.gno",
			src: `package main

import "gno.land/p/nt/ufmt"

func main() {
	ufmt.Sprintf("hello %s", "world")
}
`,
			expected: `package main

import "gno.land/p/nt/ufmt"

func main() {
	ufmt.Sprintf("hello %s", "world")
}
`,
		},
		{
			name:     "remove unused gno import",
			filename: "test.gno",
			src: `package main

import "gno.land/p/nt/ufmt"

func main() {
	println("hello")
}
`,
			expected: `package main

func main() {
	println("hello")
}
`,
		},
		{
			name:     "cannot auto-add non-standard library imports",
			filename: "test.gno",
			src: `package main

func main() {
	// ufmt is used but not imported - imports.Process cannot auto-add it
	// because it's not a standard library package
	ufmt.Sprintf("hello")
}
`,
			// imports.Process cannot resolve non-standard packages,
			// so the code remains unchanged (with unresolved ufmt reference)
			expected: `package main

func main() {
	// ufmt is used but not imported - imports.Process cannot auto-add it
	// because it's not a standard library package
	ufmt.Sprintf("hello")
}
`,
		},
		{
			name:     "preserve multiple mixed imports - remove only unused",
			filename: "test.gno",
			src: `package main

import (
	"errors"
	"fmt"
	"gno.land/p/nt/ufmt"
)

func main() {
	_ = errors.New("test")
	ufmt.Sprintf("hello")
}
`,
			expected: `package main

import (
	"errors"

	"gno.land/p/nt/ufmt"
)

func main() {
	_ = errors.New("test")
	ufmt.Sprintf("hello")
}
`,
		},
		{
			name:     "preserve aliased imports that are used",
			filename: "test.gno",
			src: `package main

import (
	u "gno.land/p/nt/ufmt"
)

func main() {
	u.Sprintf("hello")
}
`,
			expected: `package main

import (
	u "gno.land/p/nt/ufmt"
)

func main() {
	u.Sprintf("hello")
}
`,
		},
		{
			name:     "remove unused aliased import",
			filename: "test.gno",
			src: `package main

import (
	u "gno.land/p/nt/ufmt"
)

func main() {
	println("hello")
}
`,
			expected: `package main

func main() {
	println("hello")
}
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ProcessImports(tc.filename, []byte(tc.src))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(result))
		})
	}
}
