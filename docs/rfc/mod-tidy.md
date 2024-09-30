# RFC: Gno Mod Tidy

## Summary

This lint rule checks for inconsistencies between the packages imported in Gno source files and those declared in the `gno.mod` file. It helps ensure that the `gno.mod` file accurately reflects the project's dependencies.

## Motivation

In Gno projects, the `gno.mod` file serves a similar purpose to Go's `go.mod` file, declaring the project's dependencies. Keeping this file in sync with the actual imports used in the source code is crucial for proper dependency management. This lint rule aims to catch discrepancies early, promoting better dependency hygiene and preventing potential build or runtime issues.

## Proposed Implementation

The implementation consists of a function `DetectMissingModPackage` that performs the following checks:

1. Extracts imports from the Gno source files.
2. Extracts declared packages from the `gno.mod` file.
3. Compares the two sets to identify:
   a. Packages declared in `gno.mod` but not imported in the source.
   b. Packages imported in the source but not declared in `gno.mod`.

### Rule Details

- **Rule ID**: gno-mod-tidy
- **Severity**: warning
- **Category**: dependency management
- **Auto-fixable**: No (but suggests running `gno mod tidy`)
- **Description**: Detects inconsistencies between imported packages and those declared in `gno.mod`.

### Code Examples

#### Incorrect:

```go
// main.gno
package main

import (
    "gno.land/p/demo/avl"
)

// gno.mod
module example.com/mymodule

gno.land/p/demo/avl v0.0.0-latest
gno.land/p/demo/foo v0.0.0-latest  // Unused package
```

#### Correct:

```go
// main.gno
package main

import (
    "gno.land/p/demo/avl"
)

// gno.mod
module example.com/mymodule

gno.land/p/demo/avl v0.0.0-latest
```

## Implementation Impact

- Positive: Improved dependency management and prevention of potential runtime errors due to missing or unnecessary dependencies.
