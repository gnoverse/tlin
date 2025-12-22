package lints

import (
	"go/ast"
	"go/token"
	"os"
	"strconv"
	"strings"
	"unicode"

	tt "github.com/gnolang/tlin/internal/types"
)

type formatFuncInfo struct {
	formatArgIndex int
	suggestion     string
}

var formatFunctions = map[string]formatFuncInfo{
	"Sprintf": {formatArgIndex: 0, suggestion: "use a string literal directly"},
	"Printf":  {formatArgIndex: 0, suggestion: "use print() instead"},
	"Errorf":  {formatArgIndex: 0, suggestion: "use errors.New() instead"},
	"Fprintf": {formatArgIndex: 1, suggestion: "use io.WriteString() instead"},
}

// DetectFormatWithoutVerb reports formatting calls whose format string has no verbs.
// It targets ufmt (always) and fmt (only in *_test files).
func DetectFormatWithoutVerb(
	filename string,
	node *ast.File,
	fset *token.FileSet,
	severity tt.Severity,
) ([]tt.Issue, error) {
	aliasMap := BuildImportAliasMap(node)
	allowPaths := map[string]bool{
		"gno.land/p/nt/ufmt": true,
	}
	if isTestFile(filename) {
		allowPaths["fmt"] = true
	}

	if !hasAllowedImports(aliasMap, allowPaths) {
		return nil, nil
	}

	src, _ := os.ReadFile(filename)

	constants := collectStringConstants(node)

	issues := make([]tt.Issue, 0)
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		path, ok := aliasMap[pkgIdent.Name]
		if !ok || !allowPaths[path] {
			return true
		}

		info, ok := formatFunctions[sel.Sel.Name]
		if !ok {
			return true
		}

		if len(call.Args) <= info.formatArgIndex {
			return true
		}

		formatVal, ok := stringLiteralValue(call.Args[info.formatArgIndex], constants)
		if !ok {
			return true
		}

		if containsVerb(formatVal) {
			return true
		}

		funcName := sel.Sel.Name
		message := "format string has no verbs; " + info.suggestion

		startPos := fset.Position(call.Pos())
		endPos := fset.Position(call.End())

		// Build suggestion by replacing the call expression in the original line
		suggestion := buildLineSuggestion(src, startPos, endPos, funcName, formatVal)

		issues = append(issues, tt.Issue{
			Rule:       "format-without-verb",
			Filename:   filename,
			Start:      startPos,
			End:        endPos,
			Message:    message,
			Suggestion: suggestion,
			Severity:   severity,
		})

		return true
	})

	return issues, nil
}

func hasAllowedImports(aliasMap map[string]string, allowPaths map[string]bool) bool {
	for _, path := range aliasMap {
		if allowPaths[path] {
			return true
		}
	}
	return false
}

func collectStringConstants(file *ast.File) map[string]string {
	constants := make(map[string]string)

	ast.Inspect(file, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.CONST {
			return true
		}

		for _, spec := range decl.Specs {
			valSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			switch len(valSpec.Values) {
			case 0:
				continue
			case 1:
				if val, ok := constStringValue(valSpec.Values[0]); ok {
					for _, name := range valSpec.Names {
						constants[name.Name] = val
					}
				}
			default:
				for i, name := range valSpec.Names {
					if i >= len(valSpec.Values) {
						break
					}
					if val, ok := constStringValue(valSpec.Values[i]); ok {
						constants[name.Name] = val
					}
				}
			}
		}

		return true
	})

	return constants
}

func constStringValue(expr ast.Expr) (string, bool) {
	basic, ok := expr.(*ast.BasicLit)
	if !ok || basic.Kind != token.STRING {
		return "", false
	}

	val, err := strconv.Unquote(basic.Value)
	if err != nil {
		return "", false
	}

	return val, true
}

func stringLiteralValue(expr ast.Expr, constants map[string]string) (string, bool) {
	if val, ok := constStringValue(expr); ok {
		return val, true
	}

	if ident, ok := expr.(*ast.Ident); ok {
		if val, ok := constants[ident.Name]; ok {
			return val, true
		}
	}

	return "", false
}

func containsVerb(format string) bool {
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}

		i++
		if i >= len(format) {
			break
		}

		if format[i] == '%' {
			continue
		}

		for i < len(format) && strings.ContainsRune("+-#0 ", rune(format[i])) {
			i++
		}

		if i < len(format) && format[i] == '[' {
			i++
			for i < len(format) && unicode.IsDigit(rune(format[i])) {
				i++
			}
			if i < len(format) && format[i] == ']' {
				i++
			}
		}

		if i < len(format) && (format[i] == '*' || unicode.IsDigit(rune(format[i]))) {
			if format[i] == '*' {
				i++
			} else {
				for i < len(format) && unicode.IsDigit(rune(format[i])) {
					i++
				}
			}
		}

		if i < len(format) && format[i] == '.' {
			i++
			if i < len(format) && (format[i] == '*' || unicode.IsDigit(rune(format[i]))) {
				if format[i] == '*' {
					i++
				} else {
					for i < len(format) && unicode.IsDigit(rune(format[i])) {
						i++
					}
				}
			}
		}

		if i >= len(format) {
			break
		}

		if strings.IndexByte("vTtbcdoOqxXUeEfFgGsqp", format[i]) >= 0 {
			return true
		}
	}

	return false
}

func isTestFile(filename string) bool {
	return strings.HasSuffix(filename, "_test.go") || strings.HasSuffix(filename, "_test.gno")
}

func buildLineSuggestion(src []byte, startPos, endPos token.Position, funcName, formatVal string) string {
	if len(src) == 0 {
		return buildCallReplacement(funcName, formatVal)
	}

	lines := strings.Split(string(src), "\n")

	// Handle single-line case
	if startPos.Line == endPos.Line {
		lineIdx := startPos.Line - 1
		if lineIdx < 0 || lineIdx >= len(lines) {
			return buildCallReplacement(funcName, formatVal)
		}
		line := lines[lineIdx]

		// Column is 1-based
		startCol := startPos.Column - 1
		endCol := endPos.Column - 1

		if startCol < 0 || endCol > len(line) || startCol >= endCol {
			return buildCallReplacement(funcName, formatVal)
		}

		replacement := buildCallReplacement(funcName, formatVal)
		return line[:startCol] + replacement + line[endCol:]
	}

	// Multi-line call: just return the replacement for the call
	return buildCallReplacement(funcName, formatVal)
}

func buildCallReplacement(funcName, formatVal string) string {
	quotedVal := strconv.Quote(formatVal)
	switch funcName {
	case "Sprintf":
		return quotedVal
	case "Errorf":
		return "errors.New(" + quotedVal + ")"
	case "Printf":
		return "print(" + quotedVal + ")"
	case "Fprintf":
		return "io.WriteString(w, " + quotedVal + ")"
	default:
		return quotedVal
	}
}
