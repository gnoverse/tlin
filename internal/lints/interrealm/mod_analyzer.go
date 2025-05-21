package interrealm

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PackageType represents the type of Gno package
type PackageType int

const (
	TypeUnknown PackageType = iota
	TypePackage             // Normal packages (p/)
	TypeRealm               // Realm packages (r/)
)

func (pt PackageType) String() string {
	switch pt {
	case TypePackage:
		return "p"
	case TypeRealm:
		return "r"
	default:
		return "unknown"
	}
}

var (
	// Regexp to match module declaration in gno.mod file
	moduleRegex = regexp.MustCompile(`^module\s+gno\.land/([pr])/(.+)$`)

	// Common errors
	ErrNoGnoModFile  = errors.New("no gno.mod file found")
	ErrInvalidGnoMod = errors.New("invalid gno.mod file format")
)

// PackageContext holds information about a Gno package
type PackageContext struct {
	Type       PackageType // p or r
	ModulePath string      // Full module path (e.g., gno.land/p/foo)
	Name       string      // Package name (e.g., foo)
}

// DeterminePackageTypeFromDir analyzes a directory to determine if it's a p or r package
// by looking for a `gno.mod` file and parsing its contents.
func DeterminePackageTypeFromDir(dirPath string) (*PackageContext, error) {
	gnoModPath := filepath.Join(dirPath, "gno.mod")

	// Check if `gno.mod` exists
	if _, err := os.Stat(gnoModPath); os.IsNotExist(err) {
		return nil, ErrNoGnoModFile
	}

	// Open and read `gno.mod` file
	file, err := os.Open(gnoModPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// look for module declaration
		matches := moduleRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			pkgType := TypeUnknown
			if matches[1] == "p" {
				pkgType = TypePackage
			} else if matches[1] == "r" {
				pkgType = TypeRealm
			}

			return &PackageContext{
				Type:       pkgType,
				// TODO: make prefix (chain-ID) configurable
				ModulePath: "gno.land/" + matches[1] + "/" + matches[2],
				Name:       filepath.Base(matches[2]),
			}, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, ErrInvalidGnoMod
}

// getPackageContextForFile determines the package context for a given file
// by finding the nearest gno.mod file in parent directories
func GetPackageContextForFile(filePath string) (*PackageContext, error) {
	dir := filepath.Dir(filePath)

	// Walk up directories until we find a gno.mod file or reach the root
	for {
		ctx, err := DeterminePackageTypeFromDir(dir)
		if err == nil {
			return ctx, nil
		}

		if err != ErrNoGnoModFile {
			return nil, err
		}

		// move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// reached the root directory without finding gno.mod
			return nil, ErrNoGnoModFile
		}
		dir = parent
	}
}

// PackageTypeCache provides caching for package type lookups
type PackageTypeCache struct {
	cache map[string]*PackageContext
}

// NewPackageTypeCache creates a new cache instance
func NewPackageTypeCache() *PackageTypeCache {
	return &PackageTypeCache{
		cache: make(map[string]*PackageContext),
	}
}

// GetPackageContext returns the package context for a file, using cache when possible
func (c *PackageTypeCache) GetPackageContext(filePath string) (*PackageContext, error) {
	dir := filepath.Dir(filePath)

	// Check if we have a cached result
	if ctx, ok := c.cache[dir]; ok {
		return ctx, nil
	}

	// Determine package type
	ctx, err := GetPackageContextForFile(filePath)
	if err != nil {
		return nil, err
	}

	// Cache the result
	c.cache[dir] = ctx
	return ctx, nil
}
