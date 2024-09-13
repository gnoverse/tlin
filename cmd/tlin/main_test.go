package main

import (
	"bytes"
	"context"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gnoswap-labs/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockLintEngine is a mock implementation of the LintEngine interface
type MockLintEngine struct {
	mock.Mock
}

func (m *MockLintEngine) Run(filePath string) ([]types.Issue, error) {
	args := m.Called(filePath)
	return args.Get(0).([]types.Issue), args.Error(1)
}

func (m *MockLintEngine) IgnoreRule(rule string) {
	m.Called(rule)
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected Config
	}{
		{
			name: "AutoFix",
			args: []string{"-fix", "file.go"},
			expected: Config{
				AutoFix:             true,
				Paths:               []string{"file.go"},
				ConfidenceThreshold: defaultConfidenceThreshold,
			},
		},
		{
			name: "AutoFix with DryRun",
			args: []string{"-fix", "-dry-run", "file.go"},
			expected: Config{
				AutoFix:             true,
				DryRun:              true,
				Paths:               []string{"file.go"},
				ConfidenceThreshold: defaultConfidenceThreshold,
			},
		},
		{
			name: "AutoFix with custom confidence",
			args: []string{"-fix", "-confidence", "0.9", "file.go"},
			expected: Config{
				AutoFix:             true,
				Paths:               []string{"file.go"},
				ConfidenceThreshold: 0.9,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = append([]string{"cmd"}, tt.args...)
			config := parseFlags(tt.args)

			assert.Equal(t, tt.expected.AutoFix, config.AutoFix)
			assert.Equal(t, tt.expected.DryRun, config.DryRun)
			assert.Equal(t, tt.expected.ConfidenceThreshold, config.ConfidenceThreshold)
			assert.Equal(t, tt.expected.Paths, config.Paths)
		})
	}
}

