package internal

import (
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	tt "github.com/gnoswap-labs/tlin/internal/types"
)

type fileMetadata struct {
	Hash         string
	LastModified time.Time
}

type CacheEntry struct {
	Metadata     fileMetadata
	Issues       []tt.Issue
	CreatedAt    time.Time
	LastAccessed time.Time
}

type Cache struct {
	CacheDir         string
	entries          map[string]CacheEntry
	mutex            sync.RWMutex
	maxAge           time.Duration
	dependencyFiles  []string
	dependencyHashes map[string]string
}

func NewCache(cacheDir string) (*Cache, error) {
	if err := os.Mkdir(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &Cache{
		CacheDir: cacheDir,
		entries:  make(map[string]CacheEntry),
	}

	if err := cache.load(); err != nil {
		return nil, fmt.Errorf("failed to load cache: %w", err)
	}

	return cache, nil
}

func (c *Cache) load() error {
	// on memory?
	cacheFile := filepath.Join(c.CacheDir, "lint_cache.gob")
	file, err := os.Open(cacheFile)
	if os.IsNotExist(err) {
		return nil // cache file doesn't exist yet. This is fine.
	}
	if err != nil {
		return fmt.Errorf("failed to open cache file: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&c.entries); err != nil {
		return fmt.Errorf("failed to decode cache file: %w", err)
	}

	return nil
}

func (c *Cache) save() error {
	cacheFile := filepath.Join(c.CacheDir, "lint_cache.gob")
	file, err := os.Create(cacheFile)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(c.entries); err != nil {
		return fmt.Errorf("failed to encode cache file: %w", err)
	}

	return nil
}

func (c *Cache) Set(filename string, issues []tt.Issue) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	metadata, err := getFileMetadata(filename)
	if err != nil {
		return fmt.Errorf("failed to get file metadata: %w", err)
	}

	c.entries[filename] = CacheEntry{
		Metadata:     metadata,
		Issues:       issues,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
	}

	return c.save()
}

func (c *Cache) Get(filename string) ([]tt.Issue, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, exists := c.entries[filename]
	if !exists {
		return nil, false
	}

	if c.isEntryInvalid(filename, entry) {
		delete(c.entries, filename)
		return nil, false
	}

	entry.LastAccessed = time.Now()
	c.entries[filename] = entry

	return entry.Issues, true
}

func (c *Cache) isEntryInvalid(filename string, entry CacheEntry) bool {
	// too old
	if time.Since(entry.CreatedAt) > c.maxAge {
		return true
	}

	currentMetadata, err := getFileMetadata(filename)
	if err != nil || currentMetadata != entry.Metadata {
		return true
	}

	if c.haveDependenciesChanged() {
		return true
	}

	return false
}

func (c *Cache) haveDependenciesChanged() bool {
	for _, file := range c.dependencyFiles {
		hash, err := getFileHash(file)
		if err != nil {
			return true
		}

		if hash != c.dependencyHashes[file] {
			return true
		}
	}

	return false
}

func (c *Cache) updateDependencyHashes() error {
	for _, file := range c.dependencyFiles {
		hash, err := getFileHash(file)
		if err != nil {
			return fmt.Errorf("failed to get hash for %s: %w", file, err)
		}
		c.dependencyHashes[file] = hash
	}
	return nil
}

func (c *Cache) SetMaxAge(duration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.maxAge = duration
}

func (c *Cache) InvalidateAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[string]CacheEntry)
	_ = c.save() // ignore error aas this is a manual operation
}

func getFileMetadata(filename string) (fileMetadata, error) {
	file, err := os.Open(filename)
	if err != nil {
		return fileMetadata{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fileMetadata{}, fmt.Errorf("failed to calculate hash: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		return fileMetadata{}, fmt.Errorf("failed to get file info: %w", err)
	}

	return fileMetadata{
		Hash:         fmt.Sprintf("%x", hash.Sum(nil)),
		LastModified: info.ModTime(),
	}, nil
}

func getFileHash(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
