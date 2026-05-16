# go-error-family — Full Status Report

**Date:** 2026-05-16 23:19
**Branch:** master
**Head:** `3c43687` — refactor: eliminate production code clones (24→21 groups)
**Commit count (this project):** 16
**Working tree:** clean

---

## A. FULLY DONE ✓

### A1. Core Library — Production-Ready

The library delivers on its promise: **structured error protocol for Go, zero dependencies, Go 1.26+**.

| Component         | Lines | Status                                                                                                                                             |
| ----------------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `interfaces.go`   | 41    | ✓ Complete — 4 consumer interfaces (Coded, Classified, Contextual, Retryable), each embedding `error` for `errors.AsType[T]()`                     |
| `error.go`        | 184   | ✓ Complete — reference `Error` struct with `Is`, `Unwrap`, `Format` (+verbose), context, convenience accessors                                     |
| `family.go`       | 140   | ✓ Complete — data-driven `familyData` registry, `Family.DefaultMessage()`, `ParseFamily`, `IsValid`, `IsRetryable`, `ExitCode`, `Tone`, `Audience` |
| `classify.go`     | 101   | ✓ Complete — 4-tier classification (Classified → Retryable → Registered sentinels → Default Transient), lock-free snapshot in `lookupRegistered`   |
| `constructors.go` | 99    | ✓ Complete — 5 family-specific `New*` + 5 `Wrap*` + `New`/`Newf`/`Wrap`/`Wrapf`                                                                    |
| `handle.go`       | 330   | ✓ Complete — `HandleError` (CLI boundary), `HandleErrorDetailed` (programmatic), template registry, Wix-style messages, `RegisterTemplate`         |

### A2. Diagnostic Rules — Complete

| Rule                           | Lines | Status                                                                                                          |
| ------------------------------ | ----- | --------------------------------------------------------------------------------------------------------------- |
| `diagnose/diagnose.go`         | 253   | ✓ Runner with concurrent execution, confidence-sorted results, `resolveContextKey` helper, all matching helpers |
| `diagnose/context.go`          | 38    | ✓ `runCommand` + `commandExists`                                                                                |
| `diagnose/rules_postgres.go`   | 131   | ✓ pg_isready, TCP connectivity, service start suggestions                                                       |
| `diagnose/rules_filesystem.go` | 143   | ✓ Path existence, permissions, writability, readability                                                         |
| `diagnose/rules_network.go`    | 97    | ✓ DNS resolution, TCP connectivity, URL normalization via `stripAfter`                                          |
| `diagnose/rules_git.go`        | 116   | ✓ Repo existence, clean state, merge conflicts, remote reachability                                             |

### A3. AI Debug Agent — Honest Analysis-Only

| Component        | Lines | Status                                                                         |
| ---------------- | ----- | ------------------------------------------------------------------------------ |
| `agent/agent.go` | 176   | ✓ Analysis-only agent — `Analyze` produces `FixStep` suggestions, no execution |

### A4. Architectural Cleanup (Commit `167fbd9`) — Complete

All 9 phases executed:

1. ✓ Dead code removal (SystemSnapshot, FixResult, dead fields)
2. ✓ Fraud removal (ApplyFixes, Involvement, RiskLevel, execution layer)
3. ✓ Concurrency fix (lookupRegistered lock-free snapshot)
4. ✓ Template registry (exact-match replacing magic substring matching)
5. ✓ Type safety (DiagnosticFunc replacing `any` return)
6. ✓ Split brain removal (HandleResult.Diagnostics, ErrorReported, Verbose)
7. ✓ Polish (Family.IsValid)
8. ✓ Docs (AGENTS.md updated)
9. ✓ Verify (121 tests, -race clean)

### A5. Production Code Deduplication (Commit `3c43687`) — Complete

- ✓ `familyData` registry — replaced 3 switch-on-Family methods with single array lookup
- ✓ `Family.DefaultMessage()` — absorbed `familyDefaultMessage()` switch, then inlined
- ✓ `resolveContextKey()` — shared helper for 4 rule files (GitRule, PostgresRule, NetworkRule, FilesystemRule)
- ✓ `stripAfter()` — URL normalization helper
- ✓ Test helpers in `handle_test.go` — `testDiagnosticFunc`, `testOnDiagnosedPtr`

