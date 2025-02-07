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
	// scopes maps filename to a slice of nolint scopes.
	scopes map[string][]nolintScope
}

// nolintScope represents a range in the code where nolint applies.
type nolintScope struct {
	rules map[string]struct{}
	start token.Position
	end   token.Position
}

// ParseComments parses nolint comments in the given AST file and returns a Manager.
func ParseComments(f *ast.File, fset *token.FileSet) *Manager {
	manager := Manager{
		scopes: make(map[string][]nolintScope, len(f.Comments)),
	}
	stmtMap := indexStatementsByLine(f, fset)
	packageLine := fset.Position(f.Package).Line

	for _, cg := range f.Comments {
		for _, comment := range cg.List {
			ns, err := parseComment(comment, f, fset, stmtMap, packageLine)
			if err != nil {
				// ignore invalid nolint comments
				continue
			}
			filename := ns.start.Filename
			manager.scopes[filename] = append(manager.scopes[filename], ns)
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
) (nolintScope, error) {
	var ns nolintScope
	text := comment.Text

	if !strings.HasPrefix(text, nolintPrefix) {
		return ns, fmt.Errorf("invalid nolint comment")
	}

	prefixLen := len(nolintPrefix)
	rest := text[prefixLen:]

	// A nolint comment can either have a list of rules after a colon (:)
	// or if no rules are specified, it applies to all rules
	if len(rest) > 0 && rest[0] != ':' {
		return ns, fmt.Errorf("invalid nolint comment format")
	}

	if len(rest) > 0 && rest[0] == ':' {
		rest = strings.TrimPrefix(rest, ":")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return ns, fmt.Errorf("invalid nolint comment: no rules specified after colon")
		}
	}
	ns.rules = parseIgnoreRuleNames(rest)
	pos := fset.Position(comment.Slash)

	// If the comment appears before the package declaration, apply it to the entire file
	if isBeforePackageDecl(pos.Line, packageLine) {
		ns.start = fset.Position(f.Pos())
		ns.end = fset.Position(f.End())
		return ns, nil
	}

	// Check if the comment is inline (appears after code on the same line)
	if isInlineComment(fset, comment, stmtMap) {
		if stmt, exists := stmtMap[pos.Line]; exists {
			// For inline comments, apply to the scope of the current statement
			ns.start = fset.Position(stmt.Pos())
			ns.end = fset.Position(stmt.End())
			return ns, nil
		}
	}

	// For standalone comments: if there's a statement on the next line,
	// apply to that statement's scope while including the comment line itself
	nextLine := pos.Line + 1
	if stmt, exists := stmtMap[nextLine]; exists {
		ns.start = pos // Apply from the comment line
		ns.end = fset.Position(stmt.End())
		return ns, nil
	}

	// If no immediate statement follows, look for a function declaration to apply to
	if decl := findFunctionAfterLine(fset, f, pos.Line); decl != nil {
		funcPos := fset.Position(decl.Pos())
		if funcPos.Line == pos.Line+1 {
			ns.start = pos
			ns.end = fset.Position(decl.End())
			return ns, nil
		}
	}

	// default behavior:
	// apply only to the comment line
	ns.start = pos
	ns.end = pos
	return ns, nil
}

// parseIgnoreRuleNames parses the rule list from the nolint comment.
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
// If multiple statements exist on a single line, only the first statement is recorded.
func indexStatementsByLine(f *ast.File, fset *token.FileSet) map[int]ast.Stmt {
	stmtMap := make(map[int]ast.Stmt)
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if stmt, ok := n.(ast.Stmt); ok {
			line := fset.Position(stmt.Pos()).Line
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

// isInlineComment determines if a comment is inline with a statement.
// The comment is considered inline if it appears on the same line as a statement
// and its file offset is greater than the statement's starting offset.
func isInlineComment(fset *token.FileSet, comment *ast.Comment, stmtMap map[int]ast.Stmt) bool {
	pos := fset.Position(comment.Slash)
	if stmt, exists := stmtMap[pos.Line]; exists {
		stmtPos := fset.Position(stmt.Pos())
		return pos.Offset > stmtPos.Offset
	}
	return false
}

// IsNolint checks if a given position and rule are nolinted.
func (m *Manager) IsNolint(pos token.Position, ruleName string) bool {
	scopes, exists := m.scopes[pos.Filename]
	if !exists {
		return false
	}
	for _, ns := range scopes {
		if pos.Line < ns.start.Line || pos.Line > ns.end.Line {
			continue
		}
		// If the rules list is empty, nolint applies to all rules
		if len(ns.rules) == 0 {
			return true
		}
		if _, exists := ns.rules[ruleName]; exists {
			return true
		}
	}
	return false
}
