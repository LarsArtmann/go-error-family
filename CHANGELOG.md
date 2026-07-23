# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

Inspired by studying BuildFlow's `modules/errors/` package. All additions are
non-breaking and use only the Go standard library.

### Added

- **`ExitCoder` interface** (`interfaces.go`) — fifth consumer interface alongside `Coded`/`Classified`/`Contextual`/`Retryable`. Errors implementing `ExitCode() int` can override the family-based exit code on a per-error basis. Embeds `error` for `errors.AsType[T]` compatibility.
- **`Error.WithExitCode(code int) *Error`** (`error.go`) — copy-on-write mutator that sets a custom exit code. Zero means "unset, use family default."
- **`Error.WithContextAny(key string, value any) *Error`** (`error.go`) — type-safe context attachment for non-string values. Uses a type switch for common scalars (string, int, int64, uint, uint64, float64, bool, []byte, time.Time, error) and falls back to `fmt.Sprint`.
- **`WrapOnce(err, family, code, msg) *Error`** (`constructors.go`) — idempotent wrap: returns the existing `*Error` unchanged if the error chain already contains one. Prevents double-wrapping at API boundaries.
- **`WrapOncef(err, family, code, format, args...) *Error`** (`constructors.go`) — formatted variant of `WrapOnce`.
- **`errorfamilytest.AssertExitCode(tb, err, want)`** — test assertion for exit codes (checks ExitCoder override first, then family default).
- **`safeCauseString`** (`error.go`) — panic-recovery guard around `cause.Error()`. Defense-in-depth against third-party error types that panic on nil internal values. Applied to `Error()`, `Summary()`, and `formatVerbose()`.
- **`formatVerbose` now shows `exit_code` when non-zero** — improved `%+v` debug output.
- **Benchmarks**: `BenchmarkWrapOnceWrap`, `BenchmarkWrapOnceIdempotent`, `BenchmarkWithExitCode`, `BenchmarkExitCodeOverride`.
- **Examples**: `ExampleWrapOnce`, `ExampleError_WithExitCode`, `ExampleError_WithContextAny`, `ExampleExitCode`.
- **Fuzz tests**: `FuzzWrapOnce`, `FuzzContextValueToString`, `FuzzWithExitCode`.
- **`Compose(errs...)`** — re-added as backward-compatibility wrapper around `errors.Join` (was removed in v0.5.0, restored per consumer feedback; commit `8cb240a`).
- **`time.Duration` case in `contextValueToString`** — common context value type (timeouts, retry intervals) now renders via `Duration.String()` instead of `fmt.Sprint`.
- **Tests**: `TestSafeCauseStringNonStringPanics` (int, nil, struct panic recovery), `TestWriteHTTPErrorMarshalFailure` (failing writer error path).
- **CI**: `GOWORK=off go list -m all` module-graph gate, consumer-simulation job (throwaway module import), `go vet ./...` step.

### Changed

- **`ExitCode(err)` checks `ExitCoder` interface first** — a non-zero custom exit code wins over the family default. Zero falls back to `Family.ExitCode()`.
- **`HandleErrorWithContext` and `HandleErrorDetailedWithConfig`** now resolve exit codes via a shared `resolveExitCode` helper that respects the `ExitCoder` interface.
- **`Error.Error()`, `Error.Summary()`, `Error.formatVerbose()`** use `safeCauseString` instead of `%v` format for cause rendering.
- **`contextValueToString` refactored** into `contextValueToString` + `scalarToString` dispatch, eliminating `//nolint:cyclop` suppression.
- **`WithExitCode` godoc** documents POSIX exit code wrapping behavior (0-255 range, negative values wrap via `os.Exit`).

## [0.7.0] - 2026-07-09

### Changed (BREAKING)

- **Root module migrated to `encoding/json/v2`** — `error.go` (`JSON()` method) and `http.go` (`HTTPHandler` response writer) now use `encoding/json/v2` instead of `encoding/json`. This requires consumers to set `GOEXPERIMENT=jsonv2` when building on Go 1.26 (the nix devShell sets this automatically). The JSON output shape is unchanged.
- **Bridge dependencies bumped** — `samber/oops` v1.22.0 → v1.23.0, `golang.org/x/text` v0.39.0 → v0.40.0.

## [0.6.1] - 2026-07-05

Hotfix: the published `go.mod` files in 0.6.0 contained local `replace` directives plus phantom `require ... v0.0.0-00010101000000-000000000000` versions. Go strips `replace` when a module is fetched, so consumers hit unresolvable module-graph edges. `go.work` masked the defect locally.

### Fixed

- **Root `go.mod`** — removed the `replace` block and phantom `require` for `diagnose`. The root module is now genuinely zero-dependency: `GOWORK=off go list -m all` returns exactly one module.
- **`diagnose/go.mod`** — `require root v0.0.0-...` replaced with real `v0.6.0`; local `replace` removed.
- **`agent/go.mod`** — both phantom requires replaced with real `v0.6.0` + `diagnose v0.1.0`; local `replace` block removed.

### Changed

- **`examples/` is now a separate Go module** (`examples/go.mod`). Previously the root module required `diagnose` solely because `examples/cmd/custom_rule` imported it, dragging the entire `diagnose` subtree into every root consumer's module graph. Extracting examples restores the true zero-dependency invariant on root.

## [0.6.0] - 2026-07-05

Driven by consumer feedback from SEC and browser-history integrations. The root
package remains zero-dependency (all new features use only the Go standard
library: `net/http`, `log/slog`, `testing`).

### Added

- **`RegisterClassifier(func(error) (Family, bool))`** (`classify.go`/`registry.go`) — predicate-based classification for dynamic third-party errors (e.g. `*sqlite.Error`, `*pgconn.PgError`) that cannot be registered as sentinels because each occurrence is a fresh instance. Classifiers run after sentinel matching fails, in registration order; first match wins. Stored lock-free behind an `atomic.Pointer` (copy-on-write), mirroring the sentinel design. Includes package-level `RegisterClassifier`/`RegisterClassifiers` and `Registry.RegisterClassifier`/`RegisterClassifiers`; `Registry.Clone()` now copies classifiers too.
- **`Code(err) string`** (`classify.go`) — public one-liner for extracting a machine-readable error code from any error (walks the unwrap chain for the `Coded` interface). Replaces the 5-line `errors.As` boilerplate. `HandleError`'s internal `extractCode` now delegates to it.
- **`TemplateForCode(code) (MessageTemplate, bool)`** (`registry.go`/`handle.go`) — resolves a registered message template by code without wiring the full CLI pipeline. Lets HTTP/gRPC consumers look up user-facing messages directly. Available as both a `Registry` method and a package-level convenience function.
- **`Wrap{Family}f` formatted variants** (`constructors.go`) — `WrapRejectionf`, `WrapConflictf`, `WrapTransientf`, `WrapCorruptionf`, `WrapInfrastructuref`, completing the symmetry with `Newf`/`Wrapf`. All are nil-safe.
- **`HTTPStatus(err) int`** and **`HTTPHandler(fn) http.Handler`** (`http.go`) — classify→status-code helper and a ready-made net/http middleware. `HTTPHandler` wraps an error-returning handler and writes a **safe** JSON response (family/code/message) — it never leaks the raw `err.Error()`; the message comes from a registered `MessageTemplate` when available.
- **`LogError(err, *slog.Logger)`** and **`LogErrorContext(ctx, err, logger)`** (`log.go`) — structured `log/slog` logging with `family`, `code`, `retryable`, and each context key (prefixed `context.`). Transient errors log at Warn; all others at Error. Nil error is a no-op; nil logger falls back to `slog.Default`.
- **`errorfamilytest` subpackage** — test assertion helpers (`AssertFamily`, `AssertCode`, `AssertRetryable`, `AssertContext`, `AssertContextMissing`) mirroring `net/http/httptest`, keeping the `testing` import out of the production package.

### Changed

- **Classification pipeline** now has six steps (was five): registered classifiers run as step 5, before the Transient default. This is additive — errors that already declare a family (via `Classified`) bypass classifiers entirely, so the hot path is unaffected. Doc comments on `Classify` (both package-level and `Registry` method) updated.
- **`HTTPStatus` doc** now documents the rationale for each family→status mapping (notably why Corruption→500 rather than 422: a data-integrity break is the server's problem, not the client's).
- **`Code()` vs `ErrorCode()` clarified**: doc comments now explain that `ErrorCode()` is the canonical `Coded` interface contract while `Code()` is an ergonomic accessor on `*Error` (sibling of `Family()`/`Message()`); both are intentional, neither is deprecated.

## [0.5.0] - 2026-06-22

First release since `v0.4.0`. Consolidates the copy-on-write error refactor, module extraction, severity-ordered multi-error classification, lock-free sentinel lookup, structured diagnostic fixes, new family adapters, and the bridge `%s` format fix.

