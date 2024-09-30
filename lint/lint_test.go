package lint

import (
	"context"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/gnolang/tlin/internal/types"
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

func (m *mockLintEngine) RunSource(source []byte) ([]types.Issue, error) {
	args := m.Called(source)
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

func setupSourceMockEngine(expectedIssues []types.Issue, content []byte) *mockLintEngine {
	mockEngine := new(mockLintEngine)
	mockEngine.On("RunSource", content).Return(expectedIssues, nil)
	return mockEngine
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

	issues, err := ProcessFile(mockEngine, "test.go")

	assert.NoError(t, err)
	assert.Equal(t, expectedIssues, issues)
	mockEngine.AssertExpectations(t)
}

func TestProcessSource(t *testing.T) {
	t.Parallel()
	expectedIssues := []types.Issue{
		{
			Rule:     "test-rule",
			Filename: "",
			Start:    token.Position{Filename: "", Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: "", Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue",
		},
	}
	mockEngine := setupSourceMockEngine(expectedIssues, []byte("package main"))

	issues, err := ProcessSource(mockEngine, []byte("package main"))

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

	issues, err := ProcessPath(ctx, logger, mockEngine, tempDir, ProcessFile)

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

	issues, err := ProcessFiles(ctx, logger, mockEngine, paths, ProcessFile)

	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Contains(t, issues, expectedIssues[0])
	assert.Contains(t, issues, expectedIssues[1])
	mockEngine.AssertExpectations(t)
}

func TestProcessSources(t *testing.T) {
	t.Parallel()
	logger, _ := zap.NewProduction()
	ctx := context.Background()

	expectedIssues := []types.Issue{
		{
			Rule:     "rule1",
			Filename: "",
			Start:    token.Position{Filename: "", Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: "", Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 1",
		},
		{
			Rule:     "rule2",
			Filename: "",
			Start:    token.Position{Filename: "", Offset: 0, Line: 1, Column: 1},
			End:      token.Position{Filename: "", Offset: 10, Line: 1, Column: 11},
			Message:  "Test issue 2",
		},
	}

	mockEngine := new(mockLintEngine)
	mockEngine.On("RunSource", []byte("package main1")).Return([]types.Issue{expectedIssues[0]}, nil)
	mockEngine.On("RunSource", []byte("package main2")).Return([]types.Issue{expectedIssues[1]}, nil)

	issues, err := ProcessSources(ctx, logger, mockEngine, [][]byte{[]byte("package main1"), []byte("package main2")}, ProcessSource)

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

func createTempFiles(t *testing.T, dir string, fileNames ...string) []string {
	t.Helper()
	paths := make([]string, 0, len(fileNames))
	for _, fileName := range fileNames {
		filePath := filepath.Join(dir, fileName)
		_, err := os.Create(filePath)
		assert.NoError(t, err)
		paths = append(paths, filePath)
	}
	return paths
}
