package branch

import "go/ast"

type Call struct {
	Pkg string // package name
	Name string // function name
}

// DeviatingFuncs lists known control flow deviating function calls.
var DeviatingFuncs = map[Call]BranchKind{
	{"os", "Exit"}:     Exit,
	{"log", "Fatal"}:   Exit,
	{"log", "Fatalf"}:  Exit,
	{"log", "Fatalln"}: Exit,
	{"", "panic"}:      Panic,
	{"log", "Panic"}:   Panic,
	{"log", "Panicf"}:  Panic,
	{"log", "Panicln"}: Panic,
}

// ExprCall gets the call of an ExprStmt.
func ExprCall(expr *ast.ExprStmt) (Call, bool) {
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return Call{}, false
	}

	switch v := call.Fun.(type) {
	case *ast.Ident:
		return Call{Name: v.Name}, true
	case *ast.SelectorExpr:
		if ident, ok := v.X.(*ast.Ident); ok {
			return Call{Pkg: ident.Name, Name: v.Sel.Name}, true
		}
	}

	return Call{}, false
}
