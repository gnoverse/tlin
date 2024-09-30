# RFC: Unnecessary Type Conversion

## Summary

This lint rule detects unnecessary type conversions in gno code, where the type of the expression being converted is identical to the target type of the conversion.

## Motivation

Unnecessary type conversions can untidy code and potentially impact performance due to the unnecessary cost of the conversion. By identifying these problems, we can help developers write cleaner code.

## Proposed Implementation

The implementation consists of a function `DetectUnnecessaryConversions` that uses Go's `types` package to perform type checking and analysis. The core logic includes:

1. Building a type information structure for the entire file.
2. Collecting variable declarations.
3. Inspecting the AST for type conversion expressions.
4. Checking if the conversion is unnecessary based on type identity.

### Rule Details

- **Rule ID**: unnecessary-type-conversion
- **Severity**: warning
- **Category**: style/performance
- **Auto-fixable**: Yes
- **Description**: Detects type conversions where the source and target types are identical.

### Detailed Explanation of Unnecessary Type Conversion Detection

The rule considers a type conversion unnecessary if all of the following conditions are met:

1. The expression is a function call-like syntax (`ast.CallExpr`) with exactly one argument.
2. The function being called is actually a type.
3. The type of the argument is identical to the target type of the conversion.
4. The argument is not an untyped constant.

The last condition is crucial because untyped constants can be implicitly converted, and explicit type conversions for these are sometimes necessary for clarity or to force a specific type.

The `isUntypedValue` function performs a deep inspection of expressions to determine if they result in an untyped value. It checks various cases including:

- Certain binary operations (e.g., shifts, comparisons)
- Unary operations
- Basic literals
- Identifiers referring to untyped constants or the nil value
- Calls to certain builtin functions

This comprehensive check ensures that the rule doesn't falsely flag necessary type conversions of untyped constants.

### Code Examples

#### Incorrect:

```go
var x int = 5
y := int(x)  // Unnecessary conversion

var s string = "hello"
t := string(s)  // Unnecessary conversion
```

#### Correct:

```go
var x int = 5
y := x  // No conversion needed

var s string = "hello"
t := s  // No conversion needed

const untyped = 5
typed := int(untyped)  // Necessary conversion from untyped to typed
```

## Implementation Impact

- Positive: Cleaner code.
- Negative: The lint process may be slower due to the need for full type checking.

## Open Questions

1. Should we extend the rule to check for unnecessary conversions in more complex expressions, such as function call arguments or return values?
2. How do we ensure good performance of this lint rule, given that it requires full type checking of the file?
