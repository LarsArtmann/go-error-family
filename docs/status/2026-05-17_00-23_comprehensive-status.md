# go-error-family — Comprehensive Status Report

**Date:** 2026-05-17 00:23  
**Branch:** master (fd804e8)  
**Tags:** v0.1.0 (2026-05-13), v0.1.1 (2026-05-16)  
**License:** MIT  
**Go:** 1.26.2 | **Dependencies:** zero external

---

## Executive Summary

go-error-family is a structured error protocol library for Go. It is **open-sourced, published to the Go module proxy, and fully functional**. The core error model, classification system, CLI boundary handler, and diagnostic framework are production-ready. The AI agent is a deterministic stub awaiting a real provider. Test coverage is strong overall (74.9% total) but the `diagnose` package drags the average down at 59.5%. Documentation is honest and accurate — no fabricated APIs remain.

---

## a) FULLY DONE

### Core Error Model ✅

- `Family` enum with 5 behavioral classifications (Rejection, Conflict, Transient, Corruption, Infrastructure)
- `Family` methods: `String()`, `IsRetryable()`, `IsValid()`, `ExitCode()`, `DefaultMessage()`, `Audience()`, `Tone()`
- `ParseFamily()` — case-insensitive parser, defaults to `Transient`
- `Error` struct — full reference implementation with code, family, context, cause chain, timestamp
- `Error` methods: `Error()`, `Unwrap()`, `Is()`, `ErrorCode()`, `ErrorFamily()`, `ErrorContext()`, `IsRetryable()`, `Timestamp()`, `Family()`, `Code()`, `Message()`, `Cause()`, `Format()`, `WithContext()`, `WithCause()`, `Summary()`, `HasContext()`, `ContextValue()`, `MatchesContext()`, `MatchesContextValue()`
- Consumer interfaces (`Coded`, `Classified`, `Contextual`, `Retryable`) embed `error` for Go 1.26 `errors.AsType[T]()`

### Constructors ✅

- 15 factory functions: `New`, `Newf`, `Wrap`, `Wrapf` + family-specific variants (`NewRejection`, `WrapTransient`, etc.)
- Nil-safe: `Wrap(nil, ...) → nil`

### Classification Engine ✅

- `Classify(err)` — 4-step priority chain (Classified → Retryable → registered sentinel → default Transient)
- `RegisterClassification` / `RegisterClassifications` — thread-safe sentinel registration
- `IsRetryable(err)`, `ExitCode(err)` — convenience classifiers

### CLI Boundary Handler ✅

- `HandleError(err) int` — writes to stderr, returns exit code
- `HandleErrorWithConfig(err, cfg)` — customizable output, diagnostics, templates, callbacks
- `HandleErrorDetailed(err) *HandleResult` — programmatic, no output
- Template system: 4-level resolution (override → registered → built-in → family default)
- `RegisterTemplate()` — global template registration
- 10 built-in message templates for common error codes

### Diagnostic Framework ✅

- `DiagnosticRule` interface — `Name()`, `Applicable()`, `Run()`
- `Runner` — concurrent rule execution, confidence-sorted results
- 4 built-in rules: `PostgresRule`, `FilesystemRule`, `NetworkRule`, `GitRule`
- Matching helpers: `hasContextKey`, `contextValue`, `hasContextSubstring`, `familyIs`, `errorCodeContains`
- `DefaultRunner()`, `RunAuto()` — convenience functions

### AI Debug Agent ✅ (deterministic stub)

- `DebugAgent` interface with `Analyze()` method
- `AgentResult` with `RootCause`, `Confidence`, `Explanation`, `FixSteps`
- Deterministic analysis that maps failed diagnostics to root causes and fix suggestions
- `FixStep` with `Description`, `Command`, `Rationale`

### Project Infrastructure ✅

- MIT license, README with badges, CHANGELOG, AGENTS.md
- Zero external dependencies
- `go vet` clean, `gofmt -s` clean
- All tests pass

### Open Source Release ✅

- Published to Go module proxy (v0.1.1)
- GitHub: `LarsArtmann/go-error-family` — public
- Badges: Go Reference, Go Report Card

---

## b) PARTIALLY DONE

### Test Coverage (74.9% total, target: ~90%+)

| Package              | Coverage | Status             |
| -------------------- | -------- | ------------------ |
| root (`errorfamily`) | 90.8%    | 🟡 Good, not great |
| `agent`              | 100%     | ✅                 |
| `diagnose`           | 59.5%    | 🔴 Needs work      |

**Specific gaps:**

