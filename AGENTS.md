# go-error-family

Structured error protocol library. Library only — no `main`, no build system, no external deps.

## Surprising Behaviors

- **`Classify(nil)` returns `Rejection`**, not a zero value. Intentional: nil error = caller's fault.
- **`Classify` defaults unknown errors to `Transient`** (retryable). This is a fail-open design — unknown errors get retried rather than silently dropped. Same for `ParseFamily` with unrecognized strings.
- **`errors.Is` matches on `code + family` only**, ignoring message. Two `*Error`s with different messages but same code and family will match.
- **`Wrap(nil, ...)` returns `nil`** — nil-safe, but means you can't construct an error wrapping nil.
- **NetworkRule fires on ALL Transient-family errors** (via `familyIs(err, Transient)` in `Applicable`). Any Transient error triggers full DNS + TCP diagnostics, even if unrelated to networking.
- **Consumer interfaces (`Coded`, `Classified`, `Contextual`, `Retryable`) embed `error`** — this is required for Go 1.26's `errors.AsType[T]()` which demands `T` satisfy the `error` interface. Don't remove the embedding.

## Classification Precedence

`Classify(err)` checks in order — first match wins:

1. `Classified` interface → `ErrorFamily()`
2. `Retryable` interface → infer `Transient` (true) or `Rejection` (false)
3. Registered sentinels via `errors.Is` chain walk
4. Default → `Transient`

This means a type implementing both `Classified` and `Retryable` will use `Classified` and ignore `Retryable`. Registering a sentinel for an error that already implements `Classified` has no effect.

## Test Gaps

`diagnose/` and `agent/` have **zero test files**. Modifying these packages requires manual verification. Root package tests are in `errorfamily_test.go` (table-driven, `testing.T`).

## Incomplete Features

- **AI agent is scaffold** — `agent.deterministicAnalyze` produces rule-based results from diagnostic output. No AI provider is wired. The `buildPrompt` method constructs a prompt string that goes unused.
- **`handle.go` has intentionally unused parameters** in `formatWhy` and `applyTemplate` — reserved for future use. gopls will warn about these; they are not bugs.

## Diagnostic Rule Pattern

When adding a new `DiagnosticRule`, use the matching helpers in `diagnose/diagnose.go` (not `context.go`): `hasContextKey`, `contextValue`, `hasContextSubstring`, `familyIs`, `errorCodeContains`. Rules run concurrently via `Runner.Run` and results sort by confidence descending.
