package lint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/gnolang/tlin/internal"
	tt "github.com/gnolang/tlin/internal/types"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const maxShowRecentFiles = 25

// LintEngine is the engine surface the lint package depends on.
// Concrete production engine: *internal.Engine.
type LintEngine interface {
	Run(filePath string) ([]tt.Issue, error)
	RunSource(source []byte) ([]tt.Issue, error)
	RunWithContext(ctx context.Context, filePath string) ([]tt.Issue, error)
	RunSourceWithContext(ctx context.Context, source []byte) ([]tt.Issue, error)

	// Deprecated: pass WithIgnoredRules to New instead.
	IgnoreRule(rule string)
	// Deprecated: pass WithIgnoredPaths to New instead.
	IgnorePath(path string)
}

// Option configures a lint engine at construction time.
type Option func(*newOptions)

type newOptions struct {
	logger       *zap.Logger
	ignoredRules []string
	ignoredPaths []string
}

// WithLogger overrides the engine's logger. Default zap.NewNop().
func WithLogger(logger *zap.Logger) Option {
	return func(o *newOptions) {
		if logger != nil {
			o.logger = logger
		}
	}
}

// WithIgnoredRules registers rule names to skip during dispatch.
func WithIgnoredRules(names ...string) Option {
	return func(o *newOptions) { o.ignoredRules = append(o.ignoredRules, names...) }
}

// WithIgnoredPaths registers glob patterns whose matching files are
// excluded from issue output.
func WithIgnoredPaths(paths ...string) Option {
	return func(o *newOptions) { o.ignoredPaths = append(o.ignoredPaths, paths...) }
}

// New constructs a lint engine from the configuration file at
// configurationPath, applying any options. A missing or unparseable
// config file is tolerated so callers can run with defaults.
func New(configurationPath string, opts ...Option) (*internal.Engine, error) {
	resolved := newOptions{logger: zap.NewNop()}
	for _, opt := range opts {
		opt(&resolved)
	}

	config, _ := parseConfigurationFile(configurationPath)
	engine, err := internal.NewEngine(config.Rules, internal.WithLogger(resolved.logger))
	if err != nil {
		return nil, err
	}
	for _, name := range resolved.ignoredRules {
		engine.IgnoreRule(name)
	}
	for _, path := range resolved.ignoredPaths {
		engine.IgnorePath(path)
	}
	return engine, nil
}

// NewWithLogger constructs a lint engine with the given logger.
//
// Deprecated: use New(path, WithLogger(logger)) instead.
func NewWithLogger(configurationPath string, logger *zap.Logger) (*internal.Engine, error) {
	return New(configurationPath, WithLogger(logger))
}

// ProcessFiles walks each path and returns the union of issues.
// Pass a nil observer to disable progress rendering — callers
// emitting JSON or other machine-readable output should do this
// to keep stdout free of ANSI escapes.
func ProcessFiles(
	ctx context.Context,
	logger *zap.Logger,
	engine LintEngine,
	paths []string,
	observer ProgressObserver,
) ([]tt.Issue, error) {
	var allIssues []tt.Issue
	for _, path := range paths {
		issues, err := ProcessPath(ctx, logger, engine, path, observer)
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

// ProcessPath processes a single file or directory path. Directories
// are walked concurrently up to GOMAXPROCS workers; cancellation via
// ctx is observed at file boundaries.
func ProcessPath(
	ctx context.Context,
	logger *zap.Logger,
	engine LintEngine,
	path string,
	observer ProgressObserver,
) ([]tt.Issue, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("error accessing %s: %w", path, err)
	}

	if info.IsDir() {
		var files []string
		err = filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if !fileInfo.IsDir() && hasDesiredExtension(filePath) {
				files = append(files, filePath)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory: %w", err)
		}

		return processDirectory(ctx, logger, engine, path, files, observer)
	}

	if !hasDesiredExtension(path) {
		return nil, nil
	}
	issues, err := engine.RunWithContext(ctx, path)
	if issues == nil {
		// Preserve the historical empty-slice (not nil) contract
		// callers and tests rely on for the file-not-directory path.
		issues = []tt.Issue{}
	}
	return issues, err
}

// fileResult is the channel payload between worker and collector.
type fileResult struct {
	filePath string
	issues   []tt.Issue
	err      error
}

// processDirectory handles concurrent processing of multiple files
// in a directory.
func processDirectory(
	ctx context.Context,
	logger *zap.Logger,
	engine LintEngine,
	path string,
	files []string,
	observer ProgressObserver,
) ([]tt.Issue, error) {
	if observer == nil {
		observer = nopObserver{}
	}
	observer.OnStart(len(files))
	defer observer.OnDone()

	resultChan := make(chan fileResult)
	maxWorkers := runtime.NumCPU()
	sem := make(chan struct{}, maxWorkers)

	var wg sync.WaitGroup

	allIssues := []tt.Issue{}
	var firstErr error

	collectorDone := make(chan struct{})
	go func() {
		for result := range resultChan {
			if result.err == nil {
				allIssues = append(allIssues, result.issues...)
			} else if firstErr == nil {
				firstErr = result.err
			}
			observer.OnFile(filepath.Base(result.filePath))
		}
		close(collectorDone)
	}()

	for _, filePath := range files {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(fp string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			select {
			case <-ctx.Done():
				resultChan <- fileResult{filePath: fp, err: ctx.Err()}
				return
			default:
			}

			fileIssues, err := engine.RunWithContext(ctx, fp)
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

	wg.Wait()
	close(resultChan)
	<-collectorDone

	if ctx.Err() != nil {
		return allIssues, ctx.Err()
	}
	return allIssues, firstErr
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

	f, err := os.Open(configurationPath)
	if err != nil {
		return config, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}
