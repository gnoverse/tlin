package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "engine_test")

	engine, err := NewEngine(tempDir, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, engine)
	// assert.NotNil(t, engine.SymbolTable)
	assert.NotEmpty(t, engine.rules)
}

func TestNewEngineConfig(t *testing.T) {
	t.Parallel()

	config := map[string]types.ConfigRule{
		"useless-break": {
			Severity: types.SeverityOff,
		},
		"deprecated-function": {
			Severity: types.SeverityWarning,
		},
		"test-rule": {
			Severity: types.SeverityError,
		},
	}
	tempDir := createTempDir(t, "engine_test")

	engine, err := NewEngine(tempDir, nil, config)
	assert.NoError(t, err)
	assert.NotNil(t, engine)

	assert.NotEmpty(t, engine.rules)

	for key, rule := range engine.rules {
		switch key {
		case "deprecated-function":
			assert.Equal(t, types.SeverityWarning, rule.Severity())
		case "test-rule":
			assert.Fail(t, "test-rule should not be in the rules")
		}
	}

	assert.True(t, engine.ignoredRules["useless-break"])
}

func TestNewEngineContent(t *testing.T) {
	t.Parallel()

	fileContent := `package test
type TestStruct struct {}
func TestFunc() {}
var TestVar int
func (ts TestStruct) TestMethod() {}
`

	engine, err := NewEngine("", []byte(fileContent), nil)
	assert.NoError(t, err)
	assert.NotNil(t, engine)
	// assert.NotNil(t, engine.SymbolTable)
	assert.NotEmpty(t, engine.rules)
}

func TestEngine_IgnoreRule(t *testing.T) {
	t.Parallel()
	engine := &Engine{}
	engine.IgnoreRule("test_rule")

	assert.True(t, engine.ignoredRules["test_rule"])
}

func TestEngine_IgnorePath(t *testing.T) {
	t.Parallel()
	engine := &Engine{}
	engine.IgnorePath("test_path")

	assert.Equal(t, "test_path", engine.ignoredPaths[0])
}

func TestEngine_PrepareFile(t *testing.T) {
	t.Parallel()
	engine := &Engine{}

	t.Run("Go file", func(t *testing.T) {
		t.Parallel()
		goFile := "test.go"
		result, err := engine.prepareFile(goFile)
		assert.NoError(t, err)
		assert.Equal(t, goFile, result)
	})

	t.Run("Gno file", func(t *testing.T) {
		t.Parallel()
		tempDir := createTempDir(t, "gno_test")

		gnoFile := filepath.Join(tempDir, "test.gno")
		err := os.WriteFile(gnoFile, []byte("package main"), 0o644)
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
	err := os.WriteFile(testFile, []byte(content), 0o644)
	require.NoError(t, err)

	sourceCode, err := ReadSourceCode(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, sourceCode)
	assert.Len(t, sourceCode.Lines, 5)
	assert.Equal(t, "package main", sourceCode.Lines[0])
}

// create dummy source code for benchmark
var testSrc = strings.Repeat("hello world", 5000)

func BenchmarkCreateTempGoFile(b *testing.B) {
	tempDir := createTempDir(b, "benchmark")

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
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(b, ok)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")

	engine, err := NewEngine(testDataDir, nil, nil)
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

func createTempDir(tb testing.TB, prefix string) string {
	tb.Helper()
	tempDir, err := os.MkdirTemp("", prefix)
	require.NoError(tb, err)
	tb.Cleanup(func() { os.RemoveAll(tempDir) })
	return tempDir
}
