package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gnoswap-labs/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempDir creates a temporary directory and returns its path.
// It also registers a cleanup function to remove the directory after the test.
func createTempDir(t testing.TB, prefix string) string {
	tempDir, err := os.MkdirTemp("", prefix)
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	return tempDir
}

// assertFileExists checks if a file exists and has the expected content.
func assertFileExists(t *testing.T, path string, expectedContent string) {
	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, string(content))
}

func TestNewEngine(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "engine_test")

	engine, err := NewEngine(tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.SymbolTable)
	assert.NotEmpty(t, engine.rules)
}

func TestEngine_IgnoreRule(t *testing.T) {
	t.Parallel()
	engine := &Engine{}
	engine.IgnoreRule("test_rule")

	assert.True(t, engine.ignoredRules["test_rule"])
}

func TestEngine_PrepareFile(t *testing.T) {
	t.Parallel()
	engine := &Engine{}

	t.Run("Go file", func(t *testing.T) {
		goFile := "test.go"
		result, err := engine.prepareFile(goFile)
		assert.NoError(t, err)
		assert.Equal(t, goFile, result)
	})

	t.Run("Gno file", func(t *testing.T) {
		tempDir := createTempDir(t, "gno_test")

		gnoFile := filepath.Join(tempDir, "test.gno")
		err := os.WriteFile(gnoFile, []byte("package main"), 0644)
		require.NoError(t, err)

		result, err := engine.prepareFile(gnoFile)
		assert.NoError(t, err)
		assert.NotEqual(t, gnoFile, result)
		assert.True(t, filepath.Ext(result) == ".go")
	})
}

func TestEngine_CleanupTemp(t *testing.T) {
	t.Parallel()
	engine := &Engine{}

	tempDir := createTempDir(t, "cleanup_test")

	tempFile := filepath.Join(tempDir, "temp_test.go")
	_, err := os.Create(tempFile)
	require.NoError(t, err)

	engine.cleanupTemp(tempFile)
	_, err = os.Stat(tempFile)
	assert.True(t, os.IsNotExist(err))
}

func TestReadSourceCode(t *testing.T) {
	t.Parallel()
	tempDir := createTempDir(t, "source_code_test")

	testFile := filepath.Join(tempDir, "test.go")
	content := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	sourceCode, err := ReadSourceCode(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, sourceCode)
	assert.Len(t, sourceCode.Lines, 5)
	assert.Equal(t, "package main", sourceCode.Lines[0])
}

func BenchmarkFilterUndefinedIssues(b *testing.B) {
	engine := &Engine{
		SymbolTable: &SymbolTable{},
	}

	issues := []types.Issue{
		{Rule: "typecheck", Message: "undefined: someSymbol"},
		{Rule: "other", Message: "some other issue"},
		{Rule: "typecheck", Message: "undefined: anotherSymbol"},
		{Rule: "typecheck", Message: "some other typecheck issue"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.filterUndefinedIssues(issues)
	}
}

// create dummy source code for benchmark
var testSrc = strings.Repeat("hello world", 5000)

func BenchmarkCreateTempGoFile(b *testing.B) {
	tempDir := createTempDir(b, "benchmark")

	// create temp go file for benchmark
	gnoContent := []byte(testSrc)
	gnoFile := filepath.Join(tempDir, "main.gno")
	if err := os.WriteFile(gnoFile, gnoContent, 0o644); err != nil {
		b.Fatalf("failed to write temp gno file: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		f, err := createTempGoFile(gnoFile)
		if err != nil {
			b.Fatalf("failed to create temp go file: %v", err)
		}
		os.Remove(f)
	}
}

func BenchmarkRun(b *testing.B) {
	_, currentFile, _, _ := runtime.Caller(0)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")

	engine, err := NewEngine(testDataDir)
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(testDataDir, "*/*.gno"))
	if err != nil {
		b.Fatalf("failed to list files: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, file := range files {
			_, err := engine.Run(file)
			if err != nil {
				b.Fatalf("failed to run engine: %v", err)
			}
		}
	}
}
