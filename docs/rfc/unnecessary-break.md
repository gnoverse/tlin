# RFC: Unnecessary Break

## Summary

This lint rule detects useless `break` statements at the end of case clauses in `switch` or `select` statements in gno code.

## Motivation

In go, `switch` and `select` statements do not fall through to the next case by default, unlike in some other languages. This means that a `break` statement at the end of a case clause is unnecessary and can be safely removed, if gno also follows this behavior.

Detecting and removing these useless `break` statements can improve code readability and prevent confusion, especially for developers coming from languages where fall-through is the default behavior.

## Proposed Implementation

The implementation consists of a function `DetectUselessBreak` that inspects the AST of a Go file to find useless `break` statements. The core logic includes:

1. Traversing the AST to find `switch` and `select` statements.
2. For each case clause in these statements, checking if the last statement is an unnecessary `break`.

### Rule Details

- **Rule ID**: useless-break
- **Severity**: warning
- **Category**: style
- **Auto-fixable**: Yes
- **Description**: Detects useless break statements at the end of case clauses in switch or select statements.

### Detailed Explanation of Useless Break Detection

The rule considers a `break` statement useless if all of the following conditions are met:

1. It is the last statement in a case clause of a `switch` or `select` statement.
2. It is a simple `break` without a label (`breakStmt.Label == nil`).

The implementation uses `ast.Inspect` to traverse the AST and looks specifically for `*ast.SwitchStmt` and `*ast.SelectStmt` nodes. For each of these nodes, it iterates through the case clauses (`*ast.CaseClause` for switch and `*ast.CommClause` for select) and checks the last statement in each clause.

### Code Examples

#### Incorrect:

```go
switch x {
case 1:
    println("One")
    break  // Useless break
case 2:
    println("Two")
    break  // Useless break
default:
    println("Other")
    break  // Useless break
}
```

#### Correct:

```go
switch x {
case 1:
    println("One")
case 2:
    println("Two")
default:
    println("Other")
}
```

## Implementation Impact

- Positive: Improved code readability and reduced confusion.
- Negative: Minimal. Removing these statements does not change the behavior of the code.

## Open Questions

1. Should we extend the rule to detect useless `break` statements in `for` loops as well?
2. How should we handle cases where the `break` statement is preceded by a comment? Should we preserve the comment?
3. Should we provide a configuration option to ignore certain patterns or in specific contexts?
4. How do we ensure that this rule doesn't interfere with more complex control flow scenarios where a `break` might be used for clarity, even if it's technically unnecessary?
