# RFC: Gno Realm Security Rules

## Summary

Four rules that statically detect the highest-yield security issues in gno realm code, derived from the checklist in [gnolang/gno#5880](https://github.com/gnolang/gno/pull/5880) (`docs/resources/gno-ai-contract-review.md`).

## Motivation

The gno AI contract review guide enumerates ten recurring realm vulnerabilities. Four of them are detectable with single-file AST analysis at high precision; catching them in the linter is cheaper than catching them in audit.

The remaining six (caller-supplied callback authority, canonical interface assertion, `/p/`-type mutation-method leaks, iterator-field exposure, address-parameter identity, Render input sanitization) require cross-package type information or taint analysis that tlin's single-file model does not provide, and are left to the audit tooling referenced in the guide.

## Proposed Implementation

### `unsafe-previous-realm` (guide case 9)

- **Severity**: error
- **Category**: bug prevention (security)
- **Auto-fixable**: no

`chain/runtime/unsafe.PreviousRealm()` bypasses the `cur.IsCurrent()` frame verification. If a file imports `chain/runtime/unsafe` **and** declares a crossing function (any function with a `realm`-typed parameter), every call through that import is flagged; if the import exists without a direct call, the import itself is flagged.

#### Incorrect

```go
import "chain/runtime/unsafe"

func Set(cur realm, key, value string) {
    caller := unsafe.PreviousRealm().Address()
}
```

#### Correct

```go
func Set(cur realm, key, value string) {
    if !cur.IsCurrent() { panic("spoofed realm") }
    caller := cur.Previous().Address()
}
```

#### Exceptions

Files with no crossing functions (non-crossing helpers, unmigrated realms) may use the package.

### `stored-realm` (guide case 6)

- **Severity**: error
- **Category**: bug prevention (security)
- **Auto-fixable**: no

`realm` values are ephemeral call-frame data; persisting one panics at attach time. Package-level `var` declarations and struct fields of type `realm` are flagged.

#### Incorrect

```go
var savedRealm realm

type record struct{ owner realm }
```

#### Correct

```go
var savedAddr address

func Save(cur realm) { savedAddr = cur.Previous().Address() }
```

### `payment-guard` (guide case 2)

- **Severity**: error
- **Category**: bug prevention (security)
- **Auto-fixable**: no

`IsUser()` accepts `maketx run` ephemeral realms, which can consume the origin-send envelope before calling the guarded function. In any file that calls `OriginSend`, every `IsUser()` call is flagged; the guard must be `IsUserCall()`.

#### Incorrect

```go
if !cur.Previous().IsUser() { panic("not a user") }
coins := banker.OriginSend()
```

#### Correct

```go
if !cur.Previous().IsUserCall() { panic("not a direct user call") }
coins := banker.OriginSend()
```

#### Exceptions

- Files that never call `OriginSend` may use `IsUser()` (e.g. display logic).
- Plain `.go` files are skipped — the rule matches bare method names, which could collide with unrelated Go code.

### `exported-mutable-pointer` (guide case 3)

- **Severity**: warning
- **Category**: bug prevention (security)
- **Auto-fixable**: no

Returning a pointer to realm state lets any caller invoke mutation methods on it; borrow rule #2 commits those writes under the realm's authority, and readonly taint does not block method dispatch. Exported functions in `.gno` files whose pointer-typed results return a package-level variable (or its address, or a field chain rooted in one) are flagged.

Warning (not error) because single-file analysis cannot see whether the pointed-to type actually has mutation methods.

#### Incorrect

```go
var store = avl.NewTree()

func GetStore() *avl.Tree { return store }
```

#### Correct

```go
func GetValue(key string) (any, bool) { return store.Get(key) }
```

#### Exceptions

- Plain `.go` files (exported pointer getters are a legitimate Go pattern).
- Constructors returning freshly allocated values.
- Unexported functions.

## Alternatives

A package-level `PackageRule` with full `go/types` loading could resolve the six skipped cases; rejected for now — the audit-pattern harness in gnolang/gno already covers them, and the cost/precision tradeoff of duplicating it here is poor.

## Implementation Impact

Positive: the four most mechanical checklist items run on every lint. Negative: `payment-guard` and `exported-mutable-pointer` are file-scope heuristics and can false-positive (suppress with `//nolint`).

## References

- [gnolang/gno#5880 — AI contract review guide](https://github.com/gnolang/gno/pull/5880)
- `docs/resources/gno-security-guide.md` in gnolang/gno
