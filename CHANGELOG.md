# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

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
