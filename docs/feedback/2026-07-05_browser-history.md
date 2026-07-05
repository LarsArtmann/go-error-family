# Consumer Feedback — go-error-family

**From:** browser-history project (github.com/larsartmann/browser-history)
**Date:** 2026-07-05
**Version used:** v0.5.1
**Consumer:** Crush (AI assistant) + Lars

---

## What Works Great

### The 5-family taxonomy is the right abstraction

Rejection / Conflict / Transient / Corruption / Infrastructure maps cleanly to HTTP status codes (400/409/503/500/503). We built a single `mapDomainError()` that calls `errorfamily.Classify(err)` and produces correct status codes. This eliminated an entire class of "everything is 500" bugs.

### Copy-on-write sentinels are brilliant

`var ErrEmptyURL = errorfamily.NewRejection(...)` as a package-level sentinel, then `ErrEmptyURL.WithContext("field", "url")` at return sites — the sentinel stays pristine, each call gets its own context. This is the best error design pattern I've used in Go.

### `HandleError(err)` for CLI boundaries

`os.Exit(errorfamily.HandleError(err))` in `main.go` gives proper BSD exit codes (Transient→75, Rejection→65, etc.) with zero boilerplate. Clean, correct, done.

### `fmt.Errorf("...: %w", err)` preserves classification

`Classify()` walks the unwrap chain via `errors.AsType[Classified]`. This means `errorfamily.Wrap(originalRejection, Conflict, ...)` and `fmt.Errorf("context: %w", originalRejection)` both preserve the family correctly. Great interop with stdlib patterns.

---

## Pain Points

### 1. Dynamic errors can't be registered — only sentinels

**Problem:** `RegisterClassifications(map[error]Family)` works for sentinel errors (`sql.ErrNoRows`, `os.ErrNotExist`). But modernc.org/sqlite errors are **dynamic** — each error is a new `*sqlite.Error` instance with a numeric code. They can't be registered as sentinels because `errors.Is` compares by identity.

**Workaround:** We wrote a `classifyDBError(err)` helper that inspects error message strings:

```go
switch {
case strings.Contains(msg, "database is locked"):
    return errorfamily.Transient, "sqlite.locked"
case strings.Contains(msg, "constraint failed"):
    return errorfamily.Conflict, "sqlite.constraint"
}
```

String matching is fragile and embarrassing.

**Suggestion:** Add `RegisterClassifier(func(error) (Family, bool))` — a predicate-based registration that runs when sentinel matching fails. Consumers could register:

```go
errorfamily.RegisterClassifier(func(err error) (errorfamily.Family, bool) {
    var sqliteErr *sqlite.Error
    if errors.As(err, &sqliteErr) {
        switch sqliteErr.Code() {
        case 5, 6: return errorfamily.Transient, true  // BUSY, LOCKED
        case 19: return errorfamily.Conflict, true     // CONSTRAINT
        }
    }
    return errorfamily.Infrastructure, false
})
```

### 2. `Code()` vs `ErrorCode()` — two methods, same thing

**Problem:** `*errorfamily.Error` has both `Code() string` (line 78) and `ErrorCode() string` (line 53). They return the same field (`e.code`). I had to read the source to know which to use, and I picked wrong on the first try (tried `.Code` as a field, got a method value).

**Suggestion:** Deprecate one. `Code()` is the idiomatic Go name. `ErrorCode()` is redundant given the type is already `*Error`.

### 3. No `WrapRejectionf` / `WrapCorruptionf` shortcuts

**Problem:** The family shortcuts are asymmetric:

| Exists                                | Missing                                      |
| ------------------------------------- | -------------------------------------------- |
| `NewRejection(code, msg)`             | —                                            |
| `Newf(Rejection, code, fmt, args...)` | —                                            |
| `WrapRejection(err, code, msg)`       | `WrapRejectionf(err, code, fmt, args...)` ❌ |
| `WrapTransient(err, code, msg)`       | `WrapTransientf(err, code, fmt, args...)` ❌ |

I frequently need `WrapRejectionf(err, "config.validate", "field %s must not be empty", fieldName)` but have to write `Wrap(err, Rejection, code, fmt.Sprintf(...))` instead.

**Suggestion:** Add `Wrap{Family}f` variants for all 5 families.

### 4. `Classify()` returns Infrastructure for nil errors

**Problem:** `errorfamily.Classify(nil)` returns `Infrastructure` instead of panicking or returning a zero value. This silently masks nil-error bugs.

**Suggestion:** Either panic on nil (like `bytes.Buffer.Write(nil)` doesn't panic but `sync.Mutex.Lock()` on a nil mutex does) or document the behavior prominently.

---

## Minor Notes

- **`WithContextf(key, format, args...)`** — nice to have, used it for dynamic category values. Good API.
- **`HandleErrorDetailed(err)`** — returns `*HandleResult` with `SuggestedFix`. We don't use it yet but it's a great escape hatch for richer error reporting.
- The `go-error-family/constructors.go` file is well-organized — easy to find the right constructor.

---

## Summary

go-error-family is a **pleasure to use**. The taxonomy is right, the API is clean, and the copy-on-write sentinel pattern is the best Go error design I've encountered. The main gap is dynamic error classification (RegisterClassifier) — once that exists, it's essentially complete.

---

## Appendix: Resolution Status (2026-07-05)

### Pain Points

| #   | Item                                                | Status             | Resolution                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| --- | --------------------------------------------------- | ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| PP1 | Dynamic errors can't be registered — only sentinels | ✅ **DONE**        | Added `RegisterClassifier(func(error) (Family, bool))` — predicate-based classification for dynamic third-party errors. Stored lock-free behind `atomic.Pointer[[]Classifier]` (copy-on-write, same pattern as sentinels). Runs as pipeline step 5 (after sentinels, before the Transient default), in registration order. Includes package-level `RegisterClassifier`/`RegisterClassifiers` and `Registry.RegisterClassifier`/`RegisterClassifiers`. `Registry.Clone()` copies classifiers. The exact `*sqlite.Error` example from this feedback now works verbatim. |
| PP2 | `Code()` vs `ErrorCode()` — two methods, same thing | ✅ **DONE (docs)** | Doc comments on both methods now clarify the distinction: `ErrorCode()` is the canonical `Coded` interface contract (used via `errors.AsType` / `Code(err)`); `Code()` is an ergonomic accessor on `*Error` (sibling of `Family()` / `Message()`). Neither is deprecated — they serve different roles. The new `Code(err)` top-level helper is now the recommended one-liner.                                                                                                                                                                                         |
| PP3 | No `WrapRejectionf` / `WrapCorruptionf` shortcuts   | ✅ **DONE**        | Added all 5: `WrapRejectionf`, `WrapConflictf`, `WrapTransientf`, `WrapCorruptionf`, `WrapInfrastructuref`. All nil-safe (return nil if err is nil, matching `Wrap`).                                                                                                                                                                                                                                                                                                                                                                                                 |
| PP4 | `Classify()` returns Infrastructure for nil errors  | ✅ **RESOLVED**    | This was based on v0.5.0 behavior. In v0.5.1, `Classify(nil)` returns `Rejection` (not `Infrastructure`). The rationale: "nil error = caller's fault" — a programming error, not a system failure. This is documented in the `Classify` godoc and AGENTS.md "Surprising Behaviors" section.                                                                                                                                                                                                                                                                           |

### Summary Update

> "The main gap is dynamic error classification (RegisterClassifier) — once that exists, it's essentially complete."

**RegisterClassifier now exists.** The gap is closed. The string-matching workaround described in this feedback can be replaced with the exact predicate-based registration shown above.