### Added

- **`Registry` type** (`registry.go`) — injectable classification sentinels + message templates. Replaces global mutable maps with a construct-and-pass type for test isolation (no `t.Cleanup(Unregister...)` needed) and scoped error handling within a single binary. Zero value is not usable — use `NewRegistry()`.
- **`NewRegistry()`** constructor and **`DefaultRegistry`** package-level var — backward-compatible defaults for all convenience functions (`Classify`, `RegisterClassification`, `RegisterTemplate`, etc.).
- **`HandleConfig.Registry`** field — pass a custom registry to `HandleError*` functions. Falls back to `DefaultRegistry` when nil.
- **`Registry.Clone()`** — deep-copy for inherit-and-extend patterns (start from `DefaultRegistry`, clone, register scope-specific overrides without touching the global).
- **`Registry.RegisterTemplates(map)`** — batch template registration, matching the existing `RegisterClassifications` batch.
- **`Family.Severity() int`** — total order for multi-error classification (Transient < Rejection < Conflict < Infrastructure < Corruption).
- **`Family.HTTPStatus() int`** — canonical family → HTTP status mapping (Rejection→400, Conflict→409, Transient→503, Corruption→500, Infrastructure→503).
- **`Family.RetryPolicy() RetryPolicy`** — advisory retry defaults per family (Transient: 3 attempts, 100ms–5s; others: single attempt). The library does not run the loop.
- **`Error.JSON() ([]byte, error)`** — canonical JSON (`{family,code,message,context,retryable,timestamp}`) for API boundaries.
- **`Error.WithContextMap(map[string]string)`** and **`Error.WithContextf(key, format, args...)`** — batch and formatted context attachment.
- **`RegisterStdlibDefaults(reg)`** (`stdlib.go`) — maps `context`/`sql`/`os` errors with documented rationale for ambiguous cases (DeadlineExceeded→Transient, Canceled→Rejection, etc.).
- **`diagnose/`** and **`agent/`** are now independent Go modules with their own `go.mod`, enabling independent versioning. Import paths are unchanged.
- **`go.work`** expanded to 6 workspace modules (root, diagnose, agent, bridge, diagnose/git, diagnose/postgres).
- **Experimental stability notices** in package docs for `agent`, `diagnose`, `diagnose/git`, `diagnose/postgres`, and `bridge`. Root package documented as the stable classification core.
- **Fuzz tests**: `FuzzParseFamily`, `FuzzParseFamilyRoundTrip`, `FuzzClassify`, `FuzzClassifyPlainError`, `FuzzErrorFormatting` (root); `FuzzFormat` (bridge).

### Changed (BREAKING)

- **Copy-on-write errors:** `WithContext`, `WithCause`, and `WithTimestamp` now return a NEW `*Error` instead of mutating the receiver. Fixes a data race when errors are shared across goroutines (e.g. package-level sentinels). Previous chaining calls that assumed identity preservation still compile but now get a distinct pointer.
- **Template placeholder syntax** changed from `{{.key}}` to `{key}`. The old syntax collided with Go's `text/template`. Migration: replace all `{{.key}}` with `{key}` in registered templates.
- **Severity-ordered multi-error classification:** `Classify` on an `errors.Join` result now returns the worst (highest-severity) sub-error instead of the first non-Transient one. Classification is deterministic regardless of join argument order; fail-closed retry semantics preserved.
- **Lock-free sentinel lookup:** `Registry.sentinels` changed from `map[error]Family` to `atomic.Pointer[sentinelMap]` (copy-on-write). At 50 registered sentinels, `Classify` dropped from ~1330 ns/3 allocs/1832 B to ~285 ns/0 allocs/0 B.
- **Structured diagnostic fixes:** `DiagnosticResult.SuggestedFix string` replaced with `Fix struct{Summary, Command string}`. Diagnostic rules now emit the remediation summary and shell command as distinct fields. The `agent` no longer parses suggestions — `extractCommand` and `looksLikeCommand` (40+ lines of heuristic prose parsing) are deleted; `FixStep.Command` comes directly from `diagnose.Fix.Command`.
- **Module extraction:** root module no longer contains `diagnose/` and `agent/` as sub-packages — they are separate modules. Consumers using `go.work` see no difference. Local `replace` directives added until published versions resolve the extraction.
- **`agent.Config.Enabled`** now returns `(nil, error)` instead of a synthetic `AgentResult`. Calling `Analyze` on a disabled agent is a programming error, not a silent no-op.

