# RFC: Format Without Verb

## Summary

Detect calls to formatting functions (e.g., `ufmt.Sprintf`, `ufmt.Printf`) where the format string contains no formatting verbs, so the call is unnecessary or hiding a bug because no interpolation happens.

## Motivation

Using `ufmt`/`fmt` formatting helpers without verbs just returns the literal string while paying the formatting cost. When arguments are supplied but verbs are missing, values are silently ignored and the runtime emits `%!(EXTRA ...)` noise. Flagging these cases encourages `ufmt.Sprint`/`ufmt.Fprint` or raw string usage and helps surface mistakes early.

## Proposed Implementation

Walk call expressions, detect targets that are formatting-style functions from `gno.land/p/nt/ufmt` (and `fmt` in `*_test.gno`), and analyze their format strings for real verbs. Emit a diagnostic when no verbs are found.

### Rule Details

- **Rule ID**: format-without-verb
- **Severity**: warning
- **Category**: style/performance
- **Auto-fixable**: Yes (for zero-arg cases where we can safely switch to `Sprint`/`Fprint` or a literal)

### Algorithm

1. **Candidate discovery**  
   - Inspect `ast.CallExpr` nodes.  
   - Resolve the callee to a function from package path `gno.land/p/nt/ufmt`; permit `fmt` only when analyzing `*_test.gno`.  
   - Limit to format-style functions: `Sprintf`, `Printf`, `Fprintf`, `Errorf`. (Additional formatters can be added to this set if introduced.)
2. **Format argument selection**  
   - Determine which argument holds the format string: index `0` for `Sprintf`/`Printf`/`Errorf`, index `1` for `Fprintf`.  
   - Require a constant string (use `types.Info.Types` / `types.ExprString` to unwrap string literals and string-typed constants). Skip non-constant formats to avoid false positives.
3. **Verb detection**  
   - Unquote the string and scan characters. Treat `%%` as a literal percent and not a verb.  
   - On `%`, consume flags (`+ - # 0 space`), arg indexes (`[n]`), width/precision (`*` or digits with optional `.`), and the verb rune.  
   - Count a verb when a valid verb rune is reached (use the same verb set as `fmt`: `vTtbcdoqxXUeEfFgGsp`). If no verbs are found by the end of the string, mark the call as violating.
4. **Diagnostic and fixes**  
   - Message example: `"format string has no verbs; use ufmt.Sprint/ufmt.Fprint or a literal"`.  
   - Provide quick fixes when safe:  
     - `Sprintf("literal")` → replace with the literal (or `ufmt.Sprint("literal")` if keeping call style is preferred).  
     - `Printf("literal")` → `ufmt.Print("literal")` (and `Fprintf` → `ufmt.Fprint`).  
   - When extra args exist but no verbs, still emit a warning (potential bug) but omit auto-fix unless we can drop formatting entirely without changing semantics.

### Code Examples

#### Incorrect

```go
// gno
import "gno.land/p/nt/ufmt"

_ = ufmt.Sprintf("dummy")              // no verbs, unnecessary
ufmt.Printf("status ok")               // should be Print/Println
ufmt.Fprintf(w, "ready")               // should be Fprint/Fprintln
ufmt.Sprintf("value", x)               // args ignored, likely bug
```

```go
// foo_test.gno
import "fmt"

_ = fmt.Sprintf("only text")           // same rule allowed in tests
```

#### Correct

```go
ufmt.Sprintf("value: %s", name)        // has verb
ufmt.Print("status ok")                // non-format call
ufmt.Fprintf(w, "ratio: %.2f%%", r)    // verbs present (%% treated as literal %)

msg := "dynamic %s"                    // non-constant; skip lint to avoid FP
_ = ufmt.Sprintf(msg, arg)
```

### Test Coverage (must add)

- Detect `ufmt.Sprintf("literal")`.
- Detect `ufmt.Sprintf(msgConst)` where `msgConst` is a string constant with no verbs.
- Detect `ufmt.Sprintf("literal", arg)` (extra args).
- Detect `ufmt.Printf("literal")`.
- Detect `ufmt.Fprintf(w, "literal")`.
- Detect `fmt.Sprintf("literal")` inside `*_test.gno`.
- Detect `ufmt.Fprintf(w, "percent: %%d")` (only uses `%%`, still no verbs).
- Accept `ufmt.Sprintf("value: %s", v)`.
- Accept dynamic format: `fmt.Sprintf(msg, x)` where `msg` is non-constant.
