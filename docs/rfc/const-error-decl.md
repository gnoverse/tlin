# RFC: Constant Error Declaration Rule

## Summary

This rule detects and reports the use of `const` for declaring errors created with `errors.New()`. The rule enforces the usage of `var` for error declarations, preventing compilation issues that arise when using `const` for error values.

## Motivation

The result of `errors.New()` is not a constant value, even though the string passed to it is constant. Using `const` for such declarations can lead to compilation errors. This rule helps developers avoid this common mistake and use `var` for error declarations instead.

## Implementation

The rule will scan all `const` declarations in the code and check if any of them use `errors.New()`. If found, it will report an issue suggesting to use `var` instead.

### Rule Details

- **Rule ID**: const-error-declaration
- **Severity**: error
- **Category**: bug prevention, best practices
- **Auto-fixable**: Yes

### Code Examples

#### Incorrect

**Case 1** Single error in a single const declaration

```go
const err = errors.New("error")
```

Output

```plain
error: const-error-declaration
 --> foo.gno
  |
5 | const err = errors.New("error")
  | ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  = Constant declaration of errors.New() is not allowed
```

**Case 2** Multiple errors in a single const declaration

```go
const (
    err1 = errors.New("error1")
    err2 = errors.New("error2")
)
```

Output

```plain
error: const-error-declaration
 --> foo.gno
  |
5 | const (
6 |     err1 = errors.New("error")
7 |     err2 = errors.New("error2")
8 | )
  | ~~
  = Constant declaration of errors.New() is not allowed
```

#### Correct

**Case 1** Single error in a single var declaration

```go
var err = errors.New("error")
```

**Case 2** Multiple errors in a single var declaration

```go
var (
    err1 = errors.New("error1")
    err2 = errors.New("error2")
)
```

### Exceptions

There are no exceptions to this rule. All `const` declarations using `errors.New()` should be reported.

## Alternatives

An alternative approach could be to allow `const` declarations of errors but wrap them in a function call that returns an error interface. However, this approach is less idiomatic in Go and may lead to confusion.

## Implementation Impact

Positive impacts:

- Prevents compilation errors
- Encourages correct usage of error declarations
- Improves code consistency

Negative impacts:

- Potential migration costs could arise
