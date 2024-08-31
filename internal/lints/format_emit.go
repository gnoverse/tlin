package lints

import (
	"go/ast"
	"go/token"
	"strings"

	tt "github.com/gnoswap-labs/tlin/internal/types"
)

func DetectEmitFormat(filename string, node *ast.File, fset *token.FileSet) ([]tt.Issue, error) {
	var issues []tt.Issue
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
			if x, ok := fun.X.(*ast.Ident); ok && x.Name == "std" && fun.Sel.Name == "Emit" {
				if len(call.Args) > 3 && !isEmitCorrectlyFormatted(call, fset) {
					issue := tt.Issue{
						Rule:       "emit-format",
						Filename:   filename,
						Start:      fset.Position(call.Pos()),
						End:        fset.Position(call.End()),
						Message:    "Consider formatting std.Emit call for better readability",
						Suggestion: formatEmitCall(call),
						Confidence: 1.0,
					}
					issues = append(issues, issue)
				}
			}
		}

		return true
	})

	return issues, nil
}

func isEmitCorrectlyFormatted(call *ast.CallExpr, fset *token.FileSet) bool {
	if len(call.Args) < 2 {
		return true
	}

	// check if the call is multi-line
	firstArgLine := fset.Position(call.Args[0].Pos()).Line
	lastArgLine := fset.Position(call.Args[len(call.Args)-1].End()).Line
	if firstArgLine == lastArgLine {
		return false
	}

	// check if each pair of arguments is on its own line
	for i := 1; i < len(call.Args); i += 2 {
		if i+1 < len(call.Args) {
			keyLine := fset.Position(call.Args[i].Pos()).Line
			valueLine := fset.Position(call.Args[i+1].Pos()).Line
			if keyLine != valueLine {
				return false
			}
		}
	}

	return true
}

func formatEmitCall(call *ast.CallExpr) string {
	var sb strings.Builder
	sb.WriteString("std.Emit(\n")

	// event type
	if len(call.Args) > 0 {
		sb.WriteString("    ")
		sb.WriteString(formatArg(call.Args[0]))
		sb.WriteString(",\n")
	}

	// key-value pairs
	for i := 1; i < len(call.Args); i += 2 {
		sb.WriteString("    ")
		sb.WriteString(formatArg(call.Args[i]))
		sb.WriteString(", ")
		if i+1 < len(call.Args) {
			sb.WriteString(formatArg(call.Args[i+1]))
		}
		sb.WriteString(",\n")
	}

	sb.WriteString(")")
	return sb.String()
}

func formatArg(arg ast.Expr) string {
	switch v := arg.(type) {
	case *ast.BasicLit:
		return v.Value
	case *ast.Ident:
		return v.Name
	case *ast.CallExpr:
		return formatCallExpr(v)
	default:
		return "..."
	}
}

func formatCallExpr(call *ast.CallExpr) string {
	var sb strings.Builder
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		sb.WriteString(sel.X.(*ast.Ident).Name)
		sb.WriteString(".")
		sb.WriteString(sel.Sel.Name)
	} else if ident, ok := call.Fun.(*ast.Ident); ok {
		sb.WriteString(ident.Name)
	}
	sb.WriteString("(")
	for i, arg := range call.Args {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(formatArg(arg))
	}
	sb.WriteString(")")
	return sb.String()
}
