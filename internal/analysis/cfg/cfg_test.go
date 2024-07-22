package cfg

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestFromStmts(t *testing.T) {
	src := `
		package main
		func main() {
			x := 1
			if x > 0 {
				x = 2
			} else {
				x = 3
			}
			for i := 0; i < 10; i++ {
				x += i
			}
		}
	`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "src.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	var funcDecl *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, isFn := decl.(*ast.FuncDecl); isFn {
			funcDecl = fn
			break
		}
	}

	if funcDecl == nil {
		t.Fatal("No function declaration found")
	}

	cfgGraph := FromFunc(funcDecl)

	if cfgGraph.Entry == nil {
		t.Errorf("Expected Entry node, got nil")
	}
	if cfgGraph.Exit == nil {
		t.Errorf("Expected Exit node, got nil")
	}

	blocks := cfgGraph.Blocks()
	if len(blocks) == 0 {
		t.Errorf("Expected some blocks, got none")
	}

	for _, block := range blocks {
		preds := cfgGraph.Preds(block)
		succs := cfgGraph.Succs(block)
		t.Logf("Block: %v, Preds: %v, Succs: %v", block, preds, succs)
	}
}

func TestCFG(t *testing.T) {
	tests := []struct {
		name           string
		src            string
		expectedBlocks int
	}{
		{
			name: "MultiStatementFunction",
			src: `
				package main
				func main() {
					x := 1
					if x > 0 {
						x = 2
					} else {
						x = 3
					}
					for i := 0; i < 10; i++ {
						x += i
					}
				}`,
			expectedBlocks: 10,
		},
		{
			name: "Switch",
			src: `
				package main
				func withSwitch(day string) int {
					switch day {
					case "Monday":
						return 1
					case "Tuesday":
						return 2
					case "Wednesday":
						fallthrough
					case "Thursday":
						return 3
					case "Friday":
						break
					default:
						return 0
					}
				}`,
			expectedBlocks: 15,
		},
		{
			name: "TypeSwitch",
			src: `
				package main
				type MyType int
				func withTypeSwitch(i interface{}) int {
					switch i.(type) {
					case int:
						return 1
					case MyType:
						return 2
					default:
						return 0
					}
					return 0
				}`,
			expectedBlocks: 11,
		},
		{
			name: "EmptyFunc",
			src: `
				package main
				func empty() {}`,
			expectedBlocks: 2,
		},
		{
			name: "SingleStatementFunc",
			src: `
				package main
				func single() {
					x := 1
				}`,
			expectedBlocks: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "src.go", tt.src, 0)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			var funcDecl *ast.FuncDecl
			for _, decl := range node.Decls {
				if fn, isFn := decl.(*ast.FuncDecl); isFn {
					funcDecl = fn
					break
				}
			}

			if funcDecl == nil {
				t.Fatal("No function declaration found")
			}

			cfgGraph := FromFunc(funcDecl)

			if cfgGraph.Entry == nil {
				t.Error("Expected Entry node, got nil")
			}
			if cfgGraph.Exit == nil {
				t.Error("Expected Exit node, got nil")
			}

			blocks := cfgGraph.Blocks()
			if len(blocks) != tt.expectedBlocks {
				t.Errorf("Expected %d blocks, got %d", tt.expectedBlocks, len(blocks))
			}

			// Additional checks can be added here if needed
		})
	}
}

func TestPrintDot(t *testing.T) {
	src := `
package main
func main() {
	x := 1
	if x > 0 {
		x = 2
	} else {
		x = 3
	}
	for i := 0; i < 10; i++ {
		x += i
	}
}`

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "src.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}

	var funcDecl *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, isFn := decl.(*ast.FuncDecl); isFn {
			funcDecl = fn
			break
		}
	}

	if funcDecl == nil {
		t.Fatal("No function declaration found")
	}

	cfgGraph := FromFunc(funcDecl)

	var buf bytes.Buffer
	cfgGraph.PrintDot(&buf, fset, func(n ast.Stmt) string { return "" })

	output := buf.String()
	expected := `
digraph mgraph {
	mode="heir";
	splines="ortho";

	"ENTRY" -> "assignment - line 4"
	"assignment - line 4" -> "if statement - line 5"
	"if statement - line 5" -> "assignment - line 6"
	"if statement - line 5" -> "assignment - line 8"
	"assignment - line 6" -> "assignment - line 10"
	"assignment - line 8" -> "assignment - line 10"
	"for loop - line 10" -> "EXIT"
	"for loop - line 10" -> "assignment - line 11"
	"assignment - line 10" -> "for loop - line 10"
	"increment statement - line 10" -> "for loop - line 10"
	"assignment - line 11" -> "increment statement - line 10"
}
`

	if normalizeDotOutput(output) != normalizeDotOutput(expected) {
		t.Errorf("Expected DOT output:\n%s\nGot:\n%s", expected, output)
	}
}

func normalizeDotOutput(dot string) string {
	lines := strings.Split(dot, "\n")
	var normalized []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return strings.Join(normalized, "\n")
}
