package nolint

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

const nolintPrefix = "//nolint"

// Manager manages nolint scopes and checks if a position is nolinted.
type Manager struct {
	scopes map[string][]scope // filename to scopes
}

// scope represents a range in the code where nolint applies.
type scope struct {
	start token.Position
	end   token.Position
	rules map[string]struct{} // empty, null => apply to all lint rules
}

// ParseComments parses nolint comments in the given AST file and returns a nolintManager.
func ParseComments(f *ast.File, fset *token.FileSet) *Manager {
	manager := Manager{
		scopes: make(map[string][]scope, len(f.Comments)),
	}
	stmtMap := indexStatementsByLine(f, fset)
	packageLine := fset.Position(f.Package).Line

	for _, cg := range f.Comments {
		for _, comment := range cg.List {
			scope, err := parseComment(comment, f, fset, stmtMap, packageLine)
			if err != nil {
				// ignore invalid nolint comments
				continue
			}
			filename := scope.start.Filename
			manager.scopes[filename] = append(manager.scopes[filename], scope)
		}
	}
	return &manager
}

// parseComment parses a single nolint comment and determines its scope.
func parseComment(
	comment *ast.Comment,
	f *ast.File,
	fset *token.FileSet,
	stmtMap map[int]ast.Stmt,
	packageLine int,
) (scope, error) {
	var scope scope
	text := comment.Text

	if !strings.HasPrefix(text, nolintPrefix) {
		return scope, fmt.Errorf("invalid nolint comment")
	}

	prefixLen := len(nolintPrefix)
	rest := text[prefixLen:]

	if len(rest) > 0 && rest[0] != ':' {
		return scope, fmt.Errorf("invalid nolint comment format")
	}

	if len(rest) > 0 && rest[0] == ':' {
		rest = strings.TrimPrefix(rest, ":")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return scope, fmt.Errorf("invalid nolint comment: no rules specified after colon")
		}
	} else if len(rest) > 0 {
		return scope, fmt.Errorf("invalid nolint comment: expected colon after 'nolint'")
	}

	scope.rules = parseIgnoreRuleNames(rest)
	pos := fset.Position(comment.Slash)

	// check if the comment is before the package declaration
	if isBeforePackageDecl(pos.Line, packageLine) {
		scope.start = fset.Position(f.Pos())
		scope.end = fset.Position(f.End())
		return scope, nil
	}

	// check if the comment is at the end of a line (inline comment)
	if pos.Line == fset.File(comment.Slash).Line(comment.Slash) {
		// Inline comment, applies to the statement on the same line
		if stmt, exists := stmtMap[pos.Line]; exists {
			scope.start = fset.Position(stmt.Pos())
			scope.end = fset.Position(stmt.End())
			return scope, nil
		}
	}

	// check if the comment is above a statement
	nextLine := pos.Line + 1
	if stmt, exists := stmtMap[nextLine]; exists {
		scope.start = fset.Position(stmt.Pos())
		scope.end = fset.Position(stmt.End())
		return scope, nil
	}

	// check if the comment is above a function declaration
	if decl := findFunctionAfterLine(fset, f, pos.Line); decl != nil {
		funcPos := fset.Position(decl.Pos())
		if funcPos.Line == pos.Line+1 {
			scope.start = funcPos
			scope.end = fset.Position(decl.End())
			return scope, nil
		}
	}

	// Default case: apply to the line of the comment
	scope.start = pos
	scope.end = pos
	return scope, nil
}

// parseIgnoreRuleNames parses the rule list from the nolint comment more efficiently.
func parseIgnoreRuleNames(text string) map[string]struct{} {
	rulesMap := make(map[string]struct{})

	if text == "" {
		return rulesMap
	}

	rules := strings.Split(text, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule != "" {
			rulesMap[rule] = struct{}{}
		}
	}
	return rulesMap
}

// indexStatementsByLine traverses the AST once and maps each line to its corresponding statement.
func indexStatementsByLine(f *ast.File, fset *token.FileSet) map[int]ast.Stmt {
	stmtMap := make(map[int]ast.Stmt)
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if stmt, ok := n.(ast.Stmt); ok {
			line := fset.Position(stmt.Pos()).Line
			// save only the first statement of each line
			if _, exists := stmtMap[line]; !exists {
				stmtMap[line] = stmt
			}
		}
		return true
	})
	return stmtMap
}

// isBeforePackageDecl checks if a given line is before the package declaration.
func isBeforePackageDecl(line, packageLine int) bool {
	return line < packageLine
}

// findFunctionAfterLine finds the first function declaration after a given line.
func findFunctionAfterLine(fset *token.FileSet, f *ast.File, line int) *ast.FuncDecl {
	for _, decl := range f.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			funcLine := fset.Position(funcDecl.Pos()).Line
			if funcLine >= line {
				return funcDecl
			}
		}
	}
	return nil
}

// IsNolint checks if a given position and rule are nolinted.
func (m *Manager) IsNolint(pos token.Position, ruleName string) bool {
	scopes, exists := m.scopes[pos.Filename]
	if !exists {
		return false
	}
	for _, scope := range scopes {
		if pos.Line < scope.start.Line || pos.Line > scope.end.Line {
			continue
		}
		if len(scope.rules) == 0 {
			return true
		}
		if _, exists := scope.rules[ruleName]; exists {
			return true
		}
	}
	return false
}
