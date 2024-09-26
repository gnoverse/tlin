package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gnoswap-labs/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type mockLintEngine struct {
	mock.Mock
}

func (m *mockLintEngine) Run(filePath string) ([]types.Issue, error) {
	args := m.Called(filePath)
	return args.Get(0).([]types.Issue), args.Error(1)
}

func (m *mockLintEngine) IgnoreRule(rule string) {
	m.Called(rule)
}

func setupMockEngine(expectedIssues []types.Issue, filePath string) *mockLintEngine {
	mockEngine := new(mockLintEngine)
	mockEngine.On("Run", filePath).Return(expectedIssues, nil)
	return mockEngine
}

func TestParseFlags(t *testing.T) {
	t.Parallel()
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
		{
			name: "JsonOutput",
			args: []string{"-json-output", "output.json", "file.go"},
			expected: Config{
				Paths:               []string{"file.go"},
				JsonOutput:          "output.json",
				ConfidenceThreshold: defaultConfidenceThreshold,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			config := parseFlags(tt.args)

			assert.Equal(t, tt.expected.AutoFix, config.AutoFix)
			assert.Equal(t, tt.expected.DryRun, config.DryRun)
			assert.Equal(t, tt.expected.ConfidenceThreshold, config.ConfidenceThreshold)
			assert.Equal(t, tt.expected.Paths, config.Paths)
			assert.Equal(t, tt.expected.JsonOutput, config.JsonOutput)
		})
	}
}

func TestProcessFile(t *testing.T) {
	t.Parallel()
	expectedIssues := []types.Issue{
		{
			Rule:     "test-rule",
			Filename: "test.go",
			Start:    token.Position{Filename: "test.go", Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: "test.go", Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue",
		},
	}
	mockEngine := setupMockEngine(expectedIssues, "test.go")

	issues, err := processFile(mockEngine, "test.go")

	assert.NoError(t, err)
	assert.Equal(t, expectedIssues, issues)
	mockEngine.AssertExpectations(t)
}

func TestProcessPath(t *testing.T) {
	t.Parallel()
	logger, _ := zap.NewProduction()
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	paths := createTempFiles(t, tempDir, "test1.go", "test2.go")

	expectedIssues := []types.Issue{
		{
			Rule:     "rule1",
			Filename: paths[0],
			Start:    token.Position{Filename: paths[0], Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: paths[0], Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 1",
		},
		{
			Rule:     "rule2",
			Filename: paths[1],
			Start:    token.Position{Filename: paths[1], Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: paths[1], Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 2",
		},
	}

	mockEngine := new(mockLintEngine)
	mockEngine.On("Run", paths[0]).Return([]types.Issue{expectedIssues[0]}, nil)
	mockEngine.On("Run", paths[1]).Return([]types.Issue{expectedIssues[1]}, nil)

	issues, err := processPath(ctx, logger, mockEngine, tempDir, processFile)

	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Contains(t, issues, expectedIssues[0])
	assert.Contains(t, issues, expectedIssues[1])
	mockEngine.AssertExpectations(t)
}

func TestProcessFiles(t *testing.T) {
	t.Parallel()
	logger, _ := zap.NewProduction()
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	paths := createTempFiles(t, tempDir, "test1.go", "test2.go")

	expectedIssues := []types.Issue{
		{
			Rule:     "rule1",
			Filename: paths[0],
			Start:    token.Position{Filename: paths[0], Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: paths[0], Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 1",
		},
		{
			Rule:     "rule2",
			Filename: paths[1],
			Start:    token.Position{Filename: paths[1], Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: paths[1], Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 2",
		},
	}

	mockEngine := new(mockLintEngine)
	mockEngine.On("Run", paths[0]).Return([]types.Issue{expectedIssues[0]}, nil)
	mockEngine.On("Run", paths[1]).Return([]types.Issue{expectedIssues[1]}, nil)

	issues, err := processFiles(ctx, logger, mockEngine, paths, processFile)

	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Contains(t, issues, expectedIssues[0])
	assert.Contains(t, issues, expectedIssues[1])
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
		// no problem
	case <-ctx.Done():
		t.Fatal("function unexpectedly timed out")
	}
}

func TestRunCFGAnalysis(t *testing.T) {
	t.Parallel()
	logger, _ := zap.NewProduction()

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
	tempFile := createTempFileWithContent(t, testCode)
	defer os.Remove(tempFile)

	ctx := context.Background()

	output := captureOutput(t, func() {
		runCFGAnalysis(ctx, logger, []string{tempFile}, "targetFunc")
	})

	assert.Contains(t, output, "CFG for function targetFunc in file")
	assert.Contains(t, output, "digraph mgraph")
	assert.Contains(t, output, "\"for loop")
	assert.Contains(t, output, "\"assignment")
	assert.NotContains(t, output, "mainFunc")
	assert.NotContains(t, output, "ignoredFunc")

	t.Logf("output: %s", output)

	output = captureOutput(t, func() {
		runCFGAnalysis(ctx, logger, []string{tempFile}, "nonExistentFunc")
	})

	assert.Contains(t, output, "Function not found: nonExistentFunc")
}

const sliceRangeIssueExample = `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:len(slice)]
}
`

func TestRunAutoFix(t *testing.T) {
	logger, _ := zap.NewProduction()
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "autofix-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.go")
	err = os.WriteFile(testFile, []byte(sliceRangeIssueExample), 0644)
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

	mockEngine := setupMockEngine(expectedIssues, testFile)

	output := captureOutput(t, func() {
		runAutoFix(ctx, logger, mockEngine, []string{testFile}, false, 0.8)
	})

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

	// dry run test
	err = os.WriteFile(testFile, []byte(sliceRangeIssueExample), 0644)
	assert.NoError(t, err)

	output = captureOutput(t, func() {
		runAutoFix(ctx, logger, mockEngine, []string{testFile}, true, 0.8)
	})

	content, err = os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, sliceRangeIssueExample, string(content))
	assert.Contains(t, output, "Would fix issue in")
}

func TestRunJsonOutput(t *testing.T) {
	if os.Getenv("BE_CRASHER") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=TestRunJsonOutput")
		cmd.Env = append(os.Environ(), "BE_CRASHER=1")
		output, err := cmd.CombinedOutput() // stdout and stderr capture
		if e, ok := err.(*exec.ExitError); ok && !e.Success() {
			tempDir := string(bytes.TrimRight(output, "\n"))
			defer os.RemoveAll(tempDir)

			// check if issues are written
			jsonOutput := filepath.Join(tempDir, "output.json")
			content, err := os.ReadFile(jsonOutput)
			assert.NoError(t, err)

			var actualContent map[string][]types.Issue
			err = json.Unmarshal(content, &actualContent)
			assert.NoError(t, err)

			assert.Len(t, actualContent, 1)
			for filename, issues := range actualContent {
				assert.True(t, strings.HasSuffix(filename, "test.go"))
				assert.Len(t, issues, 1)
				issue := issues[0]
				assert.Equal(t, "simplify-slice-range", issue.Rule)
				assert.Equal(t, "unnecessary use of len() in slice expression, can be simplified", issue.Message)
				assert.Equal(t, "_ = slice[:]", issue.Suggestion)
				assert.Equal(t, 0.9, issue.Confidence)
				assert.Equal(t, 5, issue.Start.Line)
				assert.Equal(t, 5, issue.Start.Column)
				assert.Equal(t, 5, issue.End.Line)
				assert.Equal(t, 24, issue.End.Column)
			}

			return
		}
		t.Fatalf("process failed with error %v, expected exit status 1", err)
	}

	logger, _ := zap.NewProduction()
	ctx := context.Background()

	tempDir, err := os.MkdirTemp("", "json-test")
	assert.NoError(t, err)
	fmt.Println(tempDir)

	testFile := filepath.Join(tempDir, "test.go")
	err = os.WriteFile(testFile, []byte(sliceRangeIssueExample), 0644)
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

	mockEngine := setupMockEngine(expectedIssues, testFile)

	jsonOutput := filepath.Join(tempDir, "output.json")
	runNormalLintProcess(ctx, logger, mockEngine, []string{testFile}, jsonOutput)
}

func createTempFiles(t *testing.T, dir string, fileNames ...string) []string {
	var paths []string
	for _, fileName := range fileNames {
		filePath := filepath.Join(dir, fileName)
		_, err := os.Create(filePath)
		assert.NoError(t, err)
		paths = append(paths, filePath)
	}
	return paths
}

func createTempFileWithContent(t *testing.T, content string) string {
	tempFile, err := os.CreateTemp("", "test*.go")
	assert.NoError(t, err)
	defer tempFile.Close()

	_, err = tempFile.Write([]byte(content))
	assert.NoError(t, err)

	return tempFile.Name()
}

func captureOutput(_ *testing.T, f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
