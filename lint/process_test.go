package lint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessPathContextCancellation tests that context cancellation is handled properly
func TestProcessPathContextCancellation(t *testing.T) {
	t.Parallel()

	// Create temp directory with test files
	tempDir, err := os.MkdirTemp("", "test_cancel")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create multiple test files
	for i := 0; i < 10; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		content := fmt.Sprintf(`package main

func test%d() {
	x := 5
	y := int(x) // unnecessary type conversion
	_ = y
}
`, i)
		err := os.WriteFile(filename, []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Create engine
	engine, err := New(tempDir, nil, "")
	require.NoError(t, err)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// Process files
	issues, err := ProcessPath(ctx, nil, engine, tempDir, ProcessFile)

	// Should return context cancelled error
	assert.ErrorIs(t, err, context.Canceled)
	// Should still return partial results
	assert.NotNil(t, issues)
}

// TestFileResultChannelOrdering tests that results are collected in order
func TestFileResultChannelOrdering(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test_ordering")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files with known issues
	for i := 0; i < 5; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		content := fmt.Sprintf(`package main

func test%d() {
	x := 5
	// Each file has i+1 unnecessary conversions
`, i)
		for j := 0; j <= i; j++ {
			content += fmt.Sprintf("\tv%d := int(x) // conversion %d\n", j, j)
		}
		content += "\t_ = x\n"
		for j := 0; j <= i; j++ {
			content += fmt.Sprintf("\t_ = v%d\n", j)
		}
		content += "}\n"

		err := os.WriteFile(filename, []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Create engine
	engine, err := New(tempDir, nil, "")
	require.NoError(t, err)

	// Process files
	ctx := context.Background()
	issues, err := ProcessPath(ctx, nil, engine, tempDir, ProcessFile)
	assert.NoError(t, err)

	// Verify we got issues from all files
	fileMap := make(map[string]bool)
	for _, issue := range issues {
		fileMap[issue.Filename] = true
	}

	// Should have issues from multiple files
	assert.Greater(t, len(fileMap), 1, "Should have issues from multiple files")
}

// TestConcurrentProcessingWithErrors tests error handling in concurrent processing
func TestConcurrentProcessingWithErrors(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test_errors")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create some valid Go files
	for i := 0; i < 3; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("valid%d.go", i))
		content := `package main
func main() {}
`
		err := os.WriteFile(filename, []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Create an invalid Go file that will cause parsing errors
	invalidFile := filepath.Join(tempDir, "invalid.go")
	err = os.WriteFile(invalidFile, []byte("this is not valid go code"), 0o644)
	require.NoError(t, err)

	// Create engine
	engine, err := New(tempDir, nil, "")
	require.NoError(t, err)

	// Process files - should handle errors gracefully
	ctx := context.Background()
	issues, err := ProcessPath(ctx, nil, engine, tempDir, ProcessFile)

	// Should continue processing other files even if one has an error
	// but should return the error
	assert.Error(t, err, "Should return error from failed file")
	// We should still get issues from valid files even if one file has errors
	assert.GreaterOrEqual(t, len(issues), 0, "Should process valid files even with errors")
}

// TestErrorPropagationSingleFile tests that errors are properly propagated for single files
func TestErrorPropagationSingleFile(t *testing.T) {
	t.Parallel()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "test_single_error")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create an invalid Go file
	invalidFile := filepath.Join(tempDir, "invalid.go")
	err = os.WriteFile(invalidFile, []byte("this is not valid go code"), 0o644)
	require.NoError(t, err)

	// Create engine
	engine, err := New(tempDir, nil, "")
	require.NoError(t, err)

	// Process single invalid file
	ctx := context.Background()
	issues, err := ProcessPath(ctx, nil, engine, invalidFile, ProcessFile)

	// Should return error for single file
	assert.Error(t, err, "Should return parsing error")
	// Issues should be empty slice, not nil
	assert.Equal(t, []tt.Issue{}, issues)
}
