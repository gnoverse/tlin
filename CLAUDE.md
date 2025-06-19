# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Development
- `make build` - Build the tlin binary
- `make install` - Install tlin binary via go install ./cmd/tlin
- `make test` - Run all tests with race detection and shuffling
- `make lint` - Run golangci-lint
- `make fmt` - Format all Go files
- `make clean` - Clean build artifacts

### Running Tests
- `go test ./...` - Run all tests
- `go test -v ./internal/lints -run TestSpecificFunction` - Run a specific test
- `make test` - Run tests with race detection and shuffle enabled

### Running tlin
- `tlin .` - Lint current directory
- `tlin <path>` - Lint specific file or directory
- `tlin -fix .` - Auto-fix issues with default confidence (0.75)
- `tlin -json-output .` - Output results in JSON format
- `tlin -init` - Create .tlin.yaml configuration file

## Architecture

### Core Components
The linter follows a pipeline architecture:

1. **Entry Point** (`cmd/tlin/main.go`): CLI interface that handles flags, configuration loading, and orchestrates the linting process.

2. **Engine** (`internal/engine.go`): Core engine that:
   - Manages enabled/disabled rules based on configuration
   - Coordinates file processing across lint rules
   - Handles issue collection and formatting

3. **Lint Processing** (`lint/lint.go`): 
   - `ProcessFiles`: Main function that processes multiple files in parallel
   - `ProcessPath`: Recursively processes directories or single files
   - Handles progress tracking and parallel execution

4. **Rule System** (`internal/rule_set.go`): 
   - All lint rules implement the same interface
   - Rules return `[]types.Issue` for found problems
   - Each rule has a severity level (Error, Warning, Info)

5. **Formatters** (`formatter/`): 
   - Different formatters for different rule types
   - Default formatter handles most rules
   - Special formatters for complex rules (e.g., emit-format, early-return)

### Available Lint Rules
- `golangci-lint` - Wrapper for golangci-lint tool
- `simplify-slice-range` - Detects unnecessary slice length in range loops
- `unnecessary-type-conversion` - Finds redundant type conversions
- `cycle-detection` - Detects cyclic dependencies
- `emit-format` - Checks gno emit statement formatting
- `useless-break` - Finds unnecessary break statements
- `early-return-opportunity` - Suggests early returns to reduce nesting
- `const-error-declaration` - Ensures errors are declared as variables, not constants
- `repeated-regex-compilation` - Detects regex compilation in loops
- `deprecated` - Warns about deprecated function usage
- `unused-package` - Gno-specific rule for unused package imports
- `simplify-for-range` - Simplifies for loops that can use range

### Key Features
- **Parallel Processing**: Files are processed concurrently for performance
- **Auto-fix Support**: Many rules support automatic fixing with confidence thresholds
- **CFG Analysis**: Control Flow Graph analysis for complex checks
- **Configuration**: `.tlin.yaml` for rule customization and severity settings
- **nolint Comments**: Supports `//nolint:rule-name` to suppress specific rules

### Adding New Rules
1. Create RFC in `docs/rfc/` following the template
2. Implement rule function in `internal/lints/`
3. Add rule to `allRules` map in `internal/rule_set.go`
4. Create custom formatter in `formatter/` if needed
5. Add tests in `testdata/` directory
6. Update `GetFormatter` function if custom formatter was created