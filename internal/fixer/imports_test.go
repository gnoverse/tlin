package fixer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureImports(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		imports  []string
		expected string
	}{
		{
			name: "add single import to file without imports",
			src: `package main

func main() {
	_ = errors.New("test")
}
`,
			imports: []string{"errors"},
			expected: `package main

import "errors"

func main() {
	_ = errors.New("test")
}
`,
		},
		{
			name: "add import to file with existing imports",
			src: `package main

import "fmt"

func main() {
	fmt.Println(errors.New("test"))
}
`,
			imports: []string{"errors"},
			expected: `package main

import (
	"errors"
	"fmt"
)

func main() {
	fmt.Println(errors.New("test"))
}
`,
		},
		{
			name: "skip already existing import",
			src: `package main

import "errors"

func main() {
	_ = errors.New("test")
}
`,
			imports: []string{"errors"},
			expected: `package main

import "errors"

func main() {
	_ = errors.New("test")
}
`,
		},
		{
			name: "add multiple imports",
			src: `package main

func main() {
	io.WriteString(os.Stdout, "test")
}
`,
			imports: []string{"io", "os"},
			expected: `package main

import (
	"io"
	"os"
)

func main() {
	io.WriteString(os.Stdout, "test")
}
`,
		},
		{
			name: "no imports to add",
			src: `package main

func main() {
}
`,
			imports:  []string{},
			expected: `package main

func main() {
}
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := EnsureImports([]byte(tc.src), tc.imports)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(result))
		})
	}
}

func TestHasImport(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		importPath string
		expected   bool
	}{
		{
			name: "has import",
			src: `package main

import "errors"

func main() {}
`,
			importPath: "errors",
			expected:   true,
		},
		{
			name: "does not have import",
			src: `package main

import "fmt"

func main() {}
`,
			importPath: "errors",
			expected:   false,
		},
		{
			name: "has import in group",
			src: `package main

import (
	"errors"
	"fmt"
)

func main() {}
`,
			importPath: "errors",
			expected:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fset, file := parseSource(t, tc.src)
			_ = fset
			result := hasImport(file, tc.importPath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCollectRequiredImports(t *testing.T) {
	tests := []struct {
		name     string
		issues   []tt.Issue
		expected []string
	}{
		{
			name: "collect from single issue",
			issues: []tt.Issue{
				{RequiredImports: []string{"errors"}},
			},
			expected: []string{"errors"},
		},
		{
			name: "collect from multiple issues with duplicates",
			issues: []tt.Issue{
				{RequiredImports: []string{"errors"}},
				{RequiredImports: []string{"errors", "fmt"}},
				{RequiredImports: []string{"io"}},
			},
			expected: []string{"errors", "fmt", "io"},
		},
		{
			name: "empty issues",
			issues: []tt.Issue{
				{RequiredImports: nil},
				{RequiredImports: []string{}},
			},
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := CollectRequiredImports(tc.issues)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func parseSource(t *testing.T, src string) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	require.NoError(t, err)
	return fset, file
}
