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

	engine, err := NewEngine(tempDir, nil)
	assert.NoError(t, err)
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.SymbolTable)
	assert.NotEmpty(t, engine.rules)
}

func TestNewEngineContent(t *testing.T) {
	t.Parallel()

	fileContent := `package test
type TestStruct struct {}
func TestFunc() {}
var TestVar int
func (ts TestStruct) TestMethod() {}
`

	engine, err := NewEngine("", []byte(fileContent))
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

func TestEngine_Run_WithNoLint(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedIssues int
	}{
		{
			name: "No nolint - should report issues",
			source: `
package main

func main() {
	var unusedVar int
}
`,
			expectedIssues: 1,
		},
		{
			name: "nolint on line - should suppress issue",
			source: `
package main

func main() {
	var unusedVar int //nolint:typecheck
}
`,
			expectedIssues: 0,
		},
		{
			name: "nolint above line - should suppress issue",
			source: `
package main

func main() {
	//nolint:typecheck
	var unusedVar int
}
`,
			expectedIssues: 0,
		},
		{
			name: "nolint above package - should suppress all issues",
			source: `
//nolint:typecheck
package main

func foo() {
	var unusedVar1 int
}

func main() {
	var unusedVar2 int
}
`,
			expectedIssues: 0,
		},
		{
			name: "nolint with specific rule - should suppress only specified issue",
			source: `
package main

func main() {
	//nolint:typecheck
	var unusedVar int
	var anotherUnusedVar int
}
`,
			expectedIssues: 1,
		},
		{
			name: "nolint with multiple rules - should suppress specified issues",
			source: `
package main

func main() {
	//nolint:typecheck,shadow
	var unusedVar int
}
`,
			expectedIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := createTempDir(t, "nolint_test")
			fileName := createTempFile(t, tempDir, "test_*.go", tt.source)

			engine, err := NewEngine(tempDir, nil)
			require.NoError(t, err)

			issues, err := engine.Run(fileName)
			require.NoError(t, err)

			assert.Len(t, issues, tt.expectedIssues)
		})
	}
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

	engine, err := NewEngine(testDataDir, nil)
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

func createTempFile(tb testing.TB, dir, prefix, content string) string {
	tb.Helper()
	tempFile, err := os.CreateTemp(dir, prefix)
	require.NoError(tb, err)
	_, err = tempFile.WriteString(content)
	require.NoError(tb, err)
	err = tempFile.Close()
	require.NoError(tb, err)
	tb.Cleanup(func() { os.Remove(tempFile.Name()) })
	return tempFile.Name()
}

func createTempDir(tb testing.TB, prefix string) string {
	tb.Helper()
	tempDir, err := os.MkdirTemp("", prefix)
	require.NoError(tb, err)
	tb.Cleanup(func() { os.RemoveAll(tempDir) })
	return tempDir
}
