package main

import (
	"context"
	"go/token"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gnoswap-labs/lint/internal/types"
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
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test case
	os.Args = []string{"cmd", "-timeout", "10s", "-cyclo", "-threshold", "15", "-ignore", "rule1,rule2", "file1.go", "file2.go"}

	config := parseFlags()

	assert.Equal(t, 10*time.Second, config.Timeout)
	assert.True(t, config.CyclomaticComplexity)
	assert.Equal(t, 15, config.CyclomaticThreshold)
	assert.Equal(t, "rule1,rule2", config.IgnoreRules)
	assert.Equal(t, []string{"file1.go", "file2.go"}, config.Paths)
}

func TestProcessFile(t *testing.T) {
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
	assert.True(t, hasDesiredExtension("test.go"))
	assert.True(t, hasDesiredExtension("test.gno"))
	assert.False(t, hasDesiredExtension("test.txt"))
	assert.False(t, hasDesiredExtension("test"))
}

func TestRunWithTimeout(t *testing.T) {
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