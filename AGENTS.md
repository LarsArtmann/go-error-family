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

## Test Coverage

**Updated:** 2026-05-20

| Package              | Coverage |
| -------------------- | -------- |
| root (`errorfamily`) | 97.1%    |
| `agent`              | 100%     |
| `diagnose`           | 60.6%    |

Diagnose rules that shell out to system commands are integration-test territory.
