package lint

import (
	"context"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sync"
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

func (m *mockLintEngine) RunWithContext(_ context.Context, filePath string) ([]types.Issue, error) {
	return m.Run(filePath)
}

func (m *mockLintEngine) RunSourceWithContext(_ context.Context, source []byte) ([]types.Issue, error) {
	return m.RunSource(source)
}

func (m *mockLintEngine) RunFile(_ context.Context, filePath string) ([]types.Issue, error) {
	return m.Run(filePath)
}

func (m *mockLintEngine) RunPackage(_ context.Context, _ string, _ []string) ([]types.Issue, error) {
	return nil, nil
}

func (m *mockLintEngine) IgnoreRule(rule string) {
	m.Called(rule)
}

func (m *mockLintEngine) IgnorePath(path string) {
	m.Called(path)
}

func setupMockEngine(expectedIssues []types.Issue, filePath string) *mockLintEngine {
	mockEngine := new(mockLintEngine)
	mockEngine.On("Run", filePath).Return(expectedIssues, nil)
	return mockEngine
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

	issues, err := ProcessPath(ctx, logger, mockEngine, tempDir, nil)

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

	issues, err := ProcessFiles(ctx, logger, mockEngine, paths, nil)

	assert.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Contains(t, issues, expectedIssues[0])
	assert.Contains(t, issues, expectedIssues[1])
	mockEngine.AssertExpectations(t)
}
// processDirectory must amortize PackageRule dispatch to one
// engine.RunPackage call per parent directory, no matter how many
// files in the directory match.
func TestPackageRuleRunsOncePerPackage(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "package-rule-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	pkgADir := filepath.Join(tempDir, "pkg_a")
	pkgBDir := filepath.Join(tempDir, "pkg_b")
	assert.NoError(t, os.Mkdir(pkgADir, 0o755))
	assert.NoError(t, os.Mkdir(pkgBDir, 0o755))

	pkgAFiles := createTempFiles(t, pkgADir, "a1.go", "a2.go", "a3.go")
	pkgBFiles := createTempFiles(t, pkgBDir, "b1.go")

	engine := newPackageCountingEngine()
	for _, p := range slices.Concat(pkgAFiles, pkgBFiles) {
		engine.On("Run", p).Return([]types.Issue{}, nil)
	}

	// Pass the parent dir, not individual files, so the call goes
	// through processDirectory (the path that splits package vs file
	// dispatch).
	_, err = ProcessFiles(context.Background(), zap.NewNop(), engine, []string{tempDir}, nil)
	assert.NoError(t, err)
	engine.AssertExpectations(t)

	assert.ElementsMatch(t, []packageCall{
		{dir: pkgADir, count: 3},
		{dir: pkgBDir, count: 1},
	}, engine.packageCalls)
}

// packageCountingEngine wraps mockLintEngine so RunPackage records
// every invocation; using a slice keeps both "which dirs" and "how
// many" assertable in one shot.
type packageCountingEngine struct {
	*mockLintEngine
	mu           sync.Mutex
	packageCalls []packageCall
}

type packageCall struct {
	dir   string
	count int
}

func newPackageCountingEngine() *packageCountingEngine {
	return &packageCountingEngine{mockLintEngine: new(mockLintEngine)}
}

func (e *packageCountingEngine) RunPackage(_ context.Context, dir string, paths []string) ([]types.Issue, error) {
	e.mu.Lock()
	e.packageCalls = append(e.packageCalls, packageCall{dir: dir, count: len(paths)})
	e.mu.Unlock()
	return nil, nil
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
