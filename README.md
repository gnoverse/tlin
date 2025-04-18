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

- Requirements:
  - Go: 1.24 or higher
  - latest version of gno
  - GNU Make 3.81 or higher (for building)

To install tlin CLI, follow these steps:

1. Clone the repository

```bash
git clone https://github.com/gnolang/tlin
```

2. Move to the cloned directory

```bash
cd tlin
```

3. Install the CLI

```bash
go install ./cmd/tlin
```

that's it! You can now use the `tlin` command in your terminal.

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

## Adding Gno-Specific Lint Rules

Our linter allows addition of custom lint rules beyond the default golangci-lint rules. To add a new lint rule, follow these steps:

> ⚠️ Must update relevant tests if you have added a new rule or formatter.

1. Create an RFC (Request for Comments) document for your proposed lint rule:
   - Describe the purpose and motivation for the new rule. You can find template in [RFC](./docs/rfc/template.md)
   - Provide examples of code that the rule will flag.
   - Explain any potential edge cases or considerations.
   - Outline the proposed implementation approach.

2. Open a new issue in the tlin repository and attach your RFC document.

3. Wait for community feedback and maintainer approval. Be prepared to iterate on your RFC based on feedback.

4. Once the RFC is approved, proceed with the implementation:

   a. Create a new variable of type `LintRule` for your new rule:

   ```go
   NewRule = LintRule{severity: tt.SeverityWarning, check: lints.RunNewRule}

   func RunNewRule(filename string,  node *ast.File, fset *token.FileSet, severity tt.Severity) ([]types.Issue, error) {
       // Implement your lint rule logic here
       // return a slice of Issues and any error encountered
   }
   ```

   b. Add your rule to `allRules` mapping:

   ```go
   var allRules = ruleMap{
	"new-rule":               NewRule,
    }
   ```

5. (Optional) If your rule requires special formatting, create a new formatter in the `formatter` package:

   a. Create a new file (e.g., `formatter/new_rule.go`).

   b. Implement the `IssueFormatter` interface for your new rule:

   ```go
   type NewRuleFormatter struct{}

   func (f *NewRuleFormatter) Format(
       issue types.Issue,
       snippet *internal.SourceCode,
   ) string {
       // Implement formatting logic for new rule here.
   }
   ```

   c. Add the new formatter to the `GetFormatter` function in `formatter/fmt.go`.

   ```go
   func GetFormatter(rule string) IssueFormatter {
       switch rule {
       // ...
       case "new_rule": // Add your new rule here
           return &NewRuleFormatter{}
       default:
           return &DefaultFormatter{}
       }
   }
   ```

6. Add comprehensive tests for your new rule and formatter.

7. Update the documentation to include information about the new rule.

8. Submit a pull request with your implementation, tests, and documentation updates.

By following these steps, you can propose, discuss, and add new lint rules in a structured manner, ensuring they are properly integrated into the tlin project.

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

We welcome all forms of contributions, including bug reports, feature requests, and pull requests. Please feel free to open an issue or submit a pull request.

## Credits

- [@GodDrinkTeJAVA](https://github.com/GodDrinkTeJAVA) - Project name (`tlin`) suggestion

## License

This project is distributed under the MIT License. See `LICENSE` for more information.
