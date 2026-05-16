# go-error-family

Structured error protocol library. Library only — no `main`, no build system, no external deps.

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

## Test Coverage

**Updated:** 2026-05-16

| Package              | Coverage                                                                       |
| -------------------- | ------------------------------------------------------------------------------ |
| root (`errorfamily`) | 88.3%                                                                          |
| `agent`              | 100%                                                                           |
| `diagnose`           | 59.6% (rules that shell out to system commands are integration-test territory) |

Test files:

- `errorfamily_test.go` — root package tests
- `handle_test.go` — HandleError, HandleErrorWithConfig, HandleErrorDetailed, template overrides, diagnostics wiring
- `diagnose/diagnose_test.go` — Runner, rule matching helpers, Applicable methods, rule Run methods for local paths
- `agent/agent_test.go` — Analyze (enabled/disabled/with diagnosis/empty), extractCommand

## Agent Is Analysis-Only

The `DebugAgent` interface has a single method: `Analyze`. It produces root cause analysis and `FixStep` suggestions. The library does NOT execute fixes — the consumer decides what to do with `FixStep.Command`. The `Involvement` and `RiskLevel` concepts belong to the consumer, not the library.

## Diagnostic Rule Pattern

When adding a new `DiagnosticRule`, use the matching helpers in `diagnose/diagnose.go` (not `context.go`): `hasContextKey`, `contextValue`, `hasContextSubstring`, `familyIs`, `errorCodeContains`. Rules run concurrently via `Runner.Run` and results sort by confidence descending.

## Template System

Error messages are resolved in this order:

1. `HandleConfig.TemplateOverride[code]` — per-call consumer override
2. `lookupTemplate(code)` — global registry via `RegisterTemplate`
3. `defaultMessages[code]` — built-in exact-match templates
4. `familyDefaultMessage(family)` — generic family-based fallback

No substring matching. All lookups are exact code matches (case-insensitive).

## DiagnosticFunc Type

`HandleConfig.DiagnosticFunc` is a function type `func(ctx, err) []DiagnosticFinding`. This avoids the `any` return type that a circular-import-avoiding interface would require. The `diagnose.Runner` can be adapted to satisfy this type with a thin wrapper.
