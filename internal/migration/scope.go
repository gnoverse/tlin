package migration

import (
	"go/ast"
)

type Capability struct {
	Name string
	Kind string
}

func parentMap(file *ast.File) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) > 0 {
			parents[n] = stack[len(stack)-1]
		}
		stack = append(stack, n)
		return true
	})
	return parents
}

func ResolveCapability(node ast.Node, parents map[ast.Node]ast.Node) (Capability, bool) {
	fn := enclosingFunc(node, parents)
	if fn == nil {
		return Capability{}, false
	}
	switch x := fn.(type) {
	case *ast.FuncDecl:
		return capabilityFromFuncType(x.Type)
	case *ast.FuncLit:
		return capabilityFromFuncType(x.Type)
	default:
		return Capability{}, false
	}
}

func enclosingFunc(node ast.Node, parents map[ast.Node]ast.Node) ast.Node {
	for cur := node; cur != nil; cur = parents[cur] {
		switch cur.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return cur
		}
	}
	return nil
}

func capabilityFromFuncType(t *ast.FuncType) (Capability, bool) {
	if t == nil || t.Params == nil {
		return Capability{}, false
	}
	if cap, ok := crossingCapability(t.Params.List); ok {
		return cap, true
	}
	if cap, ok := pHelperCapability(t.Params.List); ok {
		return cap, true
	}
	return Capability{}, false
}

func crossingCapability(fields []*ast.Field) (Capability, bool) {
	if len(fields) == 0 {
		return Capability{}, false
	}
	first := fields[0]
	if len(first.Names) != 1 {
		return Capability{}, false
	}
	if !isIdentType(first.Type, "realm") {
		return Capability{}, false
	}
	return Capability{Name: first.Names[0].Name, Kind: "crossing"}, true
}

func pHelperCapability(fields []*ast.Field) (Capability, bool) {
	if len(fields) < 2 {
		return Capability{}, false
	}
	first := fields[0]
	second := fields[1]
	if len(first.Names) != 1 || first.Names[0].Name != "_" || !isIdentType(first.Type, "int") {
		return Capability{}, false
	}
	if len(second.Names) != 1 || !isIdentType(second.Type, "realm") {
		return Capability{}, false
	}
	return Capability{Name: second.Names[0].Name, Kind: "p-helper"}, true
}

func isIdentType(expr ast.Expr, name string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == name
}

func receiverFuncBody(fn ast.Node) *ast.BlockStmt {
	switch x := fn.(type) {
	case *ast.FuncDecl:
		return x.Body
	case *ast.FuncLit:
		return x.Body
	default:
		return nil
	}
}

func knownTellerVars(fn ast.Node) map[string]bool {
	body := receiverFuncBody(fn)
	vars := map[string]bool{}
	if body == nil {
		return vars
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case nil:
			return true
		case *ast.FuncLit:
			return false
		case *ast.AssignStmt:
			for i, lhs := range x.Lhs {
				if i >= len(x.Rhs) || !knownTellerExpr(x.Rhs[i], vars) {
					continue
				}
				if id, ok := lhs.(*ast.Ident); ok {
					vars[id.Name] = true
				}
			}
		case *ast.ValueSpec:
			typeIsTeller := isTellerType(x.Type)
			for i, name := range x.Names {
				if typeIsTeller || (i < len(x.Values) && knownTellerExpr(x.Values[i], vars)) {
					vars[name.Name] = true
				}
			}
		}
		return true
	})
	return vars
}

func knownTellerExpr(expr ast.Expr, vars map[string]bool) bool {
	switch x := expr.(type) {
	case *ast.Ident:
		return vars[x.Name]
	case *ast.CallExpr:
		switch fun := x.Fun.(type) {
		case *ast.Ident:
			return knownTellerFactory(fun.Name)
		case *ast.SelectorExpr:
			return knownTellerFactory(fun.Sel.Name)
		}
	}
	return false
}

func isTellerType(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Teller"
}

func knownTellerFactory(name string) bool {
	switch name {
	case "CallerTeller", "RealmTeller", "RealmSubTeller", "GetTokenTeller":
		return true
	default:
		return false
	}
}

func firstArgIsCrossingMarker(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	switch arg := call.Args[0].(type) {
	case *ast.Ident:
		return arg.Name == "cross"
	case *ast.CallExpr:
		id, ok := arg.Fun.(*ast.Ident)
		return ok && id.Name == "cross"
	default:
		return false
	}
}
