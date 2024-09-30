# RFC: Defer Checks

## Summary

This set of lint rules checks for common issues and best practices related to the use of `defer` statements in Go code. It includes checks for deferring panics, nil functions, returns in deferred functions, and defers in loops.

## Motivation

The `defer` statement can be misused in ways that lead to unexpected behavior or performance issues. These lint rules aim to catch common mistakes and anti-patterns related to `defer`, preventing potential runtime errors.

## Proposed Implementation

1. Deferring panics
2. Deferring potentially nil functions
3. Using return statements in deferred functions
4. Using defer inside loops

### Rule Details

#### 1. Defer Panic

- **Rule ID**: defer-panic
- **Severity**: warning
- **Category**: best practices
- **Auto-fixable**: No
- **Description**: Detects when `panic` is called inside a deferred function.

#### 2. Defer Nil Function

- **Rule ID**: defer-nil-func
- **Severity**: error
- **Category**: bug prevention
- **Auto-fixable**: No
- **Description**: Detects when a potentially nil function is deferred.

#### 3. Return in Defer

- **Rule ID**: return-in-defer
- **Severity**: warning
- **Category**: best practices
- **Auto-fixable**: No
- **Description**: Detects when a return statement is used inside a deferred function.

#### 4. Defer in Loop

- **Rule ID**: defer-in-loop
- **Severity**: warning
- **Category**: performance
- **Auto-fixable**: No
- **Description**: Detects when defer is used inside a loop.

### Code Examples

#### Incorrect:

```go
func example() {
    defer panic("This is bad")  // defer-panic

    var f func()
    defer f()  // defer-nil-func

    defer func() {
        return  // return-in-defer
    }()

    for i := 0; i < 10; i++ {
        defer fmt.Println(i)  // defer-in-loop
    }
}
```

#### Correct:

```go
func example() {
    defer func() {
        if err := recover(); err != nil {
            // Handle the panic
        }
    }()

    f := func() { /* do something */ }
    defer f()

    defer func() {
        // Perform cleanup without using return
    }()

    cleanup := func(i int) {
        fmt.Println(i)
    }
    for i := 0; i < 10; i++ {
        i := i  // Create a new variable to capture the loop variable
        defer cleanup(i)
    }
}
```

## Alternatives

1. Runtime checks: Some of these issues could be caught at runtime, but this would not prevent the errors from occurring in production.

## Implementation Impact

- Positive: Improved code quality and prevention of common `defer`-related bugs.

## Open Questions

1. Should we consider adding severity levels for each rule that can be configured by users?
2. Are there any additional `defer`-related checks we should consider implementing?
3. Should we provide auto-fix for any of these rules?
