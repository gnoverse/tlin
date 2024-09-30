# RFC: Unnecessary Slice Length

## Summary

This lint rule detects unnecessary use of the `len()` function in slice expressions, where the slice operation can be simplified without affecting the behavior of the code.

## Motivation

Go provides a concise syntax for slicing operations. However, developers sometimes use unnecessary `len()` calls when slicing to the end of a slice. This can make the code less readable and potentially less performant. By identifying these instances, we can help developers write more idiomatic and efficient Go code.

## Proposed Implementation

The implementation consists of a function `DetectUnnecessarySliceLength` that inspects the AST of a Go file to find slice expressions that can be simplified. The core logic includes:

1. Inspecting the AST for slice expressions.
2. Checking if the high bound of the slice is a `len()` call.
3. Verifying if the `len()` call argument matches the slice being operated on.
4. Generating appropriate suggestions and messages for different slice patterns.

### Rule Details

- **Rule ID**: simplify-slice-range
- **Severity**: warning
- **Category**: style
- **Auto-fixable**: Yes
- **Description**: Detects unnecessary use of `len()` in slice expressions that can be simplified.

### Code Examples

#### Incorrect:

```go
slice := []int{1, 2, 3, 4, 5}
subSlice1 := slice[:len(slice)]     // Unnecessary len()
subSlice2 := slice[2:len(slice)]    // Unnecessary len()
subSlice3 := slice[start:len(slice)] // Unnecessary len()
```

#### Correct:

```go
slice := []int{1, 2, 3, 4, 5}
subSlice1 := slice[:]     // Simplified
subSlice2 := slice[2:]    // Simplified
subSlice3 := slice[start:] // Simplified
```

## Open Questions

1. Should we consider more complex slice expressions, such as those involving arithmetic operations?
2. How should we handle cases where the slice and the `len()` argument are different variables but might refer to the same underlying array?
3. Should we provide a configuration option to ignore certain patterns or in specific contexts?
4. How do we ensure that this rule doesn't produce false positives in cases where the explicit use of `len()` might be preferred for readability or documentation purposes?