func TestProcessFile(t *testing.T) {
	t.Parallel()
	mockEngine := new(MockLintEngine)
	expectedIssues := []types.Issue{
		{
			Rule:     "test-rule",
			Filename: "test.go",
			Start:    token.Position{Filename: "test.go", Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: "test.go", Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue",
		},
	}
	mockEngine.On("Run", "test.go").Return(expectedIssues, nil)

	issues, err := processFile(mockEngine, "test.go")

	assert.NoError(t, err)
	assert.Equal(t, expectedIssues, issues)
	mockEngine.AssertExpectations(t)
}

func TestProcessPath(t *testing.T) {
	t.Parallel()
	logger, _ := zap.NewProduction()
	mockEngine := new(MockLintEngine)
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile1 := filepath.Join(tempDir, "test1.go")
	testFile2 := filepath.Join(tempDir, "test2.go")
	_, err = os.Create(testFile1)
	assert.NoError(t, err)
	_, err = os.Create(testFile2)
	assert.NoError(t, err)

	expectedIssues1 := []types.Issue{
		{
			Rule:     "rule1",
			Filename: testFile1,
			Start:    token.Position{Filename: testFile1, Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: testFile1, Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 1",
		},
	}
	expectedIssues2 := []types.Issue{
		{
			Rule:     "rule2",
			Filename: testFile2,
			Start:    token.Position{Filename: testFile2, Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: testFile2, Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 2",
		},
	}

	mockEngine.On("Run", testFile1).Return(expectedIssues1, nil)
	mockEngine.On("Run", testFile2).Return(expectedIssues2, nil)

	issues, err := processPath(ctx, logger, mockEngine, tempDir, processFile)

	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Contains(t, issues, expectedIssues1[0])
	assert.Contains(t, issues, expectedIssues2[0])
	mockEngine.AssertExpectations(t)
}

func TestProcessFiles(t *testing.T) {
	t.Parallel()
	logger, _ := zap.NewProduction()
	mockEngine := new(MockLintEngine)
	ctx := context.Background()

	tempFile1, err := os.CreateTemp("", "test1*.go")
	assert.NoError(t, err)
	defer os.Remove(tempFile1.Name())

	tempFile2, err := os.CreateTemp("", "test2*.go")
	assert.NoError(t, err)
	defer os.Remove(tempFile2.Name())

	paths := []string{tempFile1.Name(), tempFile2.Name()}

	expectedIssues1 := []types.Issue{
		{
			Rule:     "rule1",
			Filename: tempFile1.Name(),
			Start:    token.Position{Filename: tempFile1.Name(), Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: tempFile1.Name(), Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 1",
		},
	}
	expectedIssues2 := []types.Issue{
		{
			Rule:     "rule2",
			Filename: tempFile2.Name(),
			Start:    token.Position{Filename: tempFile2.Name(), Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: tempFile2.Name(), Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 2",
		},
	}

	mockEngine.On("Run", tempFile1.Name()).Return(expectedIssues1, nil)
	mockEngine.On("Run", tempFile2.Name()).Return(expectedIssues2, nil)

	issues, err := processFiles(ctx, logger, mockEngine, paths, processFile)

	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Contains(t, issues, expectedIssues1[0])
	assert.Contains(t, issues, expectedIssues2[0])
	mockEngine.AssertExpectations(t)
}

func TestHasDesiredExtension(t *testing.T) {
	t.Parallel()
	assert.True(t, hasDesiredExtension("test.go"))
	assert.True(t, hasDesiredExtension("test.gno"))
	assert.False(t, hasDesiredExtension("test.txt"))
	assert.False(t, hasDesiredExtension("test"))
}

func TestRunWithTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan bool)
	go func() {
		runWithTimeout(ctx, func() {
			time.Sleep(1 * time.Second)
			done <- true
		})
	}()

	select {
	case <-done:
		// Function completed successfully
	case <-ctx.Done():
		t.Fatal("Function timed out unexpectedly")
	}
}

func TestRunCFGAnalysis(t *testing.T) {
	t.Parallel()
	logger, _ := zap.NewProduction()

	tempFile, err := os.CreateTemp("", "test*.go")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	testCode := `package main 	// 1
								// 2
func mainFunc() { 				// 3
    x := 1						// 4 
    if x > 0 {					// 5
        x = 2 					// 6
    } else { 					// 7
        x = 3 					// 8
    } 							// 9
} 								// 10
								// 11
func targetFunc() { 			// 12
    y := 10 					// 13
    for i := 0; i < 5; i++ { 	// 14
        y += i 					// 15
    } 							// 16
} 								// 17
								// 18
func ignoredFunc() { 			// 19
    z := "hello" 				// 20
    println(z) 					// 21
} 								// 22
`
	_, err = tempFile.Write([]byte(testCode))
	assert.NoError(t, err)
	tempFile.Close()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	runCFGAnalysis(ctx, logger, []string{tempFile.Name()}, "targetFunc")

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, "CFG for function targetFunc in file")
	assert.Contains(t, output, "digraph mgraph")
	assert.Contains(t, output, "\"for loop")
	assert.Contains(t, output, "\"assignment")
	assert.NotContains(t, output, "mainFunc")
	assert.NotContains(t, output, "ignoredFunc")

	t.Logf("output: %s", output)

	r, w, _ = os.Pipe()
	os.Stdout = w

	runCFGAnalysis(ctx, logger, []string{tempFile.Name()}, "nonExistentFunc")

	w.Close()
	os.Stdout = oldStdout
	buf.Reset()
	io.Copy(&buf, r)
	output = buf.String()

	assert.Contains(t, output, "Function not found: nonExistentFunc")
}

func TestRunAutoFix(t *testing.T) {
	logger, _ := zap.NewProduction()
	mockEngine := new(MockLintEngine)
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "autofix-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.go")
	initialContent := `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:len(slice)]
}
`
	err = os.WriteFile(testFile, []byte(initialContent), 0644)
	assert.NoError(t, err)

	expectedIssues := []types.Issue{
		{
			Rule:       "simplify-slice-range",
			Filename:   testFile,
			Message:    "unnecessary use of len() in slice expression, can be simplified",
			Start:      token.Position{Line: 5, Column: 5},
			End:        token.Position{Line: 5, Column: 24},
			Suggestion: "_ = slice[:]",
			Confidence: 0.9,
		},
	}

	mockEngine.On("Run", testFile).Return(expectedIssues, nil)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run auto-fix
	runAutoFix(ctx, logger, mockEngine, []string{testFile}, false, 0.8)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check if the fix was applied
	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)

	expectedContent := `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:]
}
`
	assert.Equal(t, expectedContent, string(content))
	assert.Contains(t, output, "Fixed issues in")

	// Test dry-run mode
	err = os.WriteFile(testFile, []byte(initialContent), 0644)
	assert.NoError(t, err)

	r, w, _ = os.Pipe()
	os.Stdout = w

	runAutoFix(ctx, logger, mockEngine, []string{testFile}, true, 0.8)

	w.Close()
	os.Stdout = oldStdout
	buf.Reset()
	io.Copy(&buf, r)
	output = buf.String()

	content, err = os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, initialContent, string(content))
	assert.Contains(t, output, "Would fix issue in")
}
