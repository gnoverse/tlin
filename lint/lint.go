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
	config, err := parseConfigurationFile(configurationPath)
	if err != nil {
		return nil, err
	}

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

	var issues []tt.Issue
	if info.IsDir() {
		var files []string
		filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
			if !fileInfo.IsDir() && hasDesiredExtension(filePath) {
				files = append(files, filePath)
			}
			return nil
		})

		// mutex for recent files
		var recentFilesMutex sync.Mutex
		recentFiles := make([]string, maxShowRecentFiles)

		// make space for recent files
		for range maxShowRecentFiles + 1 {
			fmt.Println()
		}
		fmt.Printf("\033[%dA", maxShowRecentFiles+1)

		// channels for results and errors
		resultChan := make(chan []tt.Issue, len(files))
		errorChan := make(chan error, len(files))

		// limit the number of workers
		maxWorkers := runtime.NumCPU()
		sem := make(chan struct{}, maxWorkers)

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

		// update recent files
		updateRecentFiles := func(filename string) {
			recentFilesMutex.Lock()
			defer recentFilesMutex.Unlock()

			// update the list
			for j := maxShowRecentFiles - 1; j > 0; j-- {
				recentFiles[j] = recentFiles[j-1]
			}
			recentFiles[0] = filename

			// move the cursor up
			fmt.Printf("\033[%dA", maxShowRecentFiles)

			// print the list
			for j := range recentFiles {
				if recentFiles[j] != "" {
					// \033[2k: clear the line
					// \r: move the cursor to the beginning of the line
					fmt.Printf("\033[2K\r%s\n", recentFiles[j])
				} else {
					fmt.Printf("\033[2K\r\n")
				}
			}
		}

		// for each file, run a goroutine
		for _, filePath := range files {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				sem <- struct{}{}
				go func(fp string) {
					defer func() { <-sem }()

					// show the start of file processing
					updateRecentFiles(filepath.Base(fp))

					fileIssues, err := processor(engine, fp)
					if err != nil {
						if logger != nil {
							logger.Error("Error processing file", zap.String("file", fp), zap.Error(err))
						}
						errorChan <- err
						resultChan <- nil
					} else {
						resultChan <- fileIssues
						errorChan <- nil
					}
					bar.Add(1)
				}(filePath)
			}
		}

		// collect all results
		for range files {
			if err := <-errorChan; err != nil {
				continue
			}
			if result := <-resultChan; result != nil {
				issues = append(issues, result...)
			}
		}

		fmt.Println()
		return issues, nil
	} else if hasDesiredExtension(path) {
		fileIssues, err := processor(engine, path)
		if err != nil {
			return nil, err
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