### Changed

- Package-level `RegisterClassification`/`RegisterClassifications`/`UnregisterClassification`/`RegisterTemplate`/`UnregisterTemplate` now delegate to `DefaultRegistry` (backward compatible).
- Template resolution (override → registry → built-in default) extracted into a single shared `resolveTemplate` helper used by both `renderCLI` and `resolveSuggestedFix`, eliminating split-brain divergence. Templates are cohesive units (What/Why/Fix belong together).
- README retry wording clarified: `IsRetryable` returns a binary signal; backoff, jitter, and idempotency are the consumer's responsibility.

### Removed

- **`Compose(errs...)`** — use stdlib `errors.Join` directly. `Classify` already classifies multi-errors. One less API surface to learn.

### Fixed

- **Data race** in `WithContext`/`WithCause`/`WithTimestamp` — these methods mutated the receiver's fields and returned the same pointer. All three now use copy-on-write via a shared `clone()` helper.
- **Bridge `Format(%s)`** returned an empty string when the wrapped error's message was empty — added a `[family]` fallback matching `Error()` and `%v`. Found by `FuzzFormat`.

### Modules

Coordinated multi-module release.

- `github.com/larsartmann/go-error-family` → **v0.5.0** (breaking changes + new APIs)
- `github.com/larsartmann/go-error-family/diagnose` → **v0.1.0** (first tagged release — structured `Fix`, `MockCommandRunner`)
- `github.com/larsartmann/go-error-family/agent` → **v0.1.0** (first tagged release — structured `FixStep`)
- `github.com/larsartmann/go-error-family/bridge` → **v0.2.0** (format fix + root v0.5.0 bump)
- `github.com/larsartmann/go-error-family/diagnose/git` → **v0.4.0** (structured `Fix`, `MockCommandRunner.Set`)
- `github.com/larsartmann/go-error-family/diagnose/postgres` → **v0.4.0** (structured `Fix`, `MockCommandRunner.Set`)

## [0.4.0] - 2026-06-17

### Added

- `Family` and `Audience` now implement `encoding.TextMarshaler`/`TextUnmarshaler`, enabling YAML/JSON config decoding of error families and audiences (e.g. unmarshalling a family from a config struct tag)
- `ParseAudience(string)` and `ParseStatus(string)` — case-insensitive string parsing, completing the enum-parse trio alongside `ParseFamily`
- `Audience.IsValid()` — validation mirroring `Family.IsValid()`, giving all three enums (`Family`, `Audience`, `Status`) a consistent validation API
- `Family.Audience()` — exposes the audience metadata (User vs Operator) for each family; audience is now a first-class field in the family metadata table
- `diagnose.Status.IsValid()` — validation consistency with `Family.IsValid()` and `Audience.IsValid()`
- `Compose` rationale documentation and `example_test.go` covering the new enum APIs
- Integration tests for `FilesystemRule` (temp-dir filesystem) and `NetworkRule` (localhost DNS/TCP), raising diagnose core coverage to ~77%

### Changed

- **BREAKING:** Removed `HandleConfig.Diagnose` bool field. Diagnostics now run whenever `DiagnosticFunc` is set — no separate enable flag. Consumers using `Diagnose: true` must drop that field; diagnostic behavior is unchanged when a `DiagnosticFunc` is configured.
- **BREAKING:** `agent.Config.Enabled` now returns `(nil, error)` instead of a synthetic `AgentResult`. Calling `Analyze` on a disabled agent is a programming error rather than a silent no-op result.
- `familyInfo` gained an `Audience` field; adding a new `Family` now requires only a single entry in the `familyData` table (previously audience was implicit).

### Fixed

- `lookupRegistered` now snapshots the classification registry map before iterating, so `errors.Is` chain walks run lock-free — eliminates a deadlock risk under concurrent sentinel registration.
- `NetworkRule` returns `StatusUnknown` when no host is found in the error context, preventing undefined DNS resolution behavior.

### Modules

Coordinated multi-module release. Submodule `go.mod` files retain their root dependency at **v0.3.0** — a valid lower bound since none use v0.4.0-only APIs. Consumers pulling root v0.4.0 get it automatically via MVS.

