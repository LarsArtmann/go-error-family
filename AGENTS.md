# go-error-family

Structured error protocol library. Library only — no `main`, no build system, no external deps. Full API reference: `SKILL.md`.

**Last Updated:** 2026-06-17
**Version:** v0.6.0
**Status:** All tests pass (root + bridge + submodules), 0 lint issues, 0 race conditions
**Workspace modules:** root (zero-dep), `agent`, `bridge` (oops integration), `diagnose`, `diagnose/git`, `diagnose/postgres`

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
- **`WithContext`/`WithCause`/`WithTimestamp` are copy-on-write** — they return a NEW `*Error`, not the same pointer. Safe to chain from shared/sentinel errors. Do NOT assume identity preservation.
- **Template placeholders use `{key}`, not `{{.key}}`** — the old syntax collided with Go's `text/template`. Migration: replace all `{{.key}}` with `{key}` in registered templates.
- **Consumer interfaces (`Coded`, `Classified`, `Contextual`, `Retryable`) embed `error`** — required for Go 1.26's `errors.AsType[T]()`. Don't remove the embedding.
- **`HandleErrorWithContext` is the canonical entry point** — `HandleError` and `HandleErrorWithConfig` delegate to it. Always prefer the context-accepting variant when you have a `context.Context`.
- **Package-level `Classify`/`RegisterClassification`/`RegisterTemplate` delegate to `DefaultRegistry`** — backward compatible. For test isolation or scoped handling, construct a `NewRegistry()` and pass it via `HandleConfig.Registry`.
- **`CommandRunner` defaults to `DefaultCommandRunner{}`** — rules with a nil `Runner` field use the real system commands. Tests inject mocks.

## New APIs (v0.3.0)

| API                                       | Purpose                                                    |
| ----------------------------------------- | ---------------------------------------------------------- |
| `HandleErrorWithContext(ctx, err, cfg)`   | Context-propagating CLI boundary handler                   |
| `HandleErrorDetailedWithConfig(err, cfg)` | Template-aware structured result                           |
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
4. Registered sentinels via `errors.Is` chain walk (snapshot copy, lock-free iteration)
5. Default → `Transient`

This means a type implementing both `Classified` and `Retryable` will use `Classified` and ignore `Retryable`. Registering a sentinel for an error that already implements `Classified` has no effect.

**Multi-error behavior:** For `errors.Join(err1, err2, ...)`, each sub-error is classified recursively. The first sub-error with a non-Transient family determines the result. If all are Transient, the result is Transient. This is fail-closed: if any part of a multi-error is not retryable, the whole operation is not retryable.

## Registry Pattern

The library uses an injectable `Registry` type (`registry.go`) that holds both classification sentinels and message templates. The zero value is not usable — use `NewRegistry()`.

- **`DefaultRegistry`** is a package-level `*Registry` used by all convenience functions (`Classify`, `RegisterClassification`, `RegisterTemplate`, etc.) and by `HandleError` when `HandleConfig.Registry` is nil.
- **Custom registries** enable test isolation (no `t.Cleanup(Unregister...)` needed) and scoped error handling within a single binary. Pass via `HandleConfig.Registry`.
- **Thread-safety:** `Registry` snapshots the sentinels map under RLock before iterating — `errors.Is` runs lock-free. Same pattern as the original global registry.
- **`resolveSuggestedFix`** checks the same template chain as `renderCLI`: per-call override → registry template → built-in default → family fallback.

## Agent Is Analysis-Only

The `DebugAgent` interface has a single method: `Analyze`. It produces root cause analysis and `FixStep` suggestions. The library does NOT execute fixes — the consumer decides what to do with `FixStep.Command`. The `Involvement` and `RiskLevel` concepts belong to the consumer, not the library.

## Diagnostic Rule Pattern

When adding a new `DiagnosticRule`, use the matching helpers from the `diagnose` package: `HasContextKey`, `ContextValue`, `ResolveContextKey`, `HasContextSubstring`, `FamilyIs`, `ErrorCodeContains`. Use execution helpers `RunCommand` and `CommandExists` for system checks. Rules run concurrently via `Runner.Run` and results sort by confidence descending.

**Submodules:** `GitRule` lives in `github.com/larsartmann/go-error-family/diagnose/git`, `PostgresRule` in `github.com/larsartmann/go-error-family/diagnose/postgres`. `DefaultRunner()` only includes zero-dep rules (`FilesystemRule`, `NetworkRule`).

## Partial Success

Not a library type — partial success is a consumption pattern, not a classification concern. See SKILL.md for the recipe (collect outcomes, `Classify` each failure, pick worst family for exit code). The library provides the classification vocabulary; consumers compose the collection strategy.

## Test Coverage

**Updated:** 2026-06-17

| Package              | Coverage |
| -------------------- | -------- |
| root (`errorfamily`) | 98.4%    |
| `agent`              | 89.4%    |
| `diagnose` (core)    | 77.3%    |
| `diagnose/git`       | 98.5%    |
| `diagnose/postgres`  | 80.3%    |

Root coverage at 98.4% (up from 96.5%) after adding `registry_test.go` with 9 isolation tests.

## Fuzz Tests

`fuzz_test.go` contains: `FuzzParseFamily`, `FuzzParseFamilyRoundTrip`, `FuzzClassify`, `FuzzClassifyPlainError`, `FuzzErrorFormatting`.