### A6. Documentation

- ✓ AGENTS.md — non-obvious knowledge, classification precedence, template system docs
- ✓ docs/top-5-stupidest-things.md — analysis document
- ✓ docs/resolving-top-5-stupidest-things.md — resolution plan
- ✓ docs/planning/2026-05-16_22-32_architectural-cleanup.md — full execution plan
- ✓ README.md — quick start, architecture, philosophy

---

## B. PARTIALLY DONE ⚠️

### B1. AGENTS.md Coverage Table is Stale

The AGENTS.md coverage table still shows 88.3% for root package. Actual current coverage is **90.8%** (improved by the `Family.DefaultMessage()` method now being covered). Needs update.

### B2. README.md is Stale

The README still documents:

- `agent.DefaultConfig()` — **does not exist** anymore (removed in fraud cleanup)
- `Involvement` enum with Silent/Suggest/Assist/Autonomous levels — **removed** (agent is now analysis-only)
- `ConfirmFunc` — **removed**
- Architecture listing mentions `SystemSnapshot` in `context.go` — **deleted**
- Architecture listing shows old line counts
- The AI debug agent section's entire "involvement levels" table is obsolete

### B3. CHANGELOG.md is Empty

Only has template entries for `[Unreleased]` and `[0.1.0]` with "Initial release". No entries for the architectural cleanup, deduplication, or any real changes.

### B4. Diagnose Package Coverage — 59.5%

The diagnose package has 59.5% coverage. This is partially justified (rules shell out to system commands like `pg_isready`, `git` — integration-test territory), but the matching helpers (`hasContextKey`, `contextValue`, `hasContextSubstring`, `errorCodeContains`, `resolveContextKey`) and `Runner.Run` concurrent logic could have better unit test coverage.

---

## C. NOT STARTED ○

### C1. Version Tag

No git tags exist. No `v0.1.0` or any version. The `go.mod` just says `go 1.26.2`. For a library that others import, semantic versioning matters.

### C2. CI/CD Pipeline

No `.github/workflows/`, no Makefile, no flake.nix, no CI configuration. Tests only run manually. The project relies on the developer remembering `GOWORK=off go test -race ./...`.

### C3. Go Reference Documentation (pkg.go.dev)

No `package`-level doc examples (`func ExampleNewRejection()`). The godoc will render bare API without runnable examples, which is the primary discovery surface for Go libraries.

### C4. Integration Tests

No integration tests for diagnostic rules that shell out. The unit tests mock around command execution. A CI environment could run real `git init`, create temp files, etc.

### C5. Fuzzing

`Classify`, `ParseFamily`, and the template system (`applyContext`) accept arbitrary strings. No fuzz tests exist. These are prime fuzzing targets.

### C6. Benchmarks

No benchmarks exist. For a library that might sit in hot error paths, understanding the allocation profile of `Classify()`, `HandleError()`, and `Error.Error()` matters.

### C7. API Stability Guarantee

No `go` directive for toolchain, no compatibility promise, no deprecation policy. Early consumers have no guarantee that `HandleError` won't change signature.

### C8. Error Chain Traversal Performance

`lookupRegistered` snapshots the entire map on every `Classify` call. If `RegisterClassification` is called many times, this snapshot grows. No lazy evaluation or caching.

---

## D. TOTALLY FUCKED UP 💥

### D1. README Documents Non-Existent API

**Severity: HIGH.** The README's "AI Debug Agent" section shows code that literally cannot compile:

```go
cfg := agent.DefaultConfig()       // DOES NOT EXIST — function was deleted
cfg.Involvement = agent.InvolvementSuggest  // DOES NOT EXIST — type was deleted
cfg.ConfirmFunc = func(...) {}     // DOES NOT EXIST — field was deleted
```

Anyone following the README will get compile errors. This is the worst possible first impression for a library.

### D2. README Architecture Section References Deleted Code