- `github.com/larsartmann/go-error-family` → **v0.4.0** (breaking changes + new APIs)
- `github.com/larsartmann/go-error-family/diagnose/git` → **v0.3.0** (new `Runner` field on `GitRule`; was skipped in v0.3.0 root release)
- `github.com/larsartmann/go-error-family/diagnose/postgres` → **v0.3.0** (new `Runner` field on `PostgresRule`; was skipped in v0.3.0 root release)
- `github.com/larsartmann/go-error-family/bridge` → **v0.1.1** (lint fix, transitive dependency update)

## [0.3.0] - 2026-06-01

### Added

- `HandleErrorWithContext(ctx, err, cfg)` — new entry point that propagates caller context to diagnostic functions (fixes context.Background() hardcode)
- `HandleErrorDetailedWithConfig(err, cfg)` — template-aware structured result for HTTP/gRPC consumers
- `CommandRunner` interface in `diagnose` package — injectable command execution for testable diagnostic rules
- `DefaultCommandRunner` struct — zero-value default wrapping `RunCommand`/`CommandExists`
- `ContextKey` typed string with exported constants (`KeyHost`, `KeyPort`, `KeyPath`, `KeyDBHost`, etc.)
- `DiagnosticResult.Context` field — surfaces the error context that triggered the rule
- `ErrorContext(err)` helper in `diagnose` package — extracts context from any error
- `Error.WithTimestamp(ts)` mutator — for testing and deterministic construction
- `Compose(errs...)` helper — combines errors via `errors.Join` for partial-success patterns
- Package-level `Example` functions: `ExampleNewTransient`, `ExampleClassify`, `ExampleHandleError`, `ExampleWrapRejection`, `ExampleParseFamily`
- Expanded git tests with mock CommandRunner: dirty tree, merge conflicts, unreachable remote, no git binary (coverage 98.5%)
- Expanded postgres tests with mock CommandRunner: pg_isready success/failure, suggestStartFix variants (coverage 81.0%)
- `UnregisterClassification` — removes a previously registered sentinel classification (for test cleanup)
- `UnregisterTemplate` — removes a previously registered message template (for test cleanup)
- `diagnose.MockCommandRunner` — shared, deterministic mock for diagnostic rule tests
- `diagnose.NewMockCommandRunner()` — constructor for `MockCommandRunner`
- `diagnose.ResolveRunner(r)` — helper that returns `r` if non-nil, otherwise `DefaultCommandRunner{}`
- `TestRunnerContextCancelledMidRun` — verifies early return when context is cancelled mid-run

### Changed

- `HandleErrorWithConfig` now delegates to `HandleErrorWithContext(context.Background(), ...)`
- `HandleErrorDetailed` now uses the full template resolution chain (registered templates, consumer overrides, family fallbacks)
- `RuleSpec.ContextKeys` field type changed from `[]string` to `[]ContextKey`
- All diagnostic rules use typed `ContextKey` constants instead of raw strings
- All diagnostic rules populate `DiagnosticResult.Context` from the error's `ErrorContext()`
- `GitRule` and `PostgresRule` accept optional `Runner diagnose.CommandRunner` field (defaults to `DefaultCommandRunner`)
- `HandleError` benchmark uses `io.Discard` to suppress stderr output (532ns/op vs 1095ns/op before)
- Improved godoc on `Family`, `Tone`, `Audience`, `HandleResult`, `MessageTemplate`, `DiagnosticFinding`
- Updated `diagnose` package doc comment with custom rule pattern and CommandRunner usage
- `Runner.Run` refactored into three focused methods (`applicableRules`, `runRules`, `sortByConfidence`) for lower cyclomatic complexity
- `Runner.Run` now respects context cancellation via buffered channels with `select` on `ctx.Done()` (previously could hang on slow rules)
- Git and postgres test packages migrated from local mock types to shared `diagnose.MockCommandRunner` (~115 lines of duplicated mock code removed)
- Git and postgres `cmdRunner()` methods consolidated via `diagnose.ResolveRunner()`

### Fixed

- `HandleErrorWithConfig` now passes caller context to `DiagnosticFunc` instead of `context.Background()`
- `HandleError` benchmark no longer writes ~1M lines to stderr during benchmark runs
- `NetworkRule.resolveHost` uses `net/url.Parse` and `net.Dialer` for TCP dial instead of raw string splitting
- `FilesystemRule` uses `filepath.Ext` instead of `strings.Contains(".")` for file vs directory detection
- `Compose` doc comment corrected; `extractCommand` updated to match diagnostic fix formats

## [0.2.0] - 2026-05-26

### Changed

