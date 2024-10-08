package checker

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterDeprecatedFunctions(t *testing.T) {
	t.Parallel()
	checker := NewDeprecatedFuncChecker()

	checker.Register("fmt", "Println", "fmt.Print")
	checker.Register("os", "Remove", "os.RemoveAll")

	expected := PkgFuncMap{
		"fmt": {"Println": "fmt.Print"},
		"os":  {"Remove": "os.RemoveAll"},
	}

	assert.Equal(t, expected, checker.deprecatedFuncs)
}

func TestCheck(t *testing.T) {
	t.Parallel()
	src := `
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	os.Remove("some_file.txt")
}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "example.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	checker := NewDeprecatedFuncChecker()
	checker.Register("fmt", "Println", "fmt.Print")
	checker.Register("os", "Remove", "os.RemoveAll")

	deprecated, err := checker.Check("example.go", node, fset)
	if err != nil {
		t.Fatalf("Check failed with error: %v", err)
	}

	expected := []DeprecatedFunc{
		{
			Package:     "fmt",
			Function:    "Println",
			Alternative: "fmt.Print",
			Start: token.Position{
				Filename: "example.go",
				Offset:   55,
				Line:     10,
				Column:   2,
			},
			End: token.Position{
				Filename: "example.go",
				Offset:   83,
				Line:     10,
				Column:   30,
			},
		},
		{
			Package:     "os",
			Function:    "Remove",
			Alternative: "os.RemoveAll",
			Start: token.Position{
				Filename: "example.go",
				Offset:   85,
				Line:     11,
				Column:   2,
			},
			End: token.Position{
				Filename: "example.go",
				Offset:   111,
				Line:     11,
				Column:   28,
			},
		},
	}

	assert.Equal(t, expected, deprecated)
}

func TestCheckNoDeprecated(t *testing.T) {
	t.Parallel()
	src := `
package main

import "fmt"

func main() {
	fmt.Printf("Hello, %s\n", "World")
}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "example.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	checker := NewDeprecatedFuncChecker()
	checker.Register("fmt", "Println", "fmt.Print")
	checker.Register("os", "Remove", "os.RemoveAll")

	deprecated, err := checker.Check("example.go", node, fset)
	if err != nil {
		t.Fatalf("Check failed with error: %v", err)
	}

	assert.Equal(t, 0, len(deprecated))
}

func TestCheckMultipleDeprecatedCalls(t *testing.T) {
	t.Parallel()
	src := `
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello")
	fmt.Println("World")
	os.Remove("file1.txt")
	os.Remove("file2.txt")
}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "example.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	checker := NewDeprecatedFuncChecker()
	checker.Register("fmt", "Println", "fmt.Print")
	checker.Register("os", "Remove", "os.RemoveAll")

	deprecated, err := checker.Check("example.go", node, fset)
	if err != nil {
		t.Fatalf("Check failed with error: %v", err)
	}

	expected := []DeprecatedFunc{
		{Package: "fmt", Function: "Println", Alternative: "fmt.Print"},
		{Package: "fmt", Function: "Println", Alternative: "fmt.Print"},
		{Package: "os", Function: "Remove", Alternative: "os.RemoveAll"},
		{Package: "os", Function: "Remove", Alternative: "os.RemoveAll"},
	}

	assert.Equal(t, len(expected), len(deprecated))
	for i, exp := range expected {
		assertDeprecatedFuncEqual(t, exp, deprecated[i])
	}
}

func TestDeprecatedFuncCheckerWithAlias(t *testing.T) {
	t.Parallel()

	c := NewDeprecatedFuncChecker()
	c.Register("math", "Sqrt", "math.Pow")

	const src = `
package main

import (
	m "math"
	"fmt"
)

type MyStruct struct{}

func (s *MyStruct) Method() {}

func main() {
	result := m.Sqrt(42)
	_ = result

	fmt.Println("Hello")

	s := &MyStruct{}
	s.Method()
}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "sample.go", src, parser.ParseComments)
	assert.NoError(t, err)

	results, err := c.Check("sample.go", node, fset)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(results))

	expected := DeprecatedFunc{
		Package:     "math",
		Function:    "Sqrt",
		Alternative: "math.Pow",
	}

	assertDeprecatedFuncEqual(t, expected, results[0])
}

func TestDeprecatedFuncChecker_Check_DotImport(t *testing.T) {
	t.Parallel()

	checker := NewDeprecatedFuncChecker()
	checker.Register("fmt", "Println", "Use fmt.Print instead")

	src := `
package main

import . "fmt"

func main() {
	Println("Hello, World!")
}
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	assert.NoError(t, err)

	found, err := checker.Check("test.go", f, fset)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(found))

	if len(found) > 0 {
		df := found[0]
		if df.Package != "fmt" || df.Function != "Println" || df.Alternative != "Use fmt.Print instead" {
			t.Errorf("unexpected deprecated function info: %+v", df)
		}
	}
}

func assertDeprecatedFuncEqual(t *testing.T, expected, actual DeprecatedFunc) {
	t.Helper()
	assert.Equal(t, expected.Package, actual.Package)
	assert.Equal(t, expected.Function, actual.Function)
	assert.Equal(t, expected.Alternative, actual.Alternative)
	assert.NotEmpty(t, actual.Start.Filename)
	assert.Greater(t, actual.Start.Offset, 0)
	assert.Greater(t, actual.Start.Line, 0)
	assert.Greater(t, actual.Start.Column, 0)
}
