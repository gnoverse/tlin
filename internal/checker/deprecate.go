package checker

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// TODO: refactor this codebase

// PkgFuncMap maps package paths to function names and their alternatives
type PkgFuncMap map[string]map[string]string

// PkgTypeMethodMap maps package paths to types and their methods with alternatives
type PkgTypeMethodMap map[string]map[string]map[string]string

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
	deprecatedFuncs   PkgFuncMap
	deprecatedMethods PkgTypeMethodMap
}

// NewDeprecatedFuncChecker creates a new DeprecatedFuncChecker
func NewDeprecatedFuncChecker() *DeprecatedFuncChecker {
	return &DeprecatedFuncChecker{
		deprecatedFuncs:   make(PkgFuncMap),
		deprecatedMethods: make(PkgTypeMethodMap),
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

// RegisterMethod adds a deprecated method to the checker
func (d *DeprecatedFuncChecker) RegisterMethod(pkgName, typeName, methodName, alternative string) {
	if _, ok := d.deprecatedMethods[pkgName]; !ok {
		d.deprecatedMethods[pkgName] = make(map[string]map[string]string)
	}

	if _, ok := d.deprecatedMethods[pkgName][typeName]; !ok {
		d.deprecatedMethods[pkgName][typeName] = make(map[string]string)
	}

	d.deprecatedMethods[pkgName][typeName][methodName] = alternative
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
		// This could be a method call on a non-identifier expression
		// For example: obj.Method() or expr().Method()
		return d.checkMethodCall(fun, packageAliases, fset, call)
	}

	pkgAlias := ident.Name
	methodName := fun.Sel.Name

	// Check if it's a package function call (pkg.Func())
	pkgPath, ok := packageAliases[pkgAlias]
	if ok {
		return d.createDeprecatedFuncIfFound(pkgPath, methodName, fset, call)
	}

	// It might be a method call on a variable
	return d.checkMethodCall(fun, packageAliases, fset, call)
}

func (d *DeprecatedFuncChecker) checkMethodCall(fun *ast.SelectorExpr, packageAliases map[string]string, fset *token.FileSet, call *ast.CallExpr) *DeprecatedFunc {
	methodName := fun.Sel.Name

	// Try to determine the type of the expression this method is called on
	exprType := d.inferExpressionType(fun.X, packageAliases)
	if exprType != "" {
		// Split the type into package and type parts if it contains a dot
		var pkgPath, typeName string
		if parts := strings.Split(exprType, "."); len(parts) > 1 {
			pkgPath = parts[0]
			typeName = parts[1]
		} else {
			// If no dot, assume it's a local type
			typeName = exprType
		}

		// Check if the type is in the current package or in an imported package
		for alias, importPath := range packageAliases {
			// For alias matches, use the actual import path
			if pkgPath == alias {
				pkgPath = importPath
				break
			}
		}

		// Now check if this type+method is deprecated
		if typeMap, ok := d.deprecatedMethods[pkgPath]; ok {
			if methodMap, ok := typeMap[typeName]; ok {
				if alt, ok := methodMap[methodName]; ok {
					return &DeprecatedFunc{
						Package:     pkgPath,
						Function:    fmt.Sprintf("%s.%s", typeName, methodName),
						Alternative: alt,
						Start:       fset.Position(call.Pos()),
						End:         fset.Position(call.End()),
					}
				}
			}
		}
	}

	// Fallback: check all deprecated methods across all packages and types
	// This is less precise but catches cases where we couldn't infer the type
	for pkgPath, typeMap := range d.deprecatedMethods {
		for typeName, methodMap := range typeMap {
			if alt, ok := methodMap[methodName]; ok {
				return &DeprecatedFunc{
					Package:     pkgPath,
					Function:    fmt.Sprintf("%s.%s", typeName, methodName),
					Alternative: alt,
					Start:       fset.Position(call.Pos()),
					End:         fset.Position(call.End()),
				}
			}
		}
	}

	return nil
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

// inferExpressionType attempts to determine the type of an expression
func (d *DeprecatedFuncChecker) inferExpressionType(expr ast.Expr, packageAliases map[string]string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		// Simple identifier - variable name or type name
		return e.Name

	case *ast.SelectorExpr:
		// pkg.Type or obj.Field format
		if pkg, ok := e.X.(*ast.Ident); ok {
			// If it's a package alias, convert to actual package path
			if pkgPath, exists := packageAliases[pkg.Name]; exists {
				return pkgPath + "." + e.Sel.Name
			}
			return pkg.Name + "." + e.Sel.Name
		}
		// Handle chained selectors (a.b.c format)
		baseType := d.inferExpressionType(e.X, packageAliases)
		if baseType != "" {
			return baseType + "." + e.Sel.Name
		}

	case *ast.CallExpr:
		// Function call - infer return type when possible
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if pkg, ok := sel.X.(*ast.Ident); ok {
				funcName := sel.Sel.Name
				pkgName := pkg.Name

				// Handle package aliases
				if pkgPath, exists := packageAliases[pkgName]; exists {
					pkgName = pkgPath
				}

				// Heuristic 1: NewXxx() pattern is assumed to return Xxx type
				if strings.HasPrefix(funcName, "New") {
					typeName := strings.TrimPrefix(funcName, "New")
					return pkgName + "." + typeName
				}

				// Heuristic 2: xxxFrom() pattern is assumed to return xxx type
				if strings.HasSuffix(funcName, "From") {
					typeName := strings.TrimSuffix(funcName, "From")
					return pkgName + "." + typeName
				}

				// Heuristic 3: ParseXxx() is assumed to return Xxx type
				if strings.HasPrefix(funcName, "Parse") {
					typeName := strings.TrimPrefix(funcName, "Parse")
					return pkgName + "." + typeName
				}
			}
		}

	case *ast.UnaryExpr:
		// Unary operator (*x, &x, etc.)
		if e.Op == token.MUL {
			return "*" + d.inferExpressionType(e.X, packageAliases)
		}

	case *ast.StarExpr:
		// Pointer type (*T) or dereference (*x)
		return d.inferExpressionType(e.X, packageAliases)

	case *ast.ParenExpr:
		// Parenthesized expression (x)
		return d.inferExpressionType(e.X, packageAliases)

	case *ast.TypeAssertExpr:
		// Type assertion (x.(Type))
		if ident, ok := e.Type.(*ast.Ident); ok {
			return ident.Name
		} else if sel, ok := e.Type.(*ast.SelectorExpr); ok {
			if pkg, ok := sel.X.(*ast.Ident); ok {
				if pkgPath, exists := packageAliases[pkg.Name]; exists {
					return pkgPath + "." + sel.Sel.Name
				}
				return pkg.Name + "." + sel.Sel.Name
			}
		}

	case *ast.CompositeLit:
		// Composite literal (Type{...})
		return d.inferExpressionType(e.Type, packageAliases)

	case *ast.IndexExpr:
		// Index expression (arr[idx])
		// Difficult to infer element types of maps or slices
		// Do not handle complex cases
		return ""

	case *ast.FuncLit:
		// Function literal (func(...){...})
		// Return type inference required but complex
		return ""
	}

	return ""
}