- **BREAKING: Modularized diagnostic rules** — `GitRule` moved to `diagnose/git` submodule, `PostgresRule` moved to `diagnose/postgres` submodule. `DefaultRunner()` now includes only zero-dependency rules (`FilesystemRule`, `NetworkRule`). Consumers must opt into git/postgres diagnostics via explicit submodule import:
  ```go
  import "github.com/larsartmann/go-error-family/diagnose/git"
  runner := diagnose.NewRunner(&git.GitRule{})
  ```
- Exported all diagnostic rule helpers: `RuleSpec`, `HasContextKey`, `ContextValue`, `ResolveContextKey`, `HasContextSubstring`, `FamilyIs`, `ErrorCodeContains`, `RunCommand`, `CommandExists`
- G304 gosec exclusion for `diagnose/rules_filesystem.go` moved from inline `//nolint` to `.golangci.yml` path-based rule (eliminates golines formatting conflict)
- Extracted string constants in postgres submodule for goconst compliance

### Added

- Fuzz tests: `FuzzParseFamily`, `FuzzParseFamilyRoundTrip`, `FuzzClassify`, `FuzzClassifyPlainError`, `FuzzErrorFormatting`
- Benchmark suite: 16 benchmarks covering `Classify`, `HandleError`, `Runner.Run`, `ParseFamily`, and more
- Runnable examples in `examples/`:
  - `cmd/cli` — CLI boundary handler with contextual messages
  - `cmd/http` — HTTP middleware with family-to-status-code mapping
  - `cmd/custom_rule` — How to implement `DiagnosticRule` from scratch
- Integration tests for git submodule using temp git repos (clean, dirty, context key resolution)
- Expanded postgres test suite with 13 `Applicable` cases, table-driven `resolveHost`/`resolvePort` tests

### Fixed

- `lookupRegistered` deadlock risk eliminated — map snapshot copied before `errors.Is` iteration (lock-free)

## [0.1.1] - 2026-05-16

### Changed

- **License changed from Proprietary to MIT** — the project is now open source
- Rewrote README to accurately reflect the actual API (previous version documented fabricated agent APIs)

### Fixed

- README AI Agent section documented non-existent `Involvement` levels, `ConfirmFunc`, and `FixStep.Risk` — replaced with actual `agent.Config{Enabled, Timeout}` and `FixStep{Description, Command, Rationale}` API
- README contained a dead link to a non-existent design doc in `docs/` — removed
- `Newf` code example was missing the `errorfamily.` package prefix on `Rejection` — fixed

### Added

- Badges (Go Reference, Go Report Card) and Installation section to README
- CLI Boundary section documenting `HandleErrorWithConfig`, `HandleErrorDetailed`, `HandleConfig`, `MessageTemplate`
- Template Resolution subsection (What/Why/Fix/WayOut lookup precedence)
- Classification precedence explanation
- `ParseFamily`, `Audience()`, `Tone()`, `RegisterClassifications` (batch) documentation
- Architecture tree now accurately reflects exports (e.g. `context.go` is internal)
- Repo made public with description and topics

## [0.1.0] - 2026-05-10

### Added

- `Family` enum: Rejection, Conflict, Transient, Corruption, Infrastructure
- Small interfaces: `Coded`, `Classified`, `Contextual`, `Retryable` (each embeds `error`)
- `Error` struct: reference implementation with `Is`, `Unwrap`, `Format`, `WithContext`, `Summary`
- Family-specific constructors: `NewRejection`, `WrapTransient`, etc.
- `Classify(err)` — universal classification for any error
- `ExitCode(err)` — BSD sysexits.h exit codes from Family
- `IsRetryable(err)` — retry decision from Family
- `HandleError(err)` — CLI boundary handler with structured messages
- `HandleErrorWithConfig` — configurable handler with template overrides and diagnostics
- `HandleErrorDetailed` — structured result for HTTP/gRPC handlers
- `MessageTemplate` — Wix-style What/Why/Fix/WayOut templates with `{{.key}}` substitution
- `RegisterTemplate` — global template registry
- `RegisterClassification` / `RegisterClassifications` — map third-party errors to families
- Diagnostic rules: `PostgresRule`, `FilesystemRule`, `NetworkRule`, `GitRule`
- `diagnose.Runner` — concurrent rule execution with confidence-sorted results
- `agent.DebugAgent` interface — root cause analysis and `FixStep` suggestions
- `ParseFamily` — case-insensitive string-to-Family (defaults to Transient for unknowns)
- `Audience` and `Tone` types for presentation-layer decisions
