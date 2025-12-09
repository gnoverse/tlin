# RFC: Emit Format

## Summary

This lint rule ensures that `chain.Emit` calls (from the `runtime/chain` package) are properly formatted for better readability, especially when they contain multiple arguments.

## Motivation

The `chain.Emit` function (previously `std.Emit`, now deprecated) is commonly used for logging and event emission in Gno programs. When these calls contain multiple key-value pairs, they can become difficult to read if not properly formatted. This rule aims to improve code readability and maintainability by enforcing a consistent formatting style for `chain.Emit` calls.

## Proposed Implementation

The rule will check for `chain.Emit` calls and suggest a formatted version if the call is not properly structured. The formatting guidelines are:

1. The `chain.Emit` call should be multi-line if it has more than 3 arguments.
2. The event type (first argument) should be on its own line.
3. Each key-value pair should be on its own line.
4. The closing parenthesis should be on a new line.

### Rule Details

- **Rule ID**: emit-format
- **Severity**: warning
- **Category**: Style
- **Auto-fixable**: Yes

### Code Examples

#### Incorrect:

```go
chain.Emit(
    "OwnershipChange",
    "newOwner", newOwner.String(),
    "oldOwner", 
    oldOwner.String(),
    "anotherOwner", anotherOwner.String(),
)
```

#### Correct:

```go
chain.Emit(
    "OwnershipChange",                     // event type
    "newOwner", newOwner.String(),         // key-value pair
    "oldOwner", oldOwner.String(),         // key-value pair
    "anotherOwner", anotherOwner.String(), // key-value pair
)
```

## References

- [Effective gno](https://docs.gno.land/concepts/effective-gno#emit-gno-events-to-make-life-off-chain-easier)