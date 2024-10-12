package lint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gnolang/tlin/internal"
	"github.com/gnolang/tlin/internal/lints"
	tt "github.com/gnolang/tlin/internal/types"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type LintEngine interface {
	Run(filePath string) ([]tt.Issue, error)
	RunSource(source []byte) ([]tt.Issue, error)
	IgnoreRule(rule string)
}

// export the function NewEngine to be used in other packages
func New(rootDir string, source []byte, configurationPath string) (*internal.Engine, error) {
	config, err := parseConfigurationFile(configurationPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing configuration file: %w", err)
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
	_ context.Context,
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
		err = filepath.Walk(path, func(filePath string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !fileInfo.IsDir() && hasDesiredExtension(filePath) {
				fileIssues, err := processor(engine, filePath)
				if err != nil && logger != nil {
					logger.Error("Error processing file", zap.String("file", filePath), zap.Error(err))
				} else {
					issues = append(issues, fileIssues...)
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %w", path, err)
		}
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
	return lints.DetectHighCyclomaticComplexity(path, threshold)
}

func ProcessFile(engine LintEngine, filePath string) ([]tt.Issue, error) {
	return engine.Run(filePath)
}

func ProcessSource(engine LintEngine, source []byte) ([]tt.Issue, error) {
	return engine.RunSource(source)
}

func hasDesiredExtension(path string) bool {
	return filepath.Ext(path) == ".go" || filepath.Ext(path) == ".gno"
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
