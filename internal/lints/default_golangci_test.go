package lints

import (
	"os"
	"path/filepath"
	"testing"

	tt "github.com/gnolang/tlin/internal/types"
)

func TestRunGolangciLint(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	code := `package main

import "fmt"

func main() {
	var unused int
	fmt.Println("Hello, World!")
}
`

	if err := os.WriteFile(testFile, []byte(code), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// temporary .golangci.yml file for testing
	golangciConfig := `linters:
  enable:
    - unused
    - typecheck
`
	golangciFile := filepath.Join(tmpDir, ".golangci.yml")
	if err := os.WriteFile(golangciFile, []byte(golangciConfig), 0644); err != nil {
		t.Fatalf("failed to create golangci.yml file: %v", err)
	}

	// change working directory to temporary directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	node, fset, err := ParseFile(testFile, nil)
	if err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	issues, err := RunGolangciLint(testFile, node, fset, tt.SeverityWarning)
	if err != nil {
		t.Fatalf("failed to run golangci-lint: %v", err)
	}

	if len(issues) == 0 {
		t.Error("golangci-lint did not find any issues")
	}

	for _, issue := range issues {
		t.Logf("issue: %+v", issue)
	}

	foundUnused := false
	for _, issue := range issues {
		if issue.Rule == "unused" || issue.Rule == "typecheck" {
			foundUnused = true
			break
		}
	}

	if !foundUnused {
		t.Error("unused variable issue was not found")
	}
}
