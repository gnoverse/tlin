package nolint

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestParseNolintRules(t *testing.T) {
	t.Parallel()
	input := "rule1,rule2,rule3"
	expected := []string{"rule1", "rule2", "rule3"}
	result := parseIgnoreRuleNames(input)
	if len(result) != len(expected) {
		t.Errorf("Expected %d rules, got %d", len(expected), len(result))
	}
	for _, rule := range expected {
		if _, exists := result[rule]; !exists {
			t.Errorf("Expected rule %s not found", rule)
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

	manager := ParseComments(f, fset)
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

	manager := ParseComments(node, fset)

	tests := []struct {
		rule     string
		line     int
		expected bool
	}{
		{"anyrule", 5, true},  // Line 5 is covered by nolint without rules
		{"anyrule", 6, false}, // Line 6 is not covered
		{"rule1", 7, true},    // Line 7 is covered by nolint:rule1
		{"rule2", 9, true},    // Line 9 is covered by nolint:rule2
		{"rule3", 9, false},   // Line 9 is not covered for rule3
	}

	for _, test := range tests {
		pos := positionAtLine(test.line)
		result := manager.IsNolint(pos, test.rule)
		if result != test.expected {
			t.Errorf("IsNolint at line %d for rule '%s': expected %v, got %v", test.line, test.rule, test.expected, result)
		}
	}
}

func positionAtLine(line int) token.Position {
	return token.Position{
		Filename: "test.go",
		Line:     line,
		Column:   1,
	}
}
