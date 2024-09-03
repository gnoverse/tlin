package internal

import (
	"path/filepath"
	"runtime"
	"testing"
)

func BenchmarkGoroutineEngine(b *testing.B) {
	_, currentFile, _, _ := runtime.Caller(0)
	testDataDir := filepath.Join(filepath.Dir(currentFile), "../testdata")

	engine, err := NewEngine(testDataDir)
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
