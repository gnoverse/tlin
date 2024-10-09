package checker

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// PkgFuncMap maps package paths to function names and their alternatives
type PkgFuncMap map[string]map[string]string

// DeprecatedFunc represents a deprecated function
type DeprecatedFunc struct {
	Package     string
	Function    string
	Alternative string
	Start       token.Position
	End         token.Position
}

// DeprecatedFuncChecker checks for deprecated functions
type DeprecatedFuncChecker struct {
	deprecatedFuncs PkgFuncMap
}

// NewDeprecatedFuncChecker creates a new DeprecatedFuncChecker
func NewDeprecatedFuncChecker() *DeprecatedFuncChecker {
	return &DeprecatedFuncChecker{
		deprecatedFuncs: make(PkgFuncMap),
	}
}

// Register adds a deprecated function to the checker
//
// @notJoon [10/08/2024]: The deprecated functions are currently beign updated manually
// in the [internal/lints/deprecate_func.go] file. We could consider automatically recognizing
// and updating the map if the comments include the string `Deprecated:` in accordance with
// the `godoc` style.
//
// However, this approach would require additional file traversal, which I believe
// would be mostly unnecessary computation and may not be worth the effort.
//
// Therefore, currently, it seems more efficient to manually update the deprecated functions
// to only handle the deprecated items in gno's standard library.
func (d *DeprecatedFuncChecker) Register(pkgName, funcName, alternative string) {
	if _, ok := d.deprecatedFuncs[pkgName]; !ok {
		d.deprecatedFuncs[pkgName] = make(map[string]string)
	}
	d.deprecatedFuncs[pkgName][funcName] = alternative
}

// Check checks an AST node for deprecated functions
func (d *DeprecatedFuncChecker) Check(filename string, node *ast.File, fset *token.FileSet) ([]DeprecatedFunc, error) {
	packageAliases, err := d.getPackageAliases(node)
	if err != nil {
		return nil, fmt.Errorf("error getting package aliases: %w", err)
	}

	var found []DeprecatedFunc
	ast.Inspect(node, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if deprecatedFunc := d.checkCall(call, packageAliases, fset); deprecatedFunc != nil {
				found = append(found, *deprecatedFunc)
			}
		}
		return true
	})

	return found, nil
}

func (d *DeprecatedFuncChecker) getPackageAliases(node *ast.File) (map[string]string, error) {
	packageAliases := make(map[string]string)
	for _, imp := range node.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("error unquoting import path: %w", err)
		}
		name := d.getImportName(imp, path)
		packageAliases[name] = path
	}
	return packageAliases, nil
}

func (d *DeprecatedFuncChecker) getImportName(imp *ast.ImportSpec, path string) string {
	if imp.Name != nil {
		return imp.Name.Name
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func (d *DeprecatedFuncChecker) checkCall(call *ast.CallExpr, packageAliases map[string]string, fset *token.FileSet) *DeprecatedFunc {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		return d.checkSelectorExpr(fun, packageAliases, fset, call)
	case *ast.Ident:
		return d.checkIdent(fun, packageAliases, fset, call)
	}
	return nil
}

func (d *DeprecatedFuncChecker) checkSelectorExpr(fun *ast.SelectorExpr, packageAliases map[string]string, fset *token.FileSet, call *ast.CallExpr) *DeprecatedFunc {
	ident, ok := fun.X.(*ast.Ident)
	if !ok {
		return nil
	}
	pkgAlias := ident.Name
	funcName := fun.Sel.Name

	pkgPath, ok := packageAliases[pkgAlias]
	if !ok {
		return nil // Not a package alias, possibly a method call
	}

	return d.createDeprecatedFuncIfFound(pkgPath, funcName, fset, call)
}

func (d *DeprecatedFuncChecker) checkIdent(fun *ast.Ident, packageAliases map[string]string, fset *token.FileSet, call *ast.CallExpr) *DeprecatedFunc {
	funcName := fun.Name
	// Check dot-imported packages
	for alias, pkgPath := range packageAliases {
		if alias != "." {
			continue
		}
		if deprecatedFunc := d.createDeprecatedFuncIfFound(pkgPath, funcName, fset, call); deprecatedFunc != nil {
			return deprecatedFunc
		}
	}
	return nil
}

func (d *DeprecatedFuncChecker) createDeprecatedFuncIfFound(pkgPath, funcName string, fset *token.FileSet, call *ast.CallExpr) *DeprecatedFunc {
	if funcs, ok := d.deprecatedFuncs[pkgPath]; ok {
		if alt, ok := funcs[funcName]; ok {
			return &DeprecatedFunc{
				Package:     pkgPath,
				Function:    funcName,
				Alternative: alt,
				Start:       fset.Position(call.Pos()),
				End:         fset.Position(call.End()),
			}
		}
	}
	return nil
}