| What                                                                  | Where                        | Impact                                            |
| --------------------------------------------------------------------- | ---------------------------- | ------------------------------------------------- |
| `RegisterTemplate()` + `lookupTemplate()`                             | `handle.go`                  | 0% coverage on global template registration path  |
| `Family.DefaultMessage()`                                             | `family.go`                  | Never tested                                      |
| `Family.Tone()` — 3 of 5 families                                     | `family.go`                  | Untested for Conflict, Corruption, Infrastructure |
| `formatWhy()` / `suggestFix()` — Corruption & Infrastructure branches | `handle.go`                  | Untested branches                                 |
| All 10 `defaultMessages` entries                                      | `handle.go`                  | Only "file.not_found" and "db.connection" tested  |
| `runCommand()` non-ExitError paths                                    | `diagnose/context.go`        | Edge case: errors swallowed, exitCode=0 returned  |
| `GitRule.Run` — merge conflicts, dirty tree, unreachable remote       | `diagnose/rules_git.go`      | 17.3% coverage — worst in codebase                |
| `NetworkRule.resolvePort()`                                           | `diagnose/rules_network.go`  | Never tested                                      |
| `PostgresRule.suggestStartFix()`                                      | `diagnose/rules_postgres.go` | 0% coverage                                       |
| `AgentResult.Prevention`, `AgentResult.RelatedErrors`                 | `agent/agent.go`             | Fields exist but never populated or tested        |
| `Config.Timeout`                                                      | `agent/agent.go`             | Accepted but never enforced — dead config         |
| `buildPrompt()` output                                                | `agent/agent.go`             | Built but result discarded (`_ =`) — dead code    |
| `TestAnalyzeWithContext`                                              | `agent/agent_test.go`        | Result discarded, **zero assertions**             |

### Diagnostic Rules — Environment-Dependent Tests

- Rules make real system calls (DNS, TCP, git, pg_isready, filesystem)
- No mocking infrastructure exists
- Tests that shell out are flaky on CI environments without the expected tools
- This is the primary reason `diagnose` coverage is 59.5%

### AI Agent — Stub, Not Production

- Deterministic analysis only — no AI provider integration
- `buildPrompt()` result is discarded (dead code until real provider added)
- `Config.Timeout` is dead (never enforced)
- `AgentResult.Prevention` and `AgentResult.RelatedErrors` are unused fields

---

## c) NOT STARTED

| Item                                 | Priority | Notes                                                                                         |
| ------------------------------------ | -------- | --------------------------------------------------------------------------------------------- |
| Benchmarks                           | Medium   | Zero `Benchmark*` functions exist                                                             |
| Concurrent safety tests              | High     | No `-race` oriented tests for `RegisterClassification`, `RegisterTemplate`, `Runner.Register` |
| Fuzz tests                           | Low      | No `Fuzz*` functions — classification and parsing are fuzz-worthy                             |
| Examples (`func Example*`)           | Medium   | No Go documentation examples for godoc                                                        |
| CI pipeline (GitHub Actions)         | High     | No `.github/workflows/` — no automated test/lint on push                                      |
| Nix flake                            | Low      | `flake.nix` not present; `justfile` listed as deprecated in AGENTS.md                         |
| `Audience.String()` method           | Low      | `Audience` type has no `String()` unlike `Family` and `Status`                                |
| `Wrapf` family-specific constructors | Low      | `WrapfRejection`, `WrapfTransient`, etc. don't exist; inconsistent with `Newf` pattern        |
| Changelog for next release           | Low      | `[Unreleased]` section is empty                                                               |

---

## d) TOTALLY FUCKED UP

### pkg.go.dev Validation Error (External Bug)

**Status:** Not our fault, but blocking godoc rendering.

```
gofmt -s: "stat family.go: no such file or directory"
→ strconv.Atoi(" no such file or directory"): invalid syntax
```

Our code is `gofmt -s` clean on both HEAD and v0.1.1. This is a **parser bug in the validation system** — it hits a transient file-not-found during extraction, then its output parser crashes trying to parse the error message as a line number.

**Mitigation options:**

1. Wait and re-trigger indexing on pkg.go.dev
2. Re-trigger Go Report Card
3. Tag v0.1.2 (identical code, fresh tag) to force clean re-fetch

### `DiagnosticFinding` vs `DiagnosticResult` Type Duplication

`handle.go` defines `DiagnosticFinding` with the same fields as `diagnose.DiagnosticResult` (`Status`, `Summary`, `SuggestedFix`, `Confidence`). This exists to avoid circular imports. It's a maintenance risk — if one changes, the other must change manually. No automated check ensures they stay in sync.

