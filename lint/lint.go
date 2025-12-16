package lint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/gnolang/tlin/internal"
	"github.com/gnolang/tlin/internal/lints"
	tt "github.com/gnolang/tlin/internal/types"
	"github.com/schollz/progressbar/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const maxShowRecentFiles = 25

type LintEngine interface {
	Run(filePath string) ([]tt.Issue, error)
	RunSource(source []byte) ([]tt.Issue, error)
	IgnoreRule(rule string)
	IgnorePath(path string)
}

// export the function NewEngine to be used in other packages
func New(rootDir string, source []byte, configurationPath string) (*internal.Engine, error) {
	// Do not return error here, because we want to continue the program
	config, _ := parseConfigurationFile(configurationPath)
	return internal.NewEngine(rootDir, source, config.Rules)
}

func ProcessSources(
	ctx context.Context,
	logger *zap.Logger,
	engine LintEngine,
	sources [][]byte,
	processor func(LintEngine, []byte) ([]tt.Issue, error),
) ([]tt.Issue, error) {
	var allIssues []tt.Issue
	for i, source := range sources {
		issues, err := processor(engine, source)
		if err != nil {
			if logger != nil {
				logger.Error("Error processing source", zap.Int("source", i), zap.Error(err))
			}
			return nil, err
		}
		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

func ProcessFiles(
	ctx context.Context,
	logger *zap.Logger,
	engine LintEngine,
	paths []string,
	processor func(LintEngine, string) ([]tt.Issue, error),
) ([]tt.Issue, error) {
	var allIssues []tt.Issue
	for _, path := range paths {
		issues, err := ProcessPath(ctx, logger, engine, path, processor)
		if err != nil {
			if logger != nil {
				logger.Error("Error processing path", zap.String("path", path), zap.Error(err))
			}
			return nil, err
		}
		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

// fileResult represents the result of processing a single file
type fileResult struct {
	filePath string
	issues   []tt.Issue
	err      error
}

// processDirectory handles concurrent processing of multiple files in a directory
func processDirectory(
	ctx context.Context,
	logger *zap.Logger,
	engine LintEngine,
	path string,
	files []string,
	processor func(LintEngine, string) ([]tt.Issue, error),
) ([]tt.Issue, error) {
	// Setup UI components
	var recentFilesMutex sync.Mutex
	recentFiles := make([]string, maxShowRecentFiles)

	// Make space for recent files display
	for range maxShowRecentFiles + 1 {
		fmt.Println()
	}
	fmt.Printf("\033[%dA", maxShowRecentFiles+1)

	bar := progressbar.NewOptions(len(files),
		progressbar.OptionSetDescription(path),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	updateRecentFiles := func(filename string) {
		recentFilesMutex.Lock()
		defer recentFilesMutex.Unlock()

		for j := maxShowRecentFiles - 1; j > 0; j-- {
			recentFiles[j] = recentFiles[j-1]
		}
		recentFiles[0] = filename

		fmt.Printf("\033[%dA", maxShowRecentFiles)
		for j := range recentFiles {
			if recentFiles[j] != "" {
				fmt.Printf("\033[2K\r%s\n", recentFiles[j])
			} else {
				fmt.Printf("\033[2K\r\n")
			}
		}
	}

	// Channels and worker pool
	resultChan := make(chan fileResult)
	maxWorkers := runtime.NumCPU()
	sem := make(chan struct{}, maxWorkers)

	// WaitGroup to track all goroutines
	var wg sync.WaitGroup

	// Track errors and progress
	allIssues := []tt.Issue{}
	var firstErr error
	var errorMutex sync.Mutex
	var processedFiles int

	// Start result collector goroutine
	var collectorDone = make(chan struct{})
	go func() {
		for result := range resultChan {
			processedFiles++
			if result.err == nil {
				allIssues = append(allIssues, result.issues...)
				updateRecentFiles(filepath.Base(result.filePath))
			} else {
				// Capture first error
				errorMutex.Lock()
				if firstErr == nil && result.err != ctx.Err() {
					firstErr = result.err
				}
				errorMutex.Unlock()
			}
			bar.Add(1)
		}
		close(collectorDone)
	}()

	// Process files
	var scheduledCount int
	for _, filePath := range files {
		// Early exit if context is cancelled
		if ctx.Err() != nil {
			// Still need to wait for running goroutines
			break
		}
		scheduledCount++

		wg.Add(1)
		sem <- struct{}{}
		go func(fp string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			// Check context before processing
			select {
			case <-ctx.Done():
				resultChan <- fileResult{filePath: fp, err: ctx.Err()}
				return
			default:
			}

			fileIssues, err := processor(engine, fp)
			if err != nil && logger != nil {
				logger.Error("Error processing file", zap.String("file", fp), zap.Error(err))
			}

			resultChan <- fileResult{
				filePath: fp,
				issues:   fileIssues,
				err:      err,
			}
		}(filePath)
	}

	// Wait for all workers to complete
	wg.Wait()
	close(resultChan)

	// Wait for collector to finish
	<-collectorDone

	// Update progress bar if not all files were processed
	if scheduledCount < len(files) {
		// Fill remaining progress
		for i := processedFiles; i < len(files); i++ {
			bar.Add(1)
		}
	}

	fmt.Println()

	// Return context error if cancelled, otherwise first processing error
	if ctx.Err() != nil {
		return allIssues, ctx.Err()
	}
	return allIssues, firstErr
}

func ProcessPath(
	ctx context.Context,
	logger *zap.Logger,
	engine LintEngine,
	path string,
	processor func(LintEngine, string) ([]tt.Issue, error),
) ([]tt.Issue, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("error accessing %s: %w", path, err)
	}

	issues := []tt.Issue{}
	if info.IsDir() {
		var files []string
		err = filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return nil // Skip files with errors
			}
			if !fileInfo.IsDir() && hasDesiredExtension(filePath) {
				files = append(files, filePath)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory: %w", err)
		}

		var dirErr error
		issues, dirErr = processDirectory(ctx, logger, engine, path, files, processor)
		if dirErr != nil {
			return issues, dirErr
		}
		return issues, nil
	} else if hasDesiredExtension(path) {
		fileIssues, err := processor(engine, path)
		if err != nil {
			return issues, err
		}
		issues = append(issues, fileIssues...)
	}

	return issues, nil
}

func ProcessCyclomaticComplexity(path string, threshold int) ([]tt.Issue, error) {
	return lints.DetectHighCyclomaticComplexity(path, threshold, tt.SeverityError)
}

func ProcessFile(engine LintEngine, filePath string) ([]tt.Issue, error) {
	return engine.Run(filePath)
}

func ProcessSource(engine LintEngine, source []byte) ([]tt.Issue, error) {
	return engine.RunSource(source)
}

var desiredExtensions = map[string]bool{
	".go":  true,
	".gno": true,
}

func hasDesiredExtension(path string) bool {
	return desiredExtensions[filepath.Ext(path)]
}

// Config represents the overall configuration with a name and a slice of rules.
type Config struct {
	Name  string                   `yaml:"name"`
	Rules map[string]tt.ConfigRule `yaml:"rules"`
}

func parseConfigurationFile(configurationPath string) (Config, error) {
	var config Config

	// Read the configuration file
	f, err := os.Open(configurationPath)
	if err != nil {
		return config, err
	}
	defer f.Close()

	// Parse the configuration file
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}
