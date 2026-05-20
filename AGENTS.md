# go-error-family

Structured error protocol library. Library only — no `main`, no build system, no external deps. Full API reference: `SKILL.md`.

## Surprising Behaviors

- **`Classify(nil)` returns `Rejection`**, not a zero value. Intentional: nil error = caller's fault.
- **`Classify` defaults unknown errors to `Transient`** (retryable). Fail-open design — unknown errors get retried. Same for `ParseFamily` with unrecognized strings.
- **`errors.Is` matches on `code + family` only**, ignoring message. Two `*Error`s with different messages but same code and family will match.
- **`Wrap(nil, ...)` returns `nil`** — nil-safe, but means you can't construct an error wrapping nil.
- **Consumer interfaces (`Coded`, `Classified`, `Contextual`, `Retryable`) embed `error`** — required for Go 1.26's `errors.AsType[T]()`. Don't remove the embedding.

## Classification Precedence

`Classify(err)` checks in order — first match wins:

1. `Classified` interface → `ErrorFamily()`
2. `Retryable` interface → infer `Transient` (true) or `Rejection` (false)
3. Registered sentinels via `errors.Is` chain walk (lock-free snapshot)
4. Default → `Transient`

This means a type implementing both `Classified` and `Retryable` will use `Classified` and ignore `Retryable`. Registering a sentinel for an error that already implements `Classified` has no effect.

## Agent Is Analysis-Only

The `DebugAgent` interface has a single method: `Analyze`. It produces root cause analysis and `FixStep` suggestions. The library does NOT execute fixes — the consumer decides what to do with `FixStep.Command`. The `Involvement` and `RiskLevel` concepts belong to the consumer, not the library.

## Diagnostic Rule Pattern

When adding a new `DiagnosticRule`, use the matching helpers in `diagnose/diagnose.go` (not `context.go`): `hasContextKey`, `contextValue`, `resolveContextKey`, `hasContextSubstring`, `familyIs`, `errorCodeContains`. Rules run concurrently via `Runner.Run` and results sort by confidence descending.

## Partial Success / Batch Operations

Two types for collecting multiple errors from batch or multi-step operations:

- **`ErrorBatch`** — error-only collector (no values). Use when you only need to track failures.
- **`BatchResult[T]`** — value + error collector. Use when you need successful values alongside failures.

Both are thread-safe for concurrent `Add` calls. Both produce an `Err()` that implements `Classified`, `Coded`, `Contextual`, `Retryable` — so batch errors flow through `HandleError`, `Classify`, and all existing tools seamlessly.

### Dominant Family (Severity Order)

`Corruption > Infrastructure > Conflict > Rejection > Transient`. The most severe failure determines the batch's family, exit code, and tone. If ALL failures are Transient, the batch is retryable.

### Error Format

- Partial: `[rejection:batch] 3 of 10 items failed`
- All failed: `[transient:batch] 10 items failed`
- Verbose (`%+v`): each failure numbered on its own line

### Key Methods

`ErrorBatch`: `Add`, `AddBatch`, `Len`, `HasFailures`, `Errors`, `Families`, `DominantFamily`, `HasRetryable`, `Retryable`, `Err`, `ExitCode`

`BatchResult[T]`: `Add`, `AddOutcome`, `AddResult`, `Len`, `Successes`, `Failures`, `HasFailures`, `AllSucceeded`, `AllFailed`, `IsPartial`, `Families`, `DominantFamily`, `HasRetryable`, `RetryableFailures`, `Err`, `ExitCode`

## Test Coverage

**Updated:** 2026-05-21

| Package              | Coverage |
| -------------------- | -------- |
| root (`errorfamily`) | 98.0%    |
| `agent`              | 100%     |
| `diagnose`           | 60.6%    |

Diagnose rules that shell out to system commands are integration-test territory.
