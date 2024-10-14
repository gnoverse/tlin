package lints

import (
	"fmt"
	"go/ast"
	"go/token"

	tt "github.com/gnolang/tlin/internal/types"
)

func DetectCycle(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]tt.Issue, error) {
	c := newCycle()
	cycles := c.detectCycles(node)

	issues := make([]tt.Issue, 0, len(cycles))
	for _, cycle := range cycles {
		issue := tt.Issue{
			Rule:     "cycle-detection",
			Filename: filename,
			Start:    fset.Position(node.Pos()),
			End:      fset.Position(node.End()),
			Message:  "Detected cycle in function call: " + cycle,
			Severity: severity,
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

type cycle struct {
	dependencies map[string][]string
	visited      map[string]bool
	stack        []string
	cycles       []string
}

func newCycle() *cycle {
	return &cycle{
		dependencies: make(map[string][]string),
		visited:      make(map[string]bool),
	}
}

func (c *cycle) analyzeFuncDecl(fn *ast.FuncDecl) {
	name := fn.Name.Name
	c.dependencies[name] = []string{}

	// ignore bodyless function
	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				if ident, ok := x.Fun.(*ast.Ident); ok {
					if ident.Name == name {
						c.dependencies[name] = append(c.dependencies[name], ident.Name)
					}
				}
			case *ast.FuncLit:
				c.analyzeFuncLit(x, name)
			}
			return true
		})
	}
}

func (c *cycle) analyzeFuncLit(fn *ast.FuncLit, parentName string) {
	anonName := fmt.Sprintf("%s$anon%p", parentName, fn)
	c.dependencies[anonName] = []string{}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if ident, ok := x.Fun.(*ast.Ident); ok {
				c.dependencies[anonName] = append(c.dependencies[anonName], ident.Name)
			}
		case *ast.FuncLit:
			c.analyzeFuncLit(x, anonName)
		}
		return true
	})

	// add dependency from parent to anonymous function
	c.dependencies[parentName] = append(c.dependencies[parentName], anonName)
}

func (c *cycle) detectCycles(node ast.Node) []string {
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			c.analyzeFuncDecl(x)
		case *ast.TypeSpec:
			c.analyzeTypeSpec(x)
		case *ast.ValueSpec:
			c.analyzeValueSpec(x)
		case *ast.FuncLit:
			// handle top-level anonymous functions
			c.analyzeFuncLit(x, "topLevel")
		}
		return true
	})

	for name := range c.dependencies {
		if !c.visited[name] {
			c.dfs(name)
		}
	}

	return c.cycles
}

func (c *cycle) analyzeTypeSpec(ts *ast.TypeSpec) {
	name := ts.Name.Name
	c.dependencies[name] = []string{}

	ast.Inspect(ts.Type, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			c.dependencies[name] = append(c.dependencies[name], ident.Name)
		}
		return true
	})
}

func (c *cycle) analyzeValueSpec(vs *ast.ValueSpec) {
	for i, name := range vs.Names {
		c.dependencies[name.Name] = []string{}
		if vs.Values != nil && i < len(vs.Values) {
			ast.Inspect(vs.Values[i], func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok {
					c.dependencies[name.Name] = append(c.dependencies[name.Name], ident.Name)
				}
				return true
			})
		}
	}
}

func (c *cycle) dfs(name string) {
	c.visited[name] = true
	c.stack = append(c.stack, name)

	for _, dep := range c.dependencies[name] {
		if !c.visited[dep] {
			c.dfs(dep)
		} else if contains(c.stack, dep) && dep != name {
			cycle := append(c.stack[indexOf(c.stack, dep):], dep)
			res := fmt.Sprintf("%v", cycle)
			c.cycles = append(c.cycles, res)
		}
	}

	c.stack = c.stack[:len(c.stack)-1]
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
