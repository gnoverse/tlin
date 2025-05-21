# Gno Interrealm Lint Rules Guide

This document explains the lint rules for interrealm programming patterns in the Gno language. These rules help developers use the interrealm programming model correctly.

## Implemented Lint Rules

### 1. crossing() Position Check (`crossing-position`)

The `crossing()` statement must be the first statement in the function body. Using it elsewhere will result in an error.

```go
// Correct usage
func PublicFunc() {
    crossing()
    // Rest of the code
}

// Incorrect usage
func PublicFunc() {
    x := 1 // Another statement comes first
    crossing() // Error: crossing() must be the first statement
}
```

### 2. Prohibition of crossing() in p packages (`crossing-in-p-package`)

`crossing()` can only be used in realm packages and cannot be used in p packages.

```go
// Usage in p package - Error
package myutils // gno.land/p/demo/myutils

func HelperFunc() {
    crossing() // Error: cannot be used in p packages
}
```

### 3. MsgCall Compatibility Check (`msgcall-compatibility`)

Public functions (functions that start with a capital letter) that can be called via MsgCall must be `crossing()` functions.

```go
// Incorrect usage - Cannot be called with MsgCall
func PublicFunction() {
    // No crossing()
}

// Correct usage
func PublicFunction() {
    crossing()
    // Rest of the code
}
```

### 4. Cross Call Rules (`missing-cross-call`, `incorrect-cross-call`)

When calling crossing functions from another realm, you must use the form `cross(fn)(...)`.

```go
// Incorrect usage
otherrealm.PublicCrossingFunc() // Error: must be wrapped with cross()

// Correct usage
cross(otherrealm.PublicCrossingFunc)()
```

Do not use `cross()` for non-crossing methods of another realm.

```go
// Incorrect usage
obj := otherrealm.GetObject()
cross(obj.NonCrossingMethod)() // Error: using cross() on a non-crossing method

// Correct usage
obj := otherrealm.GetObject()
obj.NonCrossingMethod()
```

## How to Use Lint Rules

You can configure lint rules using the `InterrealmRuleOptions` struct:

```go
options := InterrealmRuleOptions{
    Severity:                  tt.SeverityWarning,
    CheckCrossingDeclarations: true,
    CheckCrossingCalls:        true,
    CheckMsgCallCompatibility: true,
}

issues, err := DetectInterrealmRules(filename, fileAst, fileSet, options)
```

## Key Design Principles

1. Crossing functions declared in realm packages are for explicit realm crossing.
2. Methods should be non-crossing by default and should only modify data associated with the object they belong to.
3. Functions that are not crossing functions cannot be called with MsgCall.
4. Method calls on objects from another realm are considered implicit crossings.
5. When calling crossing functions within the same realm, both regular calls and cross calls are allowed.