The architecture listing says `context.go — SystemSnapshot, command runner, secret redaction`. `SystemSnapshot` was deleted. `context.go` now only has `runCommand` and `commandExists` (38 lines, not the sprawling file implied).

### D3. AGENTS.md Template Tier List References Deleted Function

Line 56 says `familyDefaultMessage(family)` — this function was deleted in commit `3c43687`. The correct tier 4 is now `family.DefaultMessage()`.

---

## E. WHAT WE SHOULD IMPROVE

### E1. README Must Match Reality

The README is the #1 asset for a library. Right now it actively lies about the API. This should be fixed immediately — delete the entire involvement/execution section, replace with honest analysis-only agent documentation.

### E2. CHANGELOG Should Document Real Changes

The CHANGELOG is a ghost town. The project has had 16 commits including major architectural surgery. The CHANGELOG should capture this for anyone tracking the library.

### E3. Coverage Table in AGENTS.md

Trivial fix: update 88.3% → 90.8%.

### E4. godoc Examples

Go developers find libraries through pkg.go.dev. Without `Example*` functions, the documentation page shows a bare API. Adding 3-5 examples for `NewRejection`, `HandleError`, `Classify`, and `RegisterClassification` would dramatically improve discoverability.

### E5. Diagnose Package Test Coverage

The 59.5% figure is defensible for the rule `Run` methods (system commands), but the core `Runner.Run` concurrent logic, the matching helpers, and `resolveContextKey` should have better coverage. These are pure logic — no system dependency.

### E6. Version Tag

`git tag v0.1.0` and push. Without it, consumers must use pseudoversions from commit hashes.

### E7. `formatWhy` Takes Unused Parameters

`handle.go:202` — `formatWhy(_ string, _ map[string]string, family Family)` ignores its first two parameters. This is a code smell that will confuse readers. Either use the parameters or change the signature.

### E8. `DefaultMessage` Not Exported on `familyInfo`

`familyData[f].Message` is accessed via `Family.DefaultMessage()`. Good. But if someone adds a field to `familyInfo` and forgets to add a method, it silently doesn't work. Consider documenting the pattern.

### E9. `Error.MatchesContext` and `Error.MatchesContextValue` Are Package-Internal Duplication

These methods on `*Error` duplicate logic that exists as package-level helpers in `diagnose/diagnose.go` (`hasContextKey`, `hasContextSubstring`). Two separate implementations of the same concept.

---

## F. TOP #25 THINGS TO DO NEXT

**Priority-ordered. Pareto principle applied.**

### Tier 1: Honesty (1% effort, 51% impact)

| #   | Task                                                | Effort | Impact                                |
| --- | --------------------------------------------------- | ------ | ------------------------------------- |
| 1   | **Fix README — remove deleted API (agent section)** | 15min  | Prevents compile errors for new users |
| 2   | **Fix README — update architecture listing**        | 10min  | Matches reality                       |
| 3   | **Update CHANGELOG.md with real changes**           | 20min  | Honest project history                |
| 4   | **Fix AGENTS.md coverage table (88.3% → 90.8%)**    | 2min   | Correct docs                          |
| 5   | **Fix AGENTS.md template tier 4 reference**         | 2min   | Correct docs                          |

### Tier 2: Quality (4% effort, 64% impact)

| #   | Task                                                                                              | Effort | Impact                            |
| --- | ------------------------------------------------------------------------------------------------- | ------ | --------------------------------- |
| 6   | **Add godoc examples** (`ExampleNewRejection`, `ExampleHandleError`, `ExampleClassify`)           | 1hr    | pkg.go.dev renders properly       |
| 7   | **Improve diagnose test coverage** (Runner concurrent logic, matching helpers, resolveContextKey) | 2hr    | 59.5% → 75%+                      |
| 8   | **Add fuzz tests** for `Classify`, `ParseFamily`, `applyContext`                                  | 1hr    | Catches panics on arbitrary input |
| 9   | **Add benchmarks** for `Classify`, `HandleError`, `Error.Error`                                   | 1hr    | Performance profile for hot paths |
| 10  | **Clean up `formatWhy` unused parameters**                                                        | 5min   | Remove code smell                 |

