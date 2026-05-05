package lints

import (
	"fmt"
	"go/ast"
	"slices"

	"github.com/gnolang/tlin/internal/rule"
	tt "github.com/gnolang/tlin/internal/types"
)

func init() {
	rule.Register(cycleDetectionRule{})
}

type cycleDetectionRule struct{}

func (cycleDetectionRule) Name() string                 { return "cycle-detection" }
func (cycleDetectionRule) DefaultSeverity() tt.Severity { return tt.SeverityError }

func (cycleDetectionRule) Check(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	return DetectCycle(ctx)
}

func DetectCycle(ctx *rule.AnalysisContext) ([]tt.Issue, error) {
	c := newCycle()
	cycles := c.detectCycles(ctx.File)

	issues := make([]tt.Issue, 0, len(cycles))
	for _, cycle := range cycles {
		issue := ctx.NewIssue("cycle-detection", ctx.File.Pos(), ctx.File.End())
		issue.Message = "detected cycle in function call: " + cycle
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

// receiverInfo carries an enclosing method's receiver type and param
// name so SelectorExpr callees of the form `r.method()` can be
// resolved to "TypeName.method". Empty fields mean "no receiver" or
// "anonymous receiver" — in either case selector callees won't be
// resolved without go/types.
type receiverInfo struct {
	typeName  string
	paramName string
}

func (c *cycle) analyzeFuncDecl(fn *ast.FuncDecl) {
	name, recv := funcDeclNode(fn)
	c.dependencies[name] = []string{}
	if fn.Body == nil {
		return
	}
	c.collectCallees(fn.Body, name, recv)
}

func (c *cycle) analyzeFuncLit(fn *ast.FuncLit, parentName string, recv receiverInfo) {
	anonName := fmt.Sprintf("%s$anon%p", parentName, fn)
	c.dependencies[anonName] = []string{}
	c.collectCallees(fn.Body, anonName, recv)
	c.dependencies[parentName] = append(c.dependencies[parentName], anonName)
}

func (c *cycle) collectCallees(body *ast.BlockStmt, owner string, recv receiverInfo) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if callee := resolveCallee(x.Fun, recv); callee != "" {
				c.dependencies[owner] = append(c.dependencies[owner], callee)
			}
		case *ast.FuncLit:
			c.analyzeFuncLit(x, owner, recv)
			return false
		}
		return true
	})
}

func resolveCallee(fn ast.Expr, recv receiverInfo) string {
	switch e := fn.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if recv.paramName == "" || recv.typeName == "" {
			return ""
		}
		x, ok := e.X.(*ast.Ident)
		if !ok || x.Name != recv.paramName {
			return ""
		}
		return recv.typeName + "." + e.Sel.Name
	}
	return ""
}

func funcDeclNode(fn *ast.FuncDecl) (string, receiverInfo) {
	name := fn.Name.Name
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return name, receiverInfo{}
	}
	typeName := receiverTypeName(fn.Recv.List[0].Type)
	if typeName == "" {
		return name, receiverInfo{}
	}
	paramName := ""
	if names := fn.Recv.List[0].Names; len(names) > 0 {
		paramName = names[0].Name
	}
	return typeName + "." + name, receiverInfo{typeName: typeName, paramName: paramName}
}

// receiverTypeName extracts T from receiver type expressions like
// T, *T, T[U], *T[U, V]. Returns "" for unsupported shapes.
func receiverTypeName(expr ast.Expr) string {
	for {
		switch e := expr.(type) {
		case *ast.StarExpr:
			expr = e.X
		case *ast.IndexExpr:
			expr = e.X
		case *ast.IndexListExpr:
			expr = e.X
		case *ast.Ident:
			return e.Name
		default:
			return ""
		}
	}
}

func (c *cycle) detectCycles(node ast.Node) []string {
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			c.analyzeFuncDecl(x)
			return false
		case *ast.FuncLit:
			c.analyzeFuncLit(x, "topLevel", receiverInfo{})
			return false
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

func (c *cycle) dfs(name string) {
	c.visited[name] = true
	c.stack = append(c.stack, name)

	for _, dep := range c.dependencies[name] {
		if !c.visited[dep] {
			c.dfs(dep)
		} else if dep != name {
			if i := slices.Index(c.stack, dep); i >= 0 {
				cycle := append(c.stack[i:], dep)
				c.cycles = append(c.cycles, fmt.Sprintf("%v", cycle))
			}
		}
	}

	c.stack = c.stack[:len(c.stack)-1]
}
