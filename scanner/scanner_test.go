package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectScanner(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	files := map[string]string{
		"file1.go":        "package main",
		"file2.gno":       "package test",
		"file3.txt":       "This is a text file",
		"subdir/file4.go": "package subdir",
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err)
	}

	scanner := New(tempDir, ".go", ".gno")
	scannedFiles, err := scanner.Scan()
	require.NoError(t, err)

	assert.Equal(t, 3, len(scannedFiles), "Should find 3 Go/Gno files")

	foundPaths := make(map[string]bool)
	for _, file := range scannedFiles {
		foundPaths[file.Path] = true
		assert.Greater(t, file.Size, int64(0), "File size should be greater than 0")
	}

	assert.True(t, foundPaths[filepath.Join(tempDir, "file1.go")], "Should find file1.go")
	assert.True(t, foundPaths[filepath.Join(tempDir, "file2.gno")], "Should find file2.gno")
	assert.True(t, foundPaths[filepath.Join(tempDir, "subdir/file4.go")], "Should find subdir/file4.go")
	assert.False(t, foundPaths[filepath.Join(tempDir, "file3.txt")], "Should not find file3.txt")
}
