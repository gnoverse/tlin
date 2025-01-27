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
	"sync"
	"testing"
	"time"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/gnolang/tlin/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type mockLintEngine struct {
	mock.Mock
}

func (m *mockLintEngine) Run(filePath string) ([]tt.Issue, error) {
	args := m.Called(filePath)
	return args.Get(0).([]tt.Issue), args.Error(1)
}

func (m *mockLintEngine) RunSource(source []byte) ([]tt.Issue, error) {
	args := m.Called(source)
	return args.Get(0).([]tt.Issue), args.Error(1)
}

func (m *mockLintEngine) IgnoreRule(rule string) {
	m.Called(rule)
}

func (m *mockLintEngine) IgnorePath(path string) {
	m.Called(path)
}

func setupMockEngine(expectedIssues []tt.Issue, filePath string) *mockLintEngine {
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
				ConfigurationPath:   ".tlin.yaml",
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
				ConfigurationPath:   ".tlin.yaml",
			},
		},
		{
			name: "AutoFix with custom confidence",
			args: []string{"-fix", "-confidence", "0.9", "file.go"},
			expected: Config{
				AutoFix:             true,
				Paths:               []string{"file.go"},
				ConfidenceThreshold: 0.9,
				ConfigurationPath:   ".tlin.yaml",
			},
		},
		{
			name: "JsonOutput",
			args: []string{"-json", "file.go"},
			expected: Config{
				Paths:               []string{"file.go"},
				JsonOutput:          true,
				ConfidenceThreshold: defaultConfidenceThreshold,
				ConfigurationPath:   ".tlin.yaml",
			},
		},
		{
			name: "Output",
			args: []string{"-o", "output.svg", "file.go"},
			expected: Config{
				Paths:               []string{"file.go"},
				Output:              "output.svg",
				ConfidenceThreshold: defaultConfidenceThreshold,
				ConfigurationPath:   ".tlin.yaml",
			},
		},
		{
			name: "Configuration File",
			args: []string{"-c", "config.yaml", "file.go"},
			expected: Config{
				Paths:               []string{"file.go"},
				ConfidenceThreshold: defaultConfidenceThreshold,
				ConfigurationPath:   "config.yaml",
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
			assert.Equal(t, tt.expected.Output, config.Output)
			assert.Equal(t, tt.expected.ConfigurationPath, config.ConfigurationPath)
		})
	}
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

func TestInitConfigurationFile(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "init-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, ".tlin.yaml")

	err = initConfigurationFile(configPath)
	assert.NoError(t, err)

	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)

	expectedConfig := lint.Config{
		Name:  "tlin",
		Rules: map[string]tt.ConfigRule{},
	}
	config := &lint.Config{}
	yaml.Unmarshal(content, config)

	assert.Equal(t, expectedConfig, *config)
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
		runCFGAnalysis(ctx, logger, []string{tempFile}, "targetFunc", "")
	})

	assert.Contains(t, output, "CFG for function targetFunc in file")
	assert.Contains(t, output, "digraph mgraph")
	assert.Contains(t, output, "\"for loop")
	assert.Contains(t, output, "\"assignment")
	assert.NotContains(t, output, "mainFunc")
	assert.NotContains(t, output, "ignoredFunc")

	t.Logf("output: %s", output)

	output = captureOutput(t, func() {
		runCFGAnalysis(ctx, logger, []string{tempFile}, "nonExistentFunc", "")
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
	err = os.WriteFile(testFile, []byte(sliceRangeIssueExample), 0o644)
	assert.NoError(t, err)

	expectedIssues := []tt.Issue{
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
	err = os.WriteFile(testFile, []byte(sliceRangeIssueExample), 0o644)
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

			var actualContent map[string][]tt.Issue
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
				assert.Equal(t, tt.SeverityError, issue.Severity)
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
	err = os.WriteFile(testFile, []byte(sliceRangeIssueExample), 0o644)
	assert.NoError(t, err)

	expectedIssues := []tt.Issue{
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
	runNormalLintProcess(ctx, logger, mockEngine, []string{testFile}, true, jsonOutput)
}

func createTempFileWithContent(t *testing.T, content string) string {
	t.Helper()
	tempFile, err := os.CreateTemp("", "test*.go")
	assert.NoError(t, err)
	defer tempFile.Close()

	_, err = tempFile.Write([]byte(content))
	assert.NoError(t, err)

	return tempFile.Name()
}

var mu sync.Mutex

func captureOutput(t *testing.T, f func()) string {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()

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
