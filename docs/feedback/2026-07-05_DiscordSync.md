# go-error-family — Consumer Feedback (DiscordSync)

**Consumer:** [DiscordSync](https://github.com/LarsArtmann/DiscordSync) — Discord backup bot
**Version used:** v0.5.1
**Usage depth:** Heavy — all 5 families, `HandleError` CLI boundary, `MapError` (via cqrs-htmx), `RegisterClassifications`, consumer `errkit` wrappers, `Classify`-based retry decisions
**Date:** 2026-07-05

---

## What Works Superbly

### 1. The 5-family taxonomy is the right abstraction

Rejection / Conflict / Transient / Corruption / Infrastructure maps cleanly to both HTTP status codes AND retry decisions. Every error has exactly one correct family, and the family tells you everything you need: retry? exit code? HTTP status? audience? tone? No ambiguity.

### 2. `Classify()` fail-open default is correct

Unknown errors → Transient (retry) is the right default. It means adopting the library incrementally is safe — unclassified errors don't crash, they just get retried.

### 3. `HandleError` at the CLI boundary is genius

```go
os.Exit(errorfamily.HandleError(err))
```

One line. Correct BSD exit codes. Structured output. Done. This replaced 20 lines of `switch err := err.(type)` boilerplate.

### 4. `errors.Is` matching on code+family only

This is a deliberate, documented design choice that prevents subtle bugs where two errors with the same code but different messages are treated as different. Smart.

### 5. `Wrap(nil, ...) returns nil`

The nil-safety of all Wrap constructors eliminates an entire class of nil-dereference bugs at call sites. The consumer `errkit` wrappers build on this directly.

### 6. `RegisterClassifications` map API

```go
errorfamily.RegisterClassifications(map[error]errorfamily.Family{
    sql.ErrNoRows: errorfamily.Rejection,
    // ...
})
```

One call, registers everything. Uses `errors.Is` chain-walking so wrapped variants match. Idempotent. Perfect.

---

## What's Painful

### 1. No `Newf` with context — you need 3 chained calls

```go
// What I want:
err := errorfamily.NewTransientf("db.timeout", "query took %s on %s", duration, host).
    WithContext("query", sql)

// What exists (Newf exists but no context chaining on the constructor):
err := errorfamily.NewTransientf("db.timeout", "query took %s", duration).
    WithContext("host", host).
    WithContext("query", sql)
```

`Newf` / `Wrapf` exist but don't include context. Every call site that needs context ends up being 3+ lines. This is why we built `errkit` — to flatten this into one call.

**Suggestion:** Add variadic context pairs to the constructors:

```go
func NewTransient(code, message string, context ...string) *Error
func NewTransientf(code, format string, args ...any, context ...string) *Error // can't do this — Go can't distinguish
```

Better: a builder pattern or accept that `errkit`-style helpers are the answer and document them in the skill.

### 2. `RegisterClassification` is package-level (global state)

The DefaultRegistry is global mutable state. `RegisterClassifications` mutates it at runtime. This makes testing harder — one test's registration leaks into another. The `Registry` type exists for isolation, but the package-level functions (which are the ergonomic API) use DefaultRegistry.

**Impact:** We call `RegisterClassifications()` at startup (not in `init()`, because the project bans `init`). If we ever need test-specific registrations, we'd need to construct a custom Registry and thread it through — which defeats the package-level convenience.

**Suggestion:** This is a known tradeoff (global convenience vs testability). Document it more prominently in the skill. Consider whether `RegisterClassification` should be idempotent with a "frozen" flag after first read.

### 3. The `*Error` type has too many methods

`Error()`, `Code()`, `ErrorCode()`, `Family()`, `ErrorFamily()`, `Message()`, `Cause()`, `Timestamp()`, `WithContext()`, `WithCause()`, `WithTimestamp()`, `HasContext()`, `ContextValue()`, `Summary()`, `ErrorContext()`, `IsRetryable()`...

The interface methods (`ErrorCode()`, `ErrorFamily()`, `ErrorContext()`, `IsRetryable()`) AND the direct accessors (`Code()`, `Family()`, `ContextValue()`) exist for the same data. The consumer has to learn which to use.

**Suggestion:** The direct accessors are for `*Error`, the interface methods are for the interfaces. Document this distinction clearly. The skill does a good job, but the godoc could be more explicit.

### 4. `Classify(nil)` returns Rejection

This is documented ("nil error = caller's fault"), but it bit us. Code that does `Classify(result.err)` where `result.err` might be nil gets Rejection instead of a nil check. The fail-open philosophy is inconsistent here — nil should be Transient (harmless) or panic, not Rejection (blames the user).

**Suggestion:** Consider returning `Infrastructure` for nil (programming error, not user fault), or document this gotcha more prominently. At minimum, the skill should warn about it.

### 5. No `RegisterClassification` for error _types_ (only sentinels)

We can register `sql.ErrNoRows` (a specific error value), but we can't register "all `*os.PathError` with `ENOSPC` cause". We have to check this manually in our code (`content/errors.go` has a 15-line function to detect disk-full across 3 different error type wrappers).

**Suggestion:** Add `RegisterClassificationType[T error](family Family)` that uses `errors.As` instead of `errors.Is`. This would let consumers register "all `*fs.PathError` are Infrastructure" without enumerating every sentinel.

### 6. The `diagnose/` module is overkill for most consumers

The diagnostic rules (Filesystem, Network, Git, Postgres) are designed for ops debugging scenarios. For an application like DiscordSync, the failure modes are Discord API errors and SQLite lock contention — not "is postgres running" or "is the filesystem healthy".

**This is NOT a criticism** — it's correctly a separate module. But the skill mentions it, which creates FOMO ("should I be using this?").

**Suggestion:** Add a one-liner to the skill: "Skip diagnose/ unless your app's failure modes include infrastructure health issues (database down, disk full, network unreachable). Most apps don't need it."

---

## What's Missing

### 1. A `TemplateRegistry` for user-facing CLI messages

`HandleError` prints the error message + suggested fix. But there's no way to register a user-friendly message template for a code WITHOUT using the `HandleConfig.TemplateOverride` map every time.

We want: at startup, register `"db.migrate"` → `{What: "Database migration failed", Fix: "Check that the database file is writable"}`. Then every `HandleError(err)` call that encounters a `db.migrate` error automatically uses the template.

`Registry.RegisterTemplate` exists but it's on the injectable Registry, not DefaultRegistry (or is it? The skill is unclear here). Clarify.

### 2. Batch/partial-success pattern guidance

The skill says "use `errors.Join`" but doesn't show the canonical pattern. We had to design our own:

```go
for _, item := range items {
    if err := process(item); err != nil {
        failures = append(failures, err)
    }
}
if len(failures) > 0 {
    return errors.Join(failures...)  // Classify picks worst family
}
```

This works, but `Classify(errors.Join(...))` returns the first non-Transient family, which might not be the "worst". Consider whether multi-error classification should pick by severity (Corruption > Infrastructure > Conflict > Rejection > Transient) rather than first-match.

### 3. HTTP integration without cqrs-htmx

`Family.HTTPStatus()` is great, but there's no standalone HTTP error response writer. Consumers not using cqrs-htmx have to write their own:

```go
func writeError(w http.ResponseWriter, err error) {
    family := errorfamily.Classify(err)
    status := family.HTTPStatus()
    // ... format response, extract code, etc.
}
```

We use `cqrshtmx.MapError(err)` which does this, but it's in cqrs-htmx, not go-error-family.

**Suggestion:** Add a `httperror` subpackage: `httperror.Write(w, err)`, `httperror.Status(err) int`, `httperror.Response(err) (status int, code string, message string)`.

---

## Skill Feedback

The skill file (`/home/lars/projects/go-error-family/SKILL.md`) is **excellent overall**. Specific feedback:

### Good

- The 5-family table with Retry/Exit/Audience/Tone columns is the single most useful reference
- Quick API Reference section is comprehensive and accurate
- Classification precedence (first match wins) is clearly documented
- Gotchas table at the end catches real bugs

### Could Improve

- **No guidance on when to use `New*` vs `Wrap*`** — consumers constantly face this choice. Add: "Use `New*` when creating a new error from scratch. Use `Wrap*` when you have an underlying error to chain."
- **The `Newf`/`Wrapf` constructors aren't mentioned prominently enough** — I discovered them late. They're the right choice for messages with format verbs.
- **No example of the consumer `errkit` pattern** — every non-trivial consumer will build nil-safe wrappers with context. Show the pattern.
- **`RegisterClassifications` map variant isn't in the skill** — only the singular `RegisterClassification` is shown. The map variant is more ergonomic for bulk registration.
- **The `ParseFamily` default-to-Transient behavior** should be in the gotchas table (it's mentioned in the classify section but easy to miss).

---

## Summary Scorecard

| Dimension             | Score    | Notes                                                        |
| --------------------- | -------- | ------------------------------------------------------------ |
| API design            | 9/10     | 5-family taxonomy is excellent; minor method proliferation   |
| Documentation (skill) | 8/10     | Comprehensive; missing some patterns consumers need          |
| Ease of adoption      | 9/10     | Fail-open default makes incremental adoption safe            |
| Testability           | 7/10     | Global DefaultRegistry complicates test isolation            |
| HTTP integration      | 6/10     | `Family.HTTPStatus()` is great but no standalone HTTP helper |
| CLI integration       | 10/10    | `HandleError` is perfect                                     |
| Overall               | **8/10** | Best error library I've used in Go                           |

---

## Appendix: Resolution Status (2026-07-05)

> **Note:** This feedback document was missed during the initial feedback-implementation session. It was discovered during a self-review. Items below are marked with their current status as of 2026-07-23.

### What's Painful

| #   | Item                                                            | Status                        | Resolution                                                                                                                                                                                                                                                                 |
| --- | --------------------------------------------------------------- | ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| D1  | No `Newf` with context — need 3 chained calls                   | ⏳ **NOT STARTED**            | Constructor context ergonomics still require `.WithContext().WithContext()` chains. A builder pattern or functional options could help. This is a design decision (variadic context vs builder vs `errkit`-style helpers).                                                 |
| D2  | `RegisterClassification` is package-level (global state)        | ⏳ **NOT STARTED**            | The `Registry` type exists for isolation, but the package-level convenience functions use `DefaultRegistry`. No "frozen" flag exists yet. Needs better docs at minimum.                                                                                                    |
| D3  | `*Error` has too many methods (Code vs ErrorCode etc.)          | ✅ **PARTIALLY DONE**         | Doc comments now clarify: `ErrorCode()` = interface contract, `Code()` = ergonomic accessor. Broader method-proliferation concern (deprecating accessors in favor of interface-only) deferred to a future major version.                                                   |
| D4  | `Classify(nil)` returns Rejection (inconsistent with fail-open) | ⏳ **DESIGN DECISION NEEDED** | DiscordSync argues nil should be Infrastructure (programming error) or Transient (fail-open). Current behavior: Rejection. This contradicts the fail-open philosophy for unknown errors. Needs a deliberate product decision — changing it is a breaking change.           |
| D5  | No `RegisterClassification` for error _types_ (only sentinels)  | ✅ **PARTIALLY DONE**         | `RegisterClassifier(func(error) (Family, bool))` was added — it solves this via predicate functions. Consumers can `errors.As` inside the closure. A dedicated `RegisterClassificationType[T error](family Family)` generic would be more ergonomic but hasn't been added. |
| D6  | diagnose/ is overkill — skill should say "skip if not needed"   | ✅ **DONE**        | SKILL.md now says: "diagnose/ and agent/ are separate modules — Opt-in: skip them unless you need infrastructure debugging or AI analysis."                                                                                                                                |

### What's Missing

| #   | Item                                                                   | Status             | Resolution                                                                                                                                                                                                                                       |
| --- | ---------------------------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| D7  | `TemplateRegistry` confusion (is RegisterTemplate on DefaultRegistry?) | ⏳ **NOT STARTED** | The answer is YES — `RegisterTemplate` delegates to `DefaultRegistry`. But consumers are confused. SKILL.md should clarify this explicitly. The new `TemplateForCode(code)` helper also helps — it proves templates are on the default registry. |
| D8  | Batch/partial-success canonical example                                | ⏳ **NOT STARTED** | No canonical code example in SKILL.md. Note: the multi-error classification now picks **worst severity** (not "first non-Transient" as this feedback states) — that was fixed in v0.5.0.                                                         |
| D9  | HTTP integration without cqrs-htmx                                     | ✅ **DONE**        | Added `HTTPHandler(fn) http.Handler` and `HTTPStatus(err) int` in `http.go`. The standalone HTTP helper this feedback requested now exists.                                                                                                      |

### Skill Feedback

| #   | Item                                               | Status             | Resolution                                                                                              |
| --- | -------------------------------------------------- | ------------------ | ------------------------------------------------------------------------------------------------------- |
| —   | No guidance on `New*` vs `Wrap*`                   | ⏳ **NOT STARTED** | Needs: "Use `New*` when creating from scratch. Use `Wrap*` when you have an underlying error to chain." |
| —   | `Newf`/`Wrapf` not prominent enough                | ⏳ **NOT STARTED** | Also: new `Wrap{Family}f` variants added but not yet shown in skill examples.                           |
| —   | No `errkit` consumer pattern example               | ⏳ **NOT STARTED** | Every non-trivial consumer builds nil-safe wrappers with context. Show the pattern.                     |
| —   | `RegisterClassifications` map variant not in skill | ✅ **DONE**        | The batch map variant is now shown in SKILL.md (line ~222).                                                                                                |
| —   | `ParseFamily` default-to-Transient not in gotchas  | ✅ **DONE**        | Now in the SKILL.md "Surprising Behaviors" gotchas table.                                                                                                  |

### Note on Multi-Error Classification

This feedback says:

> `Classify(errors.Join(...))` returns the first non-Transient family, which might not be the "worst".

**This was fixed in v0.5.0.** Multi-error classification now picks the **highest-severity** sub-error (`Family.Severity()` total order: Transient(1) < Rejection(2) < Conflict(3) < Infrastructure(4) < Corruption(5)). The behavior this feedback describes no longer exists.
