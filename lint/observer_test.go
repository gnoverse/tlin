package lint

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingObserver captures the OnStart/OnFile/OnDone sequence so
// tests can assert ordering and counts.
type recordingObserver struct {
	mu     sync.Mutex
	events []string
	total  int
}

func (r *recordingObserver) OnStart(total int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.total = total
	r.events = append(r.events, "start")
}

func (r *recordingObserver) OnFile(path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, "file:"+path)
}

func (r *recordingObserver) OnDone() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, "done")
}

// OnStart fires once with the file count, OnFile once per file
// (in any order — workers are concurrent), and OnDone fires last.
func TestObserverCallbacksFire(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "observer-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	files := []string{
		filepath.Join(tempDir, "a.go"),
		filepath.Join(tempDir, "b.go"),
		filepath.Join(tempDir, "c.go"),
	}
	for _, f := range files {
		require.NoError(t, os.WriteFile(f, []byte("package main\n"), 0o644))
	}

	mockEngine := new(mockLintEngine)
	for _, f := range files {
		mockEngine.On("Run", f).Return([]types.Issue{}, nil)
	}

	rec := &recordingObserver{}
	_, err = ProcessFiles(context.Background(), nil, mockEngine, []string{tempDir}, rec)
	require.NoError(t, err)

	assert.Equal(t, len(files), rec.total, "OnStart received the file count")
	require.Len(t, rec.events, 1+len(files)+1,
		"events: 1 start + 1 file per file + 1 done; got %v", rec.events)
	assert.Equal(t, "start", rec.events[0], "OnStart fires first")
	assert.Equal(t, "done", rec.events[len(rec.events)-1], "OnDone fires last")

	fileEvents := rec.events[1 : len(rec.events)-1]
	for _, e := range fileEvents {
		assert.Contains(t, e, "file:", "middle events are OnFile callbacks")
	}
}

// A nil observer must not panic — the processing layer substitutes
// nopObserver so callers don't need to provide one for headless runs.
func TestNilObserverNoOps(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "nil-observer-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	file := filepath.Join(tempDir, "a.go")
	require.NoError(t, os.WriteFile(file, []byte("package main\n"), 0o644))

	mockEngine := new(mockLintEngine)
	mockEngine.On("Run", file).Return([]types.Issue{}, nil)

	_, err = ProcessFiles(context.Background(), nil, mockEngine, []string{tempDir}, nil)
	require.NoError(t, err)
}
