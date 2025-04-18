# tlin: Lint for gno

Advance Linter for go-like grammar languages.

[![CodeQL](https://github.com/gnoverse/tlin/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/gnoverse/tlin/actions)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/gnoverse/tlin/blob/main/LICENSE)

## Introduction

tlin is an linter designed for both [Go](https://go.dev/) and [gno](https://gno.land/) programming languages. It leverages the functionality of [golangci-lint](https://github.com/golangci/golangci-lint) as its main linting engine, providing powerful code analysis for go-like grammar languages.

Inspired by Rust's [clippy](https://github.com/rust-lang/rust-clippy), tlin aims to provide additional code improvement suggestions beyond the default golangci-lint rules.

## Features

- Support for Go (.go) and Gno (.gno) files
- Ability to add custom lint rules
- Additional code improvement suggestion, such as detecting unnecessary code
- Auto-fixing for some lint rules
- Cyclomatic complexity analysis

## Installation

To install tlin CLI, follow these steps:

```bash
git clone https://github.com/gnolang/tlin
cd tlin
go install ./cmd/tlin
```

## Usage

```bash
tlin <path>
```

Replace `<path>` with the file or directory path you want to analyze.

To check the current directory, run:

```bash
tlin .
```

## Configuration

tlin supports a configuration file (`.tlin.yaml`) to customize its behavior. You can generate a default configuration file by running:

```bash
tlin -init
```

This command will create a `.tlin.yaml` file in the current directory with the following content:

```yaml
# .tlin.yaml
name: tlin
rules:
```

You can customize the configuration file to enable or disable specific lint rules, set cyclomatic complexity thresholds, and more.

```yaml
# .tlin.yaml
name: tlin
rules:
  useless-break:
    severity: WARNING
  deprecated-function:
    severity: OFF
```

## Available Flags

tlin supports several flags to customize its behavior:

- `-timeout <duration>`: Set a timeout for the linter (default: 5m). Example: `-timeout 1m30s`
- `-cyclo`: Run cyclomatic complexity analysis
- `-threshold <int>`: Set cyclomatic complexity threshold (default: 10)
- `-ignore <rules>`: Comma-separated list of lint rules to ignore
- `-ignore-paths <paths>`: Comma-separated list of paths to ignore
- `-cfg`: Run control flow graph analysis
- `-func <name>`: Specify function name for CFG analysis
- `-fix`: Automatically fix issues
- `-dry-run`: Run in dry-run mode (show fixes without applying them)
- `-confidence <float>`: Set confidence threshold for auto-fixing (0.0 to 1.0, default: 0.75)
- `-o <path>`: Write output to a file instead of stdout
- `-json-output`: Output results in JSON format
- `-init`: Initialize a new tlin configuration file in the current directory
- `-c <path>`: Specify a custom configuration file

## Contributing

For detailed contribution guidelines, please refer to [CONTRIBUTING.md](CONTRIBUTING.md). We welcome all forms of contributions, including bug reports, feature requests, and pull requests.

## Credits

- [@GodDrinkTeJAVA](https://github.com/GodDrinkTeJAVA) - Project name (`tlin`) suggestion

## License

This project is distributed under the MIT License. See `LICENSE` for more information.
