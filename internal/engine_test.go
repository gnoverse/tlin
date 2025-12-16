package internal

import (
	"fmt"
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

func TestIgnorePaths(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")

	engine, err := NewEngine(testDataDir, nil, nil)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	engine.IgnorePath(filepath.Join(testDataDir, "regex/*"))
	engine.IgnorePath(filepath.Join(testDataDir, "slice0.gno"))

	files, err := filepath.Glob(filepath.Join(testDataDir, "*/*.gno"))
	if err != nil {
		t.Fatalf("failed to list files: %v", err)
	}

	for _, file := range files {
		issues, err := engine.Run(file)
		if err != nil {
			t.Fatalf("failed to run engine: %v", err)
		}

		// Check if the ignored file is not in the issues
		for _, issue := range issues {
			assert.NotEqual(t, issue.Filename, filepath.Join(testDataDir, "slice0.gno"))
			assert.NotContains(t, issue.Filename, filepath.Join(testDataDir, "regex"))
		}
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

// TestGnoFileMapping tests that issues from .gno files are correctly mapped back
func TestGnoFileMapping(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "gno_mapping_test")

	// Create a .gno file with known issues
	gnoFile := filepath.Join(tempDir, "test.gno")
	content := `package main

func test() {
	// Unnecessary type conversion that will be detected
	x := 5
	y := int(x) // This will trigger unnecessary-type-conversion rule
	
	// Useless break that will be detected
	for i := 0; i < 10; i++ {
		if i == 5 {
			break
		}
		break // This will trigger useless-break rule
	}
	
	_ = y
}
`
	err := os.WriteFile(gnoFile, []byte(content), 0o644)
	require.NoError(t, err)

	engine, err := NewEngine(tempDir, nil, nil)
	require.NoError(t, err)

	issues, err := engine.Run(gnoFile)
	assert.NoError(t, err)

	// Assert that we found at least one issue
	assert.NotEmpty(t, issues, "Should have found at least one issue in the .gno file")

	// Check that all issues have the original .gno filename
	issueCount := 0
	for _, issue := range issues {
		assert.Equal(t, gnoFile, issue.Filename, "Issue filename should be mapped back to .gno file")
		assert.True(t, strings.HasSuffix(issue.Filename, ".gno"), "Issue filename should end with .gno")
		issueCount++
	}

	// Ensure we actually checked some issues
	assert.Greater(t, issueCount, 0, "Should have found and checked at least one issue")
}

// TestNolintIsolation tests that nolint comments from one file don't affect another
func TestNolintIsolation(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "nolint_isolation_test")

	// Create two files with a detectable issue
	file1 := filepath.Join(tempDir, "file1.go")
	content1 := `package main

//nolint:unnecessary-type-conversion
func test1() {
	x := int(5)
	y := int(x) // unnecessary type conversion
	_ = y
}
`
	err := os.WriteFile(file1, []byte(content1), 0o644)
	require.NoError(t, err)

	file2 := filepath.Join(tempDir, "file2.go")
	content2 := `package main

func test2() {
	x := int(5)
	y := int(x) // unnecessary type conversion
	_ = y
}
`
	err = os.WriteFile(file2, []byte(content2), 0o644)
	require.NoError(t, err)

	engine, err := NewEngine(tempDir, nil, nil)
	require.NoError(t, err)

	// Run both files
	issues1, err := engine.Run(file1)
	assert.NoError(t, err)

	issues2, err := engine.Run(file2)
	assert.NoError(t, err)

	// File1 should have fewer issues due to nolint comment
	// File2 should have issues
	assert.Less(t, len(issues1), len(issues2), "File1 with nolint should have fewer issues than file2")

	// Verify that file2 has the unnecessary-type-conversion issue
	hasTypeConversionIssue := false
	for _, issue := range issues2 {
		if issue.Rule == "unnecessary-type-conversion" {
			hasTypeConversionIssue = true
			break
		}
	}
	assert.True(t, hasTypeConversionIssue, "File2 should have unnecessary-type-conversion issue")

	// Verify that file1 doesn't have the unnecessary-type-conversion issue
	for _, issue := range issues1 {
		assert.NotEqual(t, "unnecessary-type-conversion", issue.Rule, "File1 should not have unnecessary-type-conversion issue due to nolint")
	}
}

// TestConcurrentRuns tests that concurrent runs don't interfere with each other
func TestConcurrentRuns(t *testing.T) {
	t.Parallel()

	tempDir := createTempDir(t, "concurrent_test")

	// Create multiple test files with deterministic issues
	numFiles := 10
	files := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("test%d.go", i))
		// Each file has a unique pattern of unnecessary type conversions
		// File i will have i+1 unnecessary conversions
		content := fmt.Sprintf(`package main

// File %d
func test%d() {
	x := 5
`, i, i)

		// Add i+1 unnecessary type conversions
		for j := 0; j <= i; j++ {
			content += fmt.Sprintf("\tv%d := int(x) // unnecessary conversion %d in file %d\n", j, j, i)
		}

		content += "\t_ = x\n"
		// Add usages to avoid unused variable warnings
		for j := 0; j <= i; j++ {
			content += fmt.Sprintf("\t_ = v%d\n", j)
		}
		content += "}\n"

		err := os.WriteFile(filename, []byte(content), 0o644)
		require.NoError(t, err)
		files[i] = filename
	}

	engine, err := NewEngine(tempDir, nil, nil)
	require.NoError(t, err)

	// Run all files concurrently
	type result struct {
		filename string
		issues   []types.Issue
		err      error
	}

	resultChan := make(chan result, numFiles)
	for _, file := range files {
		go func(f string) {
			issues, err := engine.Run(f)
			resultChan <- result{filename: f, issues: issues, err: err}
		}(file)
	}

	// Collect results
	results := make(map[string][]types.Issue)
	totalIssues := 0
	for i := 0; i < numFiles; i++ {
		r := <-resultChan
		assert.NoError(t, r.err)
		results[r.filename] = r.issues
		totalIssues += len(r.issues)
	}

	// Verify we found issues
	assert.Greater(t, totalIssues, 0, "Should have found at least some issues across all files")

	// Verify that each file has at least one issue (since each file has at least one unnecessary conversion)
	for filename, issues := range results {
		assert.NotEmpty(t, issues, "File %s should have at least one issue", filename)

		// Verify that all issues belong to the correct file
		for _, issue := range issues {
			assert.Equal(t, filename, issue.Filename, "Issue should belong to the correct file")
		}
	}

	// Verify that files have different numbers of issues (as designed)
	// This helps ensure isolation - if nolint or other state leaked between files,
	// we might see identical issue counts
	issueCounts := make(map[int]int)
	for _, issues := range results {
		issueCounts[len(issues)]++
	}
	assert.Greater(t, len(issueCounts), 1, "Files should have varying numbers of issues, indicating proper isolation")
}
