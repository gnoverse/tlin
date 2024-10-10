package lints

import (
	"go/ast"
	"go/parser"
	"go/token"
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
	if len(result) != 4 {
		// [B A B]
		// [B A B]
		// [x y x]
		// [b a b]
		// [outer outer$anon<address> outer]
		t.Errorf("unexpected result: %v", result)
	}
}