### Tier 3: Infrastructure (20% effort, 80% impact)

| #   | Task                                                                           | Effort | Impact                              |
| --- | ------------------------------------------------------------------------------ | ------ | ----------------------------------- |
| 11  | **Tag v0.1.0**                                                                 | 1min   | Consumers get a real version        |
| 12  | **Add GitHub Actions CI** (test -race, go vet, build on 1.26)                  | 30min  | Automated quality gate              |
| 13  | **Add `go test` integration tests** for diagnostic rules (temp dirs, git init) | 2hr    | Real coverage for rule logic        |
| 14  | **Remove or wire `Error.MatchesContext` / `MatchesContextValue`**              | 30min  | Eliminate cross-package duplication |
| 15  | **Add `go vet` and `staticcheck` to CI**                                       | 15min  | Catch issues automatically          |

### Tier 4: Polish

| #   | Task                                                                                   | Effort | Impact                     |
| --- | -------------------------------------------------------------------------------------- | ------ | -------------------------- |
| 16  | **Review all exported symbols for naming quality**                                     | 1hr    | Professional API surface   |
| 17  | **Add CONTRIBUTING.md**                                                                | 30min  | Community readiness        |
| 18  | **Review godoc on all exported types/functions**                                       | 1hr    | Professional documentation |
| 19  | **Add error chain diagram to README**                                                  | 30min  | Conceptual clarity         |
| 20  | **Consider `HandleError` → `cli` subpackage** (mentioned in resolution plan, not done) | 2hr    | Separation of concerns     |

### Tier 5: Hardening

| #   | Task                                                                                   | Effort | Impact                          |
| --- | -------------------------------------------------------------------------------------- | ------ | ------------------------------- |
| 21  | **Performance audit of `lookupRegistered` snapshot** (full map copy on every Classify) | 1hr    | Hot path optimization           |
| 22  | **Add `Family.MarshalJSON` / `UnmarshalJSON`** for API serialization                   | 30min  | REST/gRPC friendliness          |
| 23  | **Add `Error.MarshalJSON` / `UnmarshalJSON`**                                          | 30min  | Structured logging friendliness |
| 24  | **Consider `errors.Join` support** for multi-error scenarios                           | 1hr    | Go 1.20+ compatibility          |
| 25  | **Consider `context.Context` integration** for error propagation                       | 2hr    | Distributed tracing             |

---

## G. TOP #1 QUESTION

**The README shows an "AI Debug Agent" with configurable involvement levels, but the code was stripped to analysis-only during the architectural cleanup. Is the intent to:**

1. **Keep the agent analysis-only** (current state) — the README should be rewritten to show `agent.New(Config{Enabled: true})` → `Analyze()` → `FixStep` suggestions only, with the consumer deciding what to do with the commands?
2. **Restore some form of execution layer** — but in a separate consumer package, not in the protocol library?

The README currently claims option 2 existed, but the code implements option 1. This is the most consequential design decision outstanding — it determines the README rewrite scope and the agent package's long-term API surface.

---

## Metrics Summary

| Metric                        | Value                                   |
| ----------------------------- | --------------------------------------- |
| Production lines              | 1,849                                   |
| Test lines                    | 1,408                                   |
| Test-to-code ratio            | 0.76:1                                  |
| Test cases                    | 121 (218 including subtests)            |
| Root package coverage         | 90.8%                                   |
| Agent package coverage        | 100%                                    |
| Diagnose package coverage     | 59.5%                                   |
| Clone groups (art-dupl -t 15) | 21 (down from 26 pre-cleanup)           |
| External dependencies         | 0                                       |
| Go version                    | 1.26.2                                  |
| Git commits                   | 16                                      |
| Git tags                      | 0                                       |
| Files in root package         | 7 prod + 2 test                         |
| Files in diagnose/            | 6 prod + 1 test                         |
| Files in agent/               | 1 prod + 1 test                         |
| Largest file                  | `diagnose/diagnose_test.go` (481 lines) |
| Largest prod file             | `handle.go` (330 lines)                 |
