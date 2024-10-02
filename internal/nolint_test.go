package internal

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestParseNolintRules(t *testing.T) {
	input := "//nolint:rule1,rule2,rule3"
	expected := map[string]struct{}{
		"rule1": {},
		"rule2": {},
		"rule3": {},
	}
	result := parseNolintRules(input)
	if len(result) != len(expected) {
		t.Errorf("Expected %d rules, got %d", len(expected), len(result))
	}
	for k := range expected {
		if _, exists := result[k]; !exists {
			t.Errorf("Expected rule %s not found", k)
		}
	}
}

func TestParseNolintComments(t *testing.T) {
	t.Parallel()
	src := `package main

//nolint:rule1,rule2
func foo() {
	// some code
}

//nolint
var x int
`

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	manager := ParseNolintComments(f, fset)
	if manager == nil {
		t.Fatal("Expected nolintManager, got nil")
	}

	pos := token.Position{Filename: "test.go", Line: 5, Column: 1}
	if !manager.IsNolint(pos, "rule1") {
		t.Errorf("Expected position to be nolinted for rule1")
	}

	pos = token.Position{Filename: "test.go", Line: 8, Column: 1}
	if !manager.IsNolint(pos, "anyrule") {
		t.Errorf("Expected position to be nolinted for any rule when no specific rules are set")
	}
}

func TestIsNolint(t *testing.T) {
	t.Parallel()
	source := `package main

func main() {
	//nolint
	fmt.Println("Line 5")
	fmt.Println("Line 6")
	fmt.Println("Line 7") //nolint:rule1
	//nolint:rule2
	fmt.Println("Line 9")
}
`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	manager := ParseNolintComments(node, fset)

	tests := []struct {
		line     int
		rule     string
		expected bool
	}{
		{5, "anyrule", true},  // Line 5 is covered by nolint without rules
		{6, "anyrule", false}, // Line 6 is not covered
		{7, "rule1", true},    // Line 7 is covered by nolint:rule1
		{9, "rule2", true},    // Line 9 is covered by nolint:rule2
		{9, "rule3", false},   // Line 9 is not covered for rule3
	}

	for _, test := range tests {
		pos := positionAtLine(test.line)
		result := manager.IsNolint(pos, test.rule)
		if result != test.expected {
			t.Errorf("IsNolint at line %d for rule '%s': expected %v, got %v", test.line, test.rule, test.expected, result)
		}
	}
}

// Helper function to get token.Position at a specific line
func positionAtLine(line int) token.Position {
	return token.Position{
		Filename: "test.go",
		Line:     line,
		Column:   1,
	}
}
