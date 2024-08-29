package internal

import (
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	tt "github.com/gnoswap-labs/lint/internal/types"
)

type FileMetadata struct {
	Hash string
	LastModified time.Time
}

type CacheEntry struct {
	Metadata FileMetadata
	Issues []tt.Issue
}

type Cache struct {
	CacheDir string
	Entries map[string]CacheEntry
}

func NewCache(cacheDir string) (*Cache, error) {
	if err := os.Mkdir(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &Cache{
		CacheDir: cacheDir,
		Entries: make(map[string]CacheEntry),
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
	if err := decoder.Decode(&c.Entries); err != nil {
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
	if err := encoder.Encode(c.Entries); err != nil {
		return fmt.Errorf("failed to encode cache file: %w", err)
	}

	return nil
}

func (c *Cache) Set(filename string, issues []tt.Issue) error {
	metadata, err := getFileMetadata(filename)
	if err != nil {
		return fmt.Errorf("failed to get file metadata: %w", err)
	}

	c.Entries[filename] = CacheEntry{
		Metadata: metadata,
		Issues: issues,
	}

	return c.save()
}

func (c *Cache) Get(filename string) ([]tt.Issue, bool) {
	entry, exists := c.Entries[filename]
	if !exists {
		return nil, false
	}

	metadata, err := getFileMetadata(filename)
	if err != nil {
		return nil, false
	}

	if entry.Metadata.Hash != metadata.Hash ||
		!entry.Metadata.LastModified.Equal(metadata.LastModified) {
		return nil, false
	}

	return entry.Issues, true
}

func getFileMetadata(filename string) (FileMetadata, error) {
	file, err := os.Open(filename)
	if err != nil {
		return FileMetadata{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil{
		return FileMetadata{}, fmt.Errorf("failed to calculate hash: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		return FileMetadata{}, fmt.Errorf("failed to get file info: %w", err)
	}

	return FileMetadata{
		Hash: fmt.Sprintf("%x", hash.Sum(nil)),
		LastModified: info.ModTime(),
	}, nil
}
