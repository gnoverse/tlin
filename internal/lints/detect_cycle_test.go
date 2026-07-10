package lints

import (
	"go/ast"
	"testing"
)

func TestAnalyzeFuncDeclWithBodylessFunction(t *testing.T) {
	c := newCycle()

	bodylessFunc := &ast.FuncDecl{
		Name: &ast.Ident{Name: "bodylessFunction"},
		Body: nil,
	}

	c.analyzeFuncDecl(bodylessFunc)

	if len(c.dependencies["bodylessFunction"]) != 0 {
		t.Errorf("there should be no dependency on bodyless function. got: %v", c.dependencies["bodylessFunction"])
	}
	if _, exists := c.dependencies["bodylessFunction"]; !exists {
		t.Error("bodyless function should be added to dependency map")
	}
}

func TestDetectCycle(t *testing.T) {
	t.Parallel()
	src := `
package main

func a() {
	b()
}

func b() {
	a()
}

func outer() {
	var inner func()
	inner = func() {
		outer()
	}
	inner()
}`

	cycles := detectCyclesInSource(t, src)
	// Expected: [a b a] and [outer outer$anon... outer].
	if len(cycles) != 2 {
		t.Errorf("expected 2 cycles, got %d: %v", len(cycles), cycles)
	}
}

// TestDetectCycle_NoFalsePositives covers patterns that the rule used
// to flag (type-level cycles via indirection, pointer init cycles)
// but that are valid Go and must NOT be reported. The gno-ibc Field/
// Schema reproducer is the lead case.
func TestDetectCycle_NoFalsePositives(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "gno-ibc Field/Schema reproducer",
			src: `package main
type Type int
type Field struct {
	Type Type
	Sub  *Schema
	Elem *Field
}
type Schema struct {
	Fields []Field
}`,
		},
		{
			name: "self-pointer linked list",
			src: `package main
type Node struct { Next *Node }`,
		},
		{
			name: "recursive map values",
			src: `package main
type Tree struct { Children map[string]*Tree }`,
		},
		{
			name: "self-typed interface method",
			src: `package main
type I interface { F() I }`,
		},
		{
			name: "function-typed field",
			src: `package main
type Lazy struct { Eval func() Lazy }`,
		},
		{
			name: "embedded mutual",
			src: `package main
type Outer struct { *Inner }
type Inner struct { *Outer }`,
		},
		{
			name: "pointer var cycle",
			src: `package main
var x = &y
var y = &x`,
		},
		{
			name: "slice of pointer to self",
			src: `package main
type T []*T`,
		},
		{
			name: "type alias chain",
			src: `package main
type A = B
type B = int`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cycles := detectCyclesInSource(t, tc.src)
			if len(cycles) != 0 {
				t.Errorf("expected no cycles, got %d: %v", len(cycles), cycles)
			}
		})
	}
}

// TestDetectCycle_MutualFunctions guards the latent-bug fix: before
// the fix, analyzeFuncDecl filtered call-site idents to self-only,
// so `a()` calling `b()` and back never registered.
func TestDetectCycle_MutualFunctions(t *testing.T) {
	t.Parallel()
	src := `package main
func a() { b() }
func b() { a() }`
	cycles := detectCyclesInSource(t, src)
	if len(cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %d: %v", len(cycles), cycles)
	}
}

// TestDetectCycle_Methods covers SelectorExpr-based recursion.
func TestDetectCycle_Methods(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		src       string
		wantCount int
	}{
		{
			name: "same-type mutual via pointer receiver",
			src: `package main
type T struct{}
func (r *T) foo() { r.bar() }
func (r *T) bar() { r.foo() }`,
			wantCount: 1,
		},
		{
			name: "same-type three-method chain",
			src: `package main
type T struct{}
func (r *T) a() { r.b() }
func (r *T) b() { r.c() }
func (r *T) c() { r.a() }`,
			wantCount: 1,
		},
		{
			// Direct self-recursion is not flagged — matches the
			// existing free-function behavior.
			name: "same-type self-recursion via receiver",
			src: `package main
type T struct{}
func (r *T) foo() { r.foo() }`,
			wantCount: 0,
		},
		{
			name: "cross-type method recursion (known limitation)",
			src: `package main
type A struct{ b *B }
type B struct{ a *A }
func (a *A) f() { a.b.g() }
func (b *B) g() { b.a.f() }`,
			// Resolving b.a.f() requires go/types; out of scope.
			wantCount: 0,
		},
		{
			name: "anonymous receiver (known limitation)",
			src: `package main
type T struct{}
func (T) foo() { /* can't resolve calls without param name */ }
func (T) bar() {}`,
			wantCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cycles := detectCyclesInSource(t, tc.src)
			if len(cycles) != tc.wantCount {
				t.Errorf("expected %d cycles, got %d: %v", tc.wantCount, len(cycles), cycles)
			}
		})
	}
}

func detectCyclesInSource(t *testing.T, src string) []string {
	t.Helper()
	f, _, err := ParseFile("", []byte(src))
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}
	return newCycle().detectCycles(f)
}
