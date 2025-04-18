# Contribution Guide

Thank you for your interest in contributing to the project. This document provides guidance on how to contribute to the project.

## Recommended Reading

tlin uses the following technologies:

- [Go](https://go.dev/) - Backend language
- [gno](https://gno.land/) - Smart contract platform
- [golangci-lint](https://github.com/golangci/golangci-lint) - Linting engine

If you are not familiar with these technologies, we recommend reading their documentation.

## Prerequisites

To build the project, you need the following tools:

- Go 1.22 or higher
- Latest version of gno
- GNU Make 3.81 or higher

## Setting Up Development Environment

1. Clone the repository:

```bash
git clone https://github.com/gnolang/tlin
```

2. Move to the project directory:

```bash
cd tlin
```

3. Install the CLI:

```bash
go install ./cmd/tlin
```

## Development Commands

The project provides several useful Make commands to help with development:

### Building and Testing

- `make all`: Run tests and build the project
- `make build`: Build the project
- `make test`: Run all tests with race detection and random test order
- `make clean`: Clean build artifacts

### Code Quality

- `make lint`: Run golangci-lint
- `make fmt`: Format all Go files (or you can run `gofumpt -w -l .)
- `make install-linter`: Install golangci-lint (optional)

### Dependencies

- `make deps`: Install all dependencies
- `make run`: Build and run the project

## Adding New Lint Rules

tlin allows adding custom lint rules beyond the default golangci-lint rules. To add a new lint rule, follow these steps:

1. Create an RFC document for your proposed lint rule:
   - Describe the purpose and motivation for the new rule
   - Provide examples of code that the rule will flag
   - Explain potential edge cases or considerations
   - Outline the proposed implementation approach

2. Open a new issue in the tlin repository and attach your RFC document.

3. Wait for community feedback and maintainer approval.

4. Once the RFC is approved, proceed with implementation:

   a. Create a new variable of type `LintRule`:

   ```go
   NewRule = LintRule{severity: tt.SeverityWarning, check: lints.RunNewRule}

   func RunNewRule(filename string, node *ast.File, fset *token.FileSet, severity tt.Severity) ([]types.Issue, error) {
       // Implement your lint rule logic here
       // Return a slice of Issues and any error encountered
   }
   ```

   b. Add your rule to the `allRules` mapping:
   ```go
   var allRules = ruleMap{
       "new-rule": NewRule,
   }
   ```

5. (Optional) If your rule requires special formatting, create a new formatter in the `formatter` package.

6. Add comprehensive tests for your new rule and formatter.

7. Update the documentation to include information about the new rule.

8. Submit a pull request with your implementation, tests, and documentation updates.

## Running Tests

While we currently don't have many tests, we encourage you to write tests for your changes. To run the tests, execute:

```bash
make test
```

This command runs all tests with race detection and random test order.

## Before Submitting a Pull Request

Before submitting a pull request, ensure that:

1. Your changes pass all tests (`make test`)
2. The code is formatted using `gofumpt` or `go fmt` (`make fmt`)
3. All lint checks pass (`make lint`)

You can run all checks at once using:

```bash
make all
```

## License

By contributing to this project, you agree to license your contributions under the [MIT License](https://github.com/gnoverse/tlin/blob/main/LICENSE).
