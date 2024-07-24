package scanner

import (
	"os"
	"path/filepath"
	"sync"
)

type FileInfo struct {
	Path string
	Size int64
}

type Scanner struct {
	rootDir    string
	extensions []string
}

func New(rootDir string, extensions ...string) *Scanner {
	return &Scanner{
		rootDir:    rootDir,
		extensions: extensions,
	}
}

func (s *Scanner) Scan() ([]FileInfo, error) {
	var (
		files []FileInfo
		mutex sync.Mutex
		wg    sync.WaitGroup
	)

	err := filepath.Walk(s.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if s.isTargetFile(path) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				fileInfo := FileInfo{
					Path: path,
					Size: info.Size(),
				}
				mutex.Lock()
				files = append(files, fileInfo)
				mutex.Unlock()
			}()
		}
		return nil
	})

	wg.Wait()
	return files, err
}

func (s *Scanner) isTargetFile(path string) bool {
	if len(s.extensions) == 0 {
		return true
	}

	ext := filepath.Ext(path)
	for _, targetExt := range s.extensions {
		if ext == targetExt {
			return true
		}
	}
	return false
}