### `Classify(nil) == Rejection` vs `ExitCode(nil) == 0`

Semantic inconsistency: classifying a nil error returns `Rejection` (caller's fault), but `ExitCode(nil)` returns 0 (success). This is documented and intentional but could confuse consumers.

### `runCommand()` Swallows Non-ExitError Errors

In `diagnose/context.go`, any error that isn't `*exec.ExitError` (e.g., context cancellation, command not found) is silently converted to `exitCode=0, err=nil`. This hides real problems from diagnostic rules.

### `TestAnalyzeWithContext` Has Zero Assertions

In `agent/agent_test.go`, line `result := a.Analyze(...)` is followed by `_ = result`. The test compiles and passes but verifies nothing.

### `hasContextSubstring(err, "git")` is Over-Broad

Matches "git" in any context value — would also match "digit", "legitimate", etc. Same pattern with `errorCodeContains(err, "timeout")` matching `app.session_timeout`.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Design

1. **Bridge `DiagnosticFinding` ↔ `DiagnosticResult`** — Create a thin adapter function with a compile-time check that the field sets match. Or extract shared types to a third package.
2. **Enforce `Config.Timeout`** — Add `context.WithTimeout` in `Analyze()`, or remove the field if the deterministic agent doesn't need it.
3. **Remove or populate dead fields** — `AgentResult.Prevention`, `AgentResult.RelatedErrors` are zero-valued forever. Either populate them or remove until the real agent exists.
4. **Remove `buildPrompt()` dead code** — The result is discarded. Either use it or delete it.
5. **Template engine escaping** — `applyContext()` uses `strings.ReplaceAll` with no escaping. Unresolved `{{.key}}` stays literal in output.
6. **Input validation** — `New()` and `WithContext()` accept empty code/family/key silently.
7. **`Classify(nil)` semantic** — Consider returning a zero `Family` or panicking. The current behavior (Rejection) is intentional but inconsistent with `ExitCode(nil) == 0`.

### Testing

8. **Add concurrent safety tests** — `go test -race` with parallel subtests for `RegisterClassification`, `RegisterTemplate`, `Runner.Register`.
9. **Add mocking for diagnostic rules** — Extract `runCommand` to an interface so rules can be unit-tested without system dependencies. This alone would push `diagnose` coverage from 59.5% to 90%+.
10. **Add Go example functions** — `ExampleNewRejection()`, `ExampleClassify()`, `ExampleHandleError()` for godoc.
11. **Fix `TestAnalyzeWithContext`** — Add actual assertions on the result.
12. **Add `defaultMessages` table-driven test** — All 10 built-in templates should be exercised.
13. **Test `RegisterTemplate` + `lookupTemplate`** — 0% coverage on the global template registration path.

### Operations

14. **Add CI pipeline** — GitHub Actions with `go test -race ./...`, `go vet`, `gofmt -s -l`, coverage enforcement.
15. **Address pkg.go.dev validation** — Tag v0.1.2 to re-trigger indexing.
16. **Add `Audience.String()`** — Consistency with `Family.String()` and `Status.String()`.

---

## f) Top 25 Things We Should Get Done Next

Ranked by impact × effort (Pareto):

### Tier 1 — High Impact, Low Effort (Do Immediately)

| #   | Item                                                                    | Effort | Impact                                         |
| --- | ----------------------------------------------------------------------- | ------ | ---------------------------------------------- |
| 1   | **Fix `TestAnalyzeWithContext` — add assertions**                       | 5 min  | Fixes a test that passes but verifies nothing  |
| 2   | **Remove `buildPrompt()` dead code** (or wire it)                       | 5 min  | Eliminates dead code in production path        |
| 3   | **Remove or gate `Config.Timeout`** — enforce it or delete it           | 10 min | Eliminates dead config that misleads consumers |
| 4   | **Remove `AgentResult.Prevention` and `RelatedErrors`** — unused fields | 5 min  | Eliminates dead struct fields                  |
| 5   | **Tag v0.1.2** — force pkg.go.dev re-indexing                           | 5 min  | Fixes godoc rendering                          |
| 6   | **Add `Audience.String()` method**                                      | 5 min  | Consistency across all enum types              |
| 7   | **Test `RegisterTemplate` + `lookupTemplate`**                          | 15 min | 0% → 100% on uncovered path                    |
| 8   | **Test `Family.DefaultMessage()` and all `Tone()` values**              | 15 min | Fills coverage gaps in core type               |
| 9   | **Add `defaultMessages` table-driven test** — all 10 templates          | 15 min | Exercises 80% of untested handle.go paths      |

### Tier 2 — High Impact, Medium Effort (Do This Week)

| #   | Item                                                                                  | Effort | Impact                                            |
| --- | ------------------------------------------------------------------------------------- | ------ | ------------------------------------------------- |
| 10  | **Add CI pipeline** (GitHub Actions) — `go test -race`, `go vet`, `gofmt -s -l`       | 1 hr   | Prevents regressions on every push                |
| 11  | **Add concurrent safety tests** — parallel subtests for global registries             | 30 min | Catches data races in production                  |
| 12  | **Extract `runCommand` to interface for mocking**                                     | 2 hr   | Unlocks 90%+ coverage for `diagnose`              |
| 13  | **Bridge `DiagnosticFinding` ↔ `DiagnosticResult`** — shared adapter or third package | 1 hr   | Eliminates maintenance risk from type duplication |
| 14  | **Fix `runCommand()` error swallowing** — return non-ExitError errors properly        | 30 min | Stops hiding real failures in diagnostics         |
| 15  | **Add Go example functions** for godoc                                                | 1 hr   | Improves discoverability on pkg.go.dev            |

### Tier 3 — Medium Impact, Medium Effort (Do This Sprint)

| #   | Item                                                                                                | Effort | Impact                                  |
| --- | --------------------------------------------------------------------------------------------------- | ------ | --------------------------------------- |
| 16  | **Tighten `hasContextSubstring` / `errorCodeContains` matching** — word boundaries or exact matches | 1 hr   | Reduces false positive rule triggers    |
| 17  | **Add input validation to `New()` / `WithContext()`** — reject empty code/family/key                | 30 min | Fail fast on invalid usage              |
| 18  | **Add benchmarks** for `Classify()`, `HandleError()`, `Runner.Run()`                                | 1 hr   | Performance regression detection        |
| 19  | **Add `formatWhy` / `suggestFix` tests for Corruption and Infrastructure**                          | 30 min | Completes branch coverage in handle.go  |
| 20  | **Template engine: escape unresolved `{{.key}}` in output**                                         | 30 min | Prevents confusing user-facing messages |

### Tier 4 — Lower Impact, Higher Effort (Backlog)

| #   | Item                                                                  | Effort | Impact                                      |
| --- | --------------------------------------------------------------------- | ------ | ------------------------------------------- |
| 21  | **Fuzz tests for `ParseFamily()` and `Classify()`**                   | 2 hr   | Finds edge cases in parsing                 |
| 22  | **Add `Wrapf` family-specific constructors** (`WrapfRejection`, etc.) | 30 min | API consistency                             |
| 23  | **Wire AI agent to actual provider** (OpenAI, Anthropic, etc.)        | 1 day  | Transforms stub into real feature           |
| 24  | **Add `flake.nix` for build/task automation**                         | 2 hr   | Standardizes dev environment                |
| 25  | **Resolve `Classify(nil)` semantic inconsistency**                    | 1 hr   | API clarity (breaking change consideration) |

---

## g) Top 1 Question I Cannot Figure Out Myself

**What is the intended AI provider integration path for the `DebugAgent`?**

The `buildPrompt()` method constructs a prompt string but discards it (`_ =`). The `Config` struct has `Enabled` and `Timeout` but no provider config (API key, model, endpoint). The deterministic analysis is a functional placeholder.

**Specifically:**

- Is the agent meant to call an external AI API at runtime, or is it designed for compile-time/embedded model integration?
- Should the provider be injected via the `DebugAgent` interface (consumer implements their own), or should the library provide a built-in provider?
- Is the `buildPrompt()` output format the intended contract, or is it expected to change?

This decision affects whether `Config` needs expanding, whether `buildPrompt()` should be exported, and how `Timeout` should be enforced.

---

## Metrics Summary

| Metric                   | Value                     |
| ------------------------ | ------------------------- |
| Total Go LOC             | 3,256                     |
| Production files         | 12                        |
| Test files               | 4                         |
| Total commits            | 20                        |
| Tests passing            | 100%                      |
| `go vet`                 | Clean                     |
| `gofmt -s`               | Clean                     |
| External dependencies    | 0                         |
| Packages                 | 3 (root, agent, diagnose) |
| Exported types           | ~15                       |
| Exported functions       | ~45                       |
| Test coverage (total)    | 74.9%                     |
| Test coverage (root)     | 90.8%                     |
| Test coverage (agent)    | 100%                      |
| Test coverage (diagnose) | 59.5%                     |
