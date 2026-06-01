# go-error-family

Structured error protocol library. Library only — no `main`, no build system, no external deps. Full API reference: `SKILL.md`.

**Last Updated:** 2026-05-31
**Version:** v0.3.0-dev
**Status:** All tests pass, 0 lint issues, 0 race conditions

## Quick Start

```bash
go test ./... -count=1 -timeout 120s -race   # all tests
golangci-lint run ./...                        # lint (all modules)
go build ./...                                 # build check
```

## Surprising Behaviors

- **`Classify(nil)` returns `Rejection`**, not a zero value. Intentional: nil error = caller's fault.
- **`Classify` defaults unknown errors to `Transient`** (retryable). Fail-open design — unknown errors get retried. Same for `ParseFamily` with unrecognized strings.
- **`errors.Is` matches on `code + family` only**, ignoring message. Two `*Error`s with different messages but same code and family will match.
- **`Wrap(nil, ...)` returns `nil`** — nil-safe, but means you can't construct an error wrapping nil.
- **Consumer interfaces (`Coded`, `Classified`, `Contextual`, `Retryable`) embed `error`** — required for Go 1.26's `errors.AsType[T]()`. Don't remove the embedding.
- **`HandleErrorWithContext` is the canonical entry point** — `HandleError` and `HandleErrorWithConfig` delegate to it. Always prefer the context-accepting variant when you have a `context.Context`.
- **`CommandRunner` defaults to `DefaultCommandRunner{}`** — rules with a nil `Runner` field use the real system commands. Tests inject mocks.

## New APIs (v0.3.0)

| API                                       | Purpose                                                    |
| ----------------------------------------- | ---------------------------------------------------------- |
| `HandleErrorWithContext(ctx, err, cfg)`   | Context-propagating CLI boundary handler                   |
| `HandleErrorDetailedWithConfig(err, cfg)` | Template-aware structured result                           |
| `Compose(errs...)`                        | `errors.Join` wrapper for partial-success                  |
| `Error.WithTimestamp(ts)`                 | Deterministic timestamp for testing                        |
| `diagnose.CommandRunner`                  | Injectable command execution interface                     |
| `diagnose.DefaultCommandRunner{}`         | Default implementation using `RunCommand`/`CommandExists`  |
| `diagnose.ContextKey`                     | Typed string for context keys (`KeyHost`, `KeyPort`, etc.) |
| `diagnose.ErrorContext(err)`              | Extract context from any error                             |
| `DiagnosticResult.Context`                | Error context that triggered the rule                      |

## Classification Precedence

`Classify(err)` checks in order — first match wins:

1. **Multi-error** (`errors.Join`) → classify each sub-error, first non-Transient wins
2. `Classified` interface → `ErrorFamily()`
3. `Retryable` interface → infer `Transient` (true) or `Rejection` (false)
4. Registered sentinels via `errors.Is` chain walk (lock-free snapshot)
5. Default → `Transient`

This means a type implementing both `Classified` and `Retryable` will use `Classified` and ignore `Retryable`. Registering a sentinel for an error that already implements `Classified` has no effect.

**Multi-error behavior:** For `errors.Join(err1, err2, ...)`, each sub-error is classified recursively. The first sub-error with a non-Transient family determines the result. If all are Transient, the result is Transient. This is fail-closed: if any part of a multi-error is not retryable, the whole operation is not retryable.

## Agent Is Analysis-Only

The `DebugAgent` interface has a single method: `Analyze`. It produces root cause analysis and `FixStep` suggestions. The library does NOT execute fixes — the consumer decides what to do with `FixStep.Command`. The `Involvement` and `RiskLevel` concepts belong to the consumer, not the library.

## Diagnostic Rule Pattern

When adding a new `DiagnosticRule`, use the matching helpers from the `diagnose` package: `HasContextKey`, `ContextValue`, `ResolveContextKey`, `HasContextSubstring`, `FamilyIs`, `ErrorCodeContains`. Use execution helpers `RunCommand` and `CommandExists` for system checks. Rules run concurrently via `Runner.Run` and results sort by confidence descending.

**Submodules:** `GitRule` lives in `github.com/larsartmann/go-error-family/diagnose/git`, `PostgresRule` in `github.com/larsartmann/go-error-family/diagnose/postgres`. `DefaultRunner()` only includes zero-dep rules (`FilesystemRule`, `NetworkRule`).

## Partial Success

Not a library type — partial success is a consumption pattern, not a classification concern. See SKILL.md for the recipe (collect outcomes, `Classify` each failure, pick worst family for exit code). The library provides the classification vocabulary; consumers compose the collection strategy.

## Test Coverage

**Updated:** 2026-05-31

| Package              | Coverage |
| -------------------- | -------- |
| root (`errorfamily`) | 97.2%    |
| `agent`              | 100%     |
| `diagnose` (core)    | 66.8%    |
| `diagnose/git`       | 98.5%    |
| `diagnose/postgres`  | 81.0%    |

Git and postgres coverage improved with mock `CommandRunner` injection. Diagnose core coverage reflects shell-out rules that are tested via integration.

## Fuzz Tests

`fuzz_test.go` contains: `FuzzParseFamily`, `FuzzParseFamilyRoundTrip`, `FuzzClassify`, `FuzzClassifyPlainError`, `FuzzErrorFormatting`.

## Lint Configuration

**Updated:** 2026-05-31

- G304 (gosec file inclusion) is excluded for `diagnose/rules_filesystem.go` via `.golangci.yml` path-based exclusion — `os.Open(path)` and `os.Create(testFile)` are intentional in diagnostic rules.
- Do NOT use `//nolint:gosec` directives for G304 in the diagnose package — the `.golangci.yml` exclusion handles it. Inline nolint directives break when `golines` wraps lines.
- `ContextKey` type replaces raw strings in rule specs. `CodeContains` fields still use raw strings (different semantic — substring matching on error codes, not context keys).
- `CommandRunner` interface allows mock injection; `DefaultCommandRunner` wraps real system calls.
