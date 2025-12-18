package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossFileIssueFilenames tests that issues pointing to other files are not remapped
func TestCrossFileIssueFilenames(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "crossfile_test")

	// Create two .go files that might have cross-file issues
	file1 := filepath.Join(tempDir, "file1.go")
	content1 := `package main

import "fmt"

func Function1() {
	fmt.Println("File 1")
}
`
	err := os.WriteFile(file1, []byte(content1), 0o644)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir, "file2.go")
	content2 := `package main

import "fmt"

func Function2() {
	fmt.Println("File 2")
	x := 5
	y := int(x) // unnecessary type conversion
	_ = y
}
`
	err = os.WriteFile(file2, []byte(content2), 0o644)
	require.NoError(t, err)

	engine, err := NewEngine(tempDir, nil, nil)
	require.NoError(t, err)

	// Run engine on file2
	issues, err := engine.Run(file2)
	assert.NoError(t, err)

	// All issues from file2 should still point to file2
	for _, issue := range issues {
		assert.Equal(t, file2, issue.Filename, "Issues should not be remapped for regular .go files")
	}
}

// TestGnoTempFileRemapping tests that only issues pointing to temp files are remapped
func TestGnoTempFileRemapping(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "gno_temp_test")

	// Create a .gno file
	gnoFile := filepath.Join(tempDir, "test.gno")
	content := `package main

func test() {
	x := 5
	y := int(x) // unnecessary type conversion
	_ = y
}
`
	err := os.WriteFile(gnoFile, []byte(content), 0o644)
	require.NoError(t, err)

	engine, err := NewEngine(tempDir, nil, nil)
	require.NoError(t, err)

	// Run the engine
	issues, err := engine.Run(gnoFile)
	assert.NoError(t, err)

	// All issues should point to the original .gno file
	for _, issue := range issues {
		assert.Equal(t, gnoFile, issue.Filename, "Issues from temp files should be remapped to original .gno file")
		assert.Contains(t, issue.Filename, ".gno", "Remapped filename should be the .gno file")
	}
}