## Bridge Submodule (`bridge/`)

Connects go-error-family with `samber/oops`. Separate module with its own `go.mod` (depends on both libraries). The root package remains zero-dependency.

| API                        | Purpose                                                                               |
| -------------------------- | ------------------------------------------------------------------------------------- |
| `bridge.Wrap(err, family)` | Attach a Family to any error, preserving OopsError context                            |
| `bridge.AutoWrap(err)`     | Infer Family from oops metadata (tags + domain), then wrap                            |
| `bridge.InferFamily(err)`  | Derive Family from oops tags (explicit) → domain (structural) → Transient (fail-open) |
| `ClassifiedError`          | Embeds `oops.OopsError`; satisfies `Classified`, `Coded`, `Retryable`, `Contextual`   |

**Tag overrides** (checked first): `retryable`, `transient`, `conflict`, `corruption`/`corrupted`, `rejection`/`rejected`, `infrastructure`/`infra`.
**Domain defaults** (checked second): `validation`/`auth` → Rejection, `database`/`network`/`cache`/`queue` → Transient, `storage`/`infra`/`startup` → Infrastructure, `data`/`schema`/`migration` → Corruption.

**Surprising:** `Wrap(nil, family)` returns a ClassifiedError with zero OopsError — `Error()` returns `[family]`, `Unwrap()` returns nil. This is intentional: nil is still classifiable.

## Lint Configuration

**Updated:** 2026-06-17

- `bridge` package-level lookup tables (`domainDefaults`, `tagOverrides`) suppress `gochecknoglobals` via inline `//nolint` — same pattern as root's immutable lookup tables.

- G304 (gosec file inclusion) is excluded for `diagnose/rules_filesystem.go` via `.golangci.yml` path-based exclusion — `os.Open(path)` and `os.Create(testFile)` are intentional in diagnostic rules.
- Do NOT use `//nolint:gosec` directives for G304 in the diagnose package — the `.golangci.yml` exclusion handles it. Inline nolint directives break when `golines` wraps lines.
- `ContextKey` type replaces raw strings in rule specs. `CodeContains` fields still use raw strings (different semantic — substring matching on error codes, not context keys).
- `CommandRunner` interface allows mock injection; `DefaultCommandRunner` wraps real system calls.
- `gochecknoglobals` is enabled but suppressed via `//nolint:gochecknoglobals` on each legitimate package-level var (mutex-protected registries, immutable lookup tables, rule specs) — the BuildFlow pre-commit auto-configure hook re-enables it if disabled in `.golangci.yml`.
- `exhaustruct` is enabled but most project types are excluded via `.golangci.yml` because they have intentional optional fields (HandleConfig, MessageTemplate, DiagnosticResult, etc.). Test files also exclude exhaustruct.
- `flake.nix` uses `pkgs.go_1_26` as `goPkg` — do NOT use `let goPkg = goPkg;` (infinite recursion).
- `lookupRegistered` is now `Registry.lookupSentinel` — still snapshots the map before iterating, `errors.Is` runs lock-free. No deadlock possible.
- `HandleConfig.Registry` field added — when nil, falls back to `DefaultRegistry`. `resolveSuggestedFix` checks registry templates alongside built-in defaults.
- `Registry` is excluded from `exhaustruct` via `.golangci.yml` — the `mu` field (sync.RWMutex) has a correct zero value set by `NewRegistry()`.
- `HandleConfig.Diagnose` bool was removed — diagnostics run whenever `DiagnosticFunc` is set. No separate enable flag.
- `diagnose.Status` has `IsValid()` matching `Family.IsValid()` pattern.
- `diagnose.sortByConfidence` uses `slices.SortFunc` (Go 1.26 stdlib).
- CI now has explicit `bridge/` test and lint steps, plus `go build ./examples/...` step.
- `familyInfo` includes `Audience` field — adding a new Family truly requires only one entry in `familyData`.
- `NetworkRule.Run` returns `StatusUnknown` when no host found in error context (prevents undefined DNS behavior).
- `Audience.IsValid()` mirrors `Family.IsValid()` and `Status.IsValid()` — all three enum types have consistent validation.
- `ParseAudience` and `ParseStatus` mirror `ParseFamily` — case-insensitive string parsing for all enums.
- `Family` and `Audience` implement `encoding.TextMarshaler`/`TextUnmarshaler` for YAML/JSON config.
- `agent.Config.Enabled` now returns `(nil, error)` instead of synthetic result — calling `Analyze` on a disabled agent is a programming error.

## Known Limitations

- **`applyContext` uses `{key}` syntax (handle.go):** Template values are substituted via `strings.ReplaceAll` without HTML escaping. This is intentional for CLI output (stderr) but would be unsafe for HTML rendering. Consumers building HTTP responses should escape values before embedding in HTML.
- **`agent.Config.Enabled` is now honest:** A disabled agent returns `(nil, error)` instead of a synthetic `AgentResult`. Calling `Analyze` on a disabled agent is a programming error, not a silent result.
- **`ClassifiedError` value-embeds `oops.OopsError`:** The zero value has nil internals. Methods like `Error()` and `Is()` guard against this, but future methods added to `ClassifiedError` must handle the zero-OopsError case.
- **Examples built in CI:** `examples/cmd/` is now compiled by a CI step (`go build ./examples/...`).
