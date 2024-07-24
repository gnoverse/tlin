package lints

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestDetectCycle(t *testing.T) {
	src := `
package main

type A struct {
	B *B
}

type B struct {
	A *A
}

var (
	x = &y
	y = &x
)

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

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	cycle := newCycle()
	result := cycle.detectCycles(f)
	if len(result) != 6 {
		// [B A B]
		// [B A B]
		// [x y x]
		// [b a b]
		// [outer outer$anon<address> outer]
		// [outer outer]
		t.Errorf("unexpected result: %v", result)
	}
}
