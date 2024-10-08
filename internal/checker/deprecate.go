package checker

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// pkgPath -> funcName -> alternative
type deprecatedFuncMap map[string]map[string]string

// DeprecatedFunc represents a deprecated function.
type DeprecatedFunc struct {
	Package     string
	Function    string
	Alternative string
	Position    token.Position
}

// DeprecatedFuncChecker checks for deprecated functions.
type DeprecatedFuncChecker struct {
	deprecatedFuncs deprecatedFuncMap
}

func NewDeprecatedFuncChecker() *DeprecatedFuncChecker {
	return &DeprecatedFuncChecker{
		deprecatedFuncs: make(deprecatedFuncMap),
	}
}

func (d *DeprecatedFuncChecker) Register(pkgName, funcName, alternative string) {
	if _, ok := d.deprecatedFuncs[pkgName]; !ok {
		d.deprecatedFuncs[pkgName] = make(map[string]string)
	}
	d.deprecatedFuncs[pkgName][funcName] = alternative
}

// Check checks a AST node for deprecated functions.
//
// TODO: use this in the linter rule implementation
func (d *DeprecatedFuncChecker) Check(
	filename string,
	node *ast.File,
	fset *token.FileSet,
) ([]DeprecatedFunc, error) {
	var found []DeprecatedFunc

	packageAliases := make(map[string]string)
	for _, imp := range node.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		name := ""
		if imp.Name != nil {
			name = imp.Name.Name
		} else {
			parts := strings.Split(path, "/")
			name = parts[len(parts)-1]
		}
		packageAliases[name] = path
	}

	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			ident, ok := fun.X.(*ast.Ident)
			if !ok {
				return true
			}
			pkgAlias := ident.Name
			funcName := fun.Sel.Name

			pkgPath, ok := packageAliases[pkgAlias]
			if !ok {
				// Not a package alias, possibly a method call
				return true
			}

			if funcs, ok := d.deprecatedFuncs[pkgPath]; ok {
				if alt, ok := funcs[funcName]; ok {
					found = append(found, DeprecatedFunc{
						Package:     pkgPath,
						Function:    funcName,
						Alternative: alt,
						Position:    fset.Position(call.Pos()),
					})
				}
			}
		case *ast.Ident:
			// Handle functions imported via dot imports
			funcName := fun.Name
			// Check dot-imported packages
			for alias, pkgPath := range packageAliases {
				if alias != "." {
					continue
				}
				if funcs, ok := d.deprecatedFuncs[pkgPath]; ok {
					if alt, ok := funcs[funcName]; ok {
						found = append(found, DeprecatedFunc{
							Package:     pkgPath,
							Function:    funcName,
							Alternative: alt,
							Position:    fset.Position(call.Pos()),
						})
						break
					}
				}
			}
		}
		return true
	})

	return found, nil
}
