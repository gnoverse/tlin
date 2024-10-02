package internal

import (
	"go/ast"
	"go/token"
	"log"
	"strings"
)

const nolintPrefix = "//nolint"

// nolintScope represents a range in the code where nolint applies.
type nolintScope struct {
	start token.Position
	end   token.Position
	rules map[string]struct{} // empty, null => apply to all lint rules
}

// nolintManager manages nolint scopes and checks if a position is nolinted.
type nolintManager struct {
	scopes map[string][]nolintScope // filename to scopes
}

// ParseNolintComments parses nolint comments in the given AST file and returns a nolintManager.
func ParseNolintComments(f *ast.File, fset *token.FileSet) *nolintManager {
	manager := nolintManager{
		scopes: make(map[string][]nolintScope, len(f.Comments)),
	}
	stmtMap := indexStatementsByLine(f, fset)
	packageLine := fset.Position(f.Package).Line

	for _, cg := range f.Comments {
		for _, comment := range cg.List {
			if !strings.HasPrefix(comment.Text, nolintPrefix) {
				continue
			}
			scope, err := parseNolintComment(comment, f, fset, stmtMap, packageLine)
			if err != nil {
				log.Printf("Failed to parse nolint comment: %v", err)
				continue
			}
			filename := scope.start.Filename
			manager.scopes[filename] = append(manager.scopes[filename], scope)
		}
	}
	return &manager
}

// parseNolintComment parses a single nolint comment and determines its scope.
func parseNolintComment(
	comment *ast.Comment,
	f *ast.File,
	fset *token.FileSet,
	stmtMap map[int]ast.Stmt,
	packageLine int,
) (nolintScope, error) {
	var scope nolintScope
	text := strings.TrimSpace(strings.TrimPrefix(comment.Text, nolintPrefix))

	// parse specific rules
	scope.rules = parseNolintRules(text)
	pos := fset.Position(comment.Slash)

	// check if the comment is before the package declaration
	if isBeforePackageDecl(pos.Line, packageLine) {
		scope.start = fset.Position(f.Pos())
		scope.end = fset.Position(f.End())
		return scope, nil
	}

	// skip the whole function if it is directly above the function declaration
	if decl := findFunctionAfterLine(fset, f, pos.Line); decl != nil {
		funcPos := fset.Position(decl.Pos())
		if funcPos.Line == pos.Line+1 {
			scope.start = funcPos
			scope.end = fset.Position(decl.End())
			return scope, nil
		}
	}

	// find statement at the same line or the next line using the pre-indexed map
	if stmt, exists := stmtMap[pos.Line]; exists {
		scope.start = fset.Position(stmt.Pos())
		scope.end = fset.Position(stmt.End())
		return scope, nil
	} else if stmt, exists := stmtMap[pos.Line+1]; exists {
		scope.start = fset.Position(stmt.Pos())
		scope.end = fset.Position(stmt.End())
		return scope, nil
	}

	// use default comment position
	scope.start = pos
	scope.end = pos
	return scope, nil
}

// parseNolintRules parses the rule list from the nolint comment more efficiently.
func parseNolintRules(text string) map[string]struct{} {
	rulesMap := make(map[string]struct{})

	// Find the index of the first colon
	colon := strings.IndexByte(text, ':')
	if colon == -1 || colon == len(text)-1 {
		return rulesMap
	}

	start := colon + 1
	n := len(text)
	for i := start; i <= n; i++ {
		// if we reach a comma or the end of the string, process the rule
		if i == n || text[i] == ',' {
			// trim leading and trailing spaces
			end := i
			for start < end && text[start] == ' ' {
				start++
			}
			for end > start && text[end-1] == ' ' {
				end--
			}
			if start < end {
				rule := text[start:end]
				rulesMap[rule] = struct{}{}
			}
			start = i + 1
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
func (m *nolintManager) IsNolint(pos token.Position, ruleName string) bool {
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
