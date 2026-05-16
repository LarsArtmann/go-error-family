# Status Report — go-error-family

**Date:** 2026-05-16 21:38
**Repo:** `github.com/larsartmann/go-error-family`
**Branch:** master
**Last commit:** `eb80d34` — test: add comprehensive test coverage for agent, diagnose, handle, and error packages
**Status:** Clean working tree. All tests pass (130/130). Coverage: root 87%, agent 97.4%, diagnose 54.8%.

---

## Executive Summary

The library has matured significantly since the initial status report on 2026-05-10. Three major gaps have been closed: test coverage jumped from 1 test file (26 tests) to 4 test files (130 tests), the `HandleError` CLI handler now wires to the diagnostic runner, and the AGENTS.md provides focused agent context.

**Remaining concerns:** The AI agent is still scaffold (no real provider), diagnostic rules still use direct system calls (54.8% coverage — the untestable parts), CHANGELOG is stale, and no external repo consumes the library yet.

---

## A) FULLY DONE ✓

### Core Protocol Package (`errorfamily`)

| Component                | File                  | Status                                                                                                                                    |
| ------------------------ | --------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `Family` int enum        | `family.go`           | Complete — 5 families, String/ParseFamily/IsRetryable/ExitCode/Tone/Audience                                                              |
| Consumer interfaces      | `interfaces.go`       | Complete — Coded, Classified, Contextual, Retryable (embed `error` for AsType)                                                            |
| Reference `Error` struct | `error.go`            | Complete — Is/Unwrap/Format/Context/Summary/MatchesContext/Timestamp                                                                      |
| Classification engine    | `classify.go`         | Complete — Classify/IsRetryable/ExitCode/RegisterClassification(s)                                                                        |
| Constructors             | `constructors.go`     | Complete — New/Newf/Wrap/Wrapf + 10 family-specific shortcuts                                                                             |
| CLI boundary handler     | `handle.go`           | Complete — HandleError/HandleErrorDetailed/MessageTemplate + diagnostics wiring                                                           |
| Root package tests       | `errorfamily_test.go` | 36 test cases — families, error struct, constructors, classification, chain traversal, timestamps, audience, builder patterns             |
| Handler tests            | `handle_test.go`      | 15 test cases — HandleError/HandleErrorWithConfig/HandleErrorDetailed, template overrides, diagnostics wiring, all families, plain errors |

### Diagnostic System (`diagnose`)

| Component           | File                  | Status                                                                                                                  |
| ------------------- | --------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Runner + interfaces | `diagnose.go`         | Complete — concurrent execution, confidence sort, variable shadowing fixed                                              |
| System snapshot     | `context.go`          | Complete — dead fields (DiskFree, Uptime) removed, secret redaction                                                     |
| PostgresRule        | `rules_postgres.go`   | Complete — pg_isready, TCP fallback, platform-aware fixes                                                               |
| FilesystemRule      | `rules_filesystem.go` | Complete — stat, permissions, writability, auto-fix mkdir                                                               |
| NetworkRule         | `rules_network.go`    | Complete — DNS, TCP. Fixed: no longer fires on ALL Transient errors                                                     |
| GitRule             | `rules_git.go`        | Complete — repo check, dirty state, merge conflicts, remote                                                             |
| Diagnose tests      | `diagnose_test.go`    | 47 test cases — Runner, all Applicable methods, all Run methods for local paths, helper functions, concurrent execution |

### AI Debug Agent (`agent`)

| Component           | File            | Status                                                                                                                               |
| ------------------- | --------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Full agent scaffold | `agent.go`      | Complete — 4 involvement levels, risk classification, deterministic fallback                                                         |
| Agent tests         | `agent_test.go` | 20 test cases — all involvement levels, Analyze (enabled/disabled/with diagnosis/empty), ApplyFixes, Config defaults, extractCommand |

### Documentation & Infrastructure

| Doc                        | Status                                                                                                   |
| -------------------------- | -------------------------------------------------------------------------------------------------------- |
| README.md                  | Complete — quick start, constructors, classification, custom types, diagnostics, agent, architecture     |
| Design document (in docs/) | Complete — 1,119 lines                                                                                   |
| AGENTS.md                  | Complete — focused on non-obvious knowledge (surprising behaviors, classification precedence, test gaps) |
| git-town.toml              | Present                                                                                                  |
| AUTHORS                    | Present                                                                                                  |

---

## B) PARTIALLY DONE 🔧

### Diagnostic Rules — Testing Gaps Remain

The `diagnose_test.go` covers Applicable methods, Runner mechanics, and Run methods for local/available paths. But:

- **PostgresRule.Run** — only tested when pg_isready is absent (TCP fallback). No test against real or mock PostgreSQL.
- **GitRule.Run** — tested against current repo (which is clean) and /tmp (not a repo). No test for merge conflicts, dirty state, or unreachable remotes.
- **NetworkRule.Run** — no test for actual DNS resolution or TCP connectivity (only the host resolution helper).
- **FilesystemRule** — tested with nonexistent paths and existing files, but no test for permission-denied scenarios or the auto-fix callback.

Coverage: 54.8% — the uncovered 45% is mostly the system-call paths in rules.

### AI Debug Agent — Scaffold Only

Agent tests prove the involvement/approval logic works correctly, but:

- No actual AI provider integration (deterministicAnalyze only)
- `ApplyFixes` marks steps as applied but doesn't execute commands
- `buildPrompt` constructs a prompt string that goes unused
- The allowed/forbidden command glob matching is untested

### CLI Boundary Handler — Wired but Basic

HandleConfig now calls DiagnosticRunner.Run when `cfg.Diagnose` is true and wires OnDiagnosed callback. But:

- `codeToWhat()` and `codeToFix()` use fragile `strings.Contains` matching
- No default MessageTemplate overrides registered
- `HandleErrorDetailed()` returns structured data but no consumer uses it
- Verbose output formatting exists in HandleConfig but verbose diagnostic printing was partially implemented and then reverted

---

## C) NOT STARTED ✗

### Integration with Any Consumer Repo

Zero repos import `go-error-family`. The library exists in isolation.

| Repo                    | What needs to happen                                                       | Status      |
| ----------------------- | -------------------------------------------------------------------------- | ----------- |
| **go-cqrs-lite**        | Extract Family/Classify/RegisterClassification into go-error-family import | Not started |
| **go-finding**          | Add ErrorCode()/ErrorContext() methods to FindingError                     | Not started |
| **hierarchical-errors** | Add ErrorCode() to its Error struct                                        | Not started |
| **auto-deduplicate**    | Add ErrorFamily()/ErrorContext(), move Present() to CLI layer              | Not started |
| **docs-organizer**      | Add Is, ErrorCode, ErrorContext to DocsError                               | Not started |

### CI/CD Pipeline

No GitHub Actions, no linting pipeline, no release automation.

### Go Module Publishing

Module is on GitHub but has no tagged releases. No `v0.1.0`.

### Linting

No `.golangci.yml` configuration. gopls still warns about:

- `handle.go`: 3 unused parameters in `formatWhy`, `applyTemplate` (intentional, reserved for future)
- No other warnings

### Nix Flake

No `flake.nix` — the LarsArtmann ecosystem standard for build automation.

### ADR (Architecture Decision Records)

No formal ADRs in `docs/adr/`.

### CHANGELOG

CHANGELOG.md is a skeleton — does not reflect any of the actual work done since initial commit.

---

## D) TOTALLY FUCKED UP 💥

### Nothing Is Broken Right Now

Build is clean. All 130 tests pass. No compile errors. No panics. `go vet` clean.

### Historical Fuck-ups (Fixed This Session)

1. **`handle.go` had broken uncommitted changes** — An earlier session added diagnostic wiring but introduced undefined `diagnostics` variable, causing build failure. Fixed by reverting and re-implementing correctly.

2. **NetworkRule over-triggering** — Previously matched ALL `Transient`-family errors, meaning any transient database timeout would trigger DNS + TCP diagnostics. Fixed to match specific network-related substrings only.

3. **Variable shadowing in Runner.Run** — Closure parameter `rule` shadowed the outer loop variable, `err` shadowed the function parameter, and `r` in the filter loop shadowed the receiver. Fixed by renaming.

### Still Problematic (Not Broken, But Wrong)

1. **`applyTemplate` has an unused `family` parameter** — Reserved for future use but gopls complains. Intentional technical debt.

2. **`formatWhy` has unused `code` and `context` parameters** — Same. The Wix Why section is only family-based currently.

3. **Agent `ApplyFixes` is performative theater** — Marks steps as applied without executing anything. Autonomous involvement level approves everything and does nothing.

4. **`classify.go` sentinel registry uses `errors.Is` under RLock** — If a target's `Is()` method tries to classify (calling back into `Classify`), this could deadlock. Theoretical only, but lock ordering is fragile.

---

## E) WHAT WE SHOULD IMPROVE 📈

### Architecture

1. **Wire AI agent to a real provider** — The scaffold proves the design works. Time to integrate OpenAI/Anthropic/Crush SDK.
2. **Add `Mark(err, sentinel)` function** — Identity stamping without global registry, inspired by cockroachdb/errors.
3. **Extract CommandRunner interface** — Makes diagnostic rules unit-testable without PostgreSQL/Git on PATH.

### Quality

4. **Update CHANGELOG.md** — Currently a skeleton. Should reflect all 8 commits since initial release.
5. **Add `.golangci.yml`** — Consistent linting with the rest of the ecosystem.
6. **Add CI pipeline** — GitHub Actions for build + test + vet on push/PR.
7. **Add `flake.nix`** — LarsArtmann ecosystem standard for build/test automation.
8. **Increase diagnose coverage** — Integration tests for rules that shell out, or extract interfaces for mockability.

### Ecosystem

9. **Tag `v0.1.0-alpha`** — Signals API stability expectations. Module is on GitHub but has no version.
10. **Integrate with first consumer repo** — Proves the protocol works in practice.
11. **Write ADR-001** — Document why 5 families, why int not string, why embed error in interfaces.

---

## F) TOP 25 THINGS TO DO NEXT

### Critical — Quality Gates

| #   | Task                                                            | Effort | Impact                 |
| --- | --------------------------------------------------------------- | ------ | ---------------------- |
| 1   | Update CHANGELOG.md with all changes since initial commit       | 30min  | Historical accuracy    |
| 2   | Fix remaining gopls warnings (unused params) or document intent | 15min  | Zero-warning hygiene   |
| 3   | Tag `v0.1.0-alpha`                                              | 5min   | API stability signal   |
| 4   | Add GitHub Actions CI (build + test + vet)                      | 1h     | Automated quality gate |
| 5   | Add `.golangci.yml`                                             | 30min  | Consistent linting     |

### High — Test Coverage

| #   | Task                                                                            | Effort | Impact                                   |
| --- | ------------------------------------------------------------------------------- | ------ | ---------------------------------------- |
| 6   | Integration tests for GitRule (dirty repo, merge conflicts, unreachable remote) | 1h     | GitRule only tested for clean repo       |
| 7   | Integration tests for PostgresRule (mock pg_isready, TCP server)                | 1h     | PostgresRule TCP path untested           |
| 8   | Integration tests for NetworkRule (DNS resolution, TCP connect, timeout)        | 1h     | NetworkRule only tested for host parsing |
| 9   | Extract CommandRunner/ConnectionTester interfaces for mockability               | 1h     | Enables unit tests without system tools  |
| 10  | Test FilesystemRule auto-fix callback (mkdir in temp dir)                       | 30min  | AutoFix path never executed in tests     |

### High — Ecosystem Integration

| #   | Task                                                          | Effort | Impact                |
| --- | ------------------------------------------------------------- | ------ | --------------------- |
| 11  | Migrate go-cqrs-lite to import go-error-family                | 2h     | First real consumer   |
| 12  | Add go-error-family to projects-management-automation go.work | 15min  | Workspace integration |
| 13  | Add ErrorCode()/ErrorContext() to go-finding FindingError     | 30min  | Second consumer       |
| 14  | Add ErrorCode()/ErrorContext() to docs-organizer DocsError    | 30min  | Third consumer        |

### Medium — Feature Completeness

| #   | Task                                                         | Effort | Impact                                |
| --- | ------------------------------------------------------------ | ------ | ------------------------------------- |
| 15  | Wire AI agent to real provider (OpenAI/Anthropic SDK)        | 3h     | Agent actually works                  |
| 16  | Implement actual command execution in ApplyFixes (sandboxed) | 2h     | Autonomous mode is not a lie          |
| 17  | Add `Mark(err, sentinel)` identity stamping                  | 30min  | Alternative to RegisterClassification |
| 18  | Add default MessageTemplate overrides for common error codes | 1h     | Better out-of-box UX                  |
| 19  | Add golangci.yml configuration                               | 30min  | Consistent linting                    |
| 20  | Add flake.nix for build/test automation                      | 1h     | Ecosystem standard                    |

### Lower — Polish

| #   | Task                                                   | Effort | Impact                     |
| --- | ------------------------------------------------------ | ------ | -------------------------- |
| 21  | Write ADR-001: Why Family int over string              | 30min  | Architecture documentation |
| 22  | Write examples/ directory with runnable Go examples    | 1h     | GoDoc integration          |
| 23  | Add IsPostgresRunning helper to consumer repos         | 15min  | Useful utility             |
| 24  | Update README with v0.1. status and installation badge | 15min  | Professional appearance    |
| 25  | Benchmark Classify() performance for hot-path usage    | 30min  | Performance baseline       |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF ❓

**Should this library stay as a zero-dependency stdlib-only package, or is it acceptable to add an AI provider SDK as an optional dependency?**

The current architecture has `agent/` as a scaffold that returns deterministic results. Making it useful requires an AI provider (OpenAI, Anthropic, or Crush SDK). But:

- **Zero-dependency is a selling point** — the README implicitly promises "no external deps". Adding `go-openai` breaks that promise.
- **Optional dependency via Go build tags** — could put the real provider behind `//go:build !noai` but this is unusual and confusing.
- **Separate package** — `agent/openai/` as a separate module with its own go.mod. Clean separation but more maintenance.
- **Consumer provides the provider** — The `DebugAgent` is already an interface. Consumers could inject their own AI client. Library stays dependency-free. But then the agent package is just types + scaffold.

The rest of the LarsArtmann Go ecosystem (go-cqrs-lite, go-finding) is stdlib-only. But those don't have AI integration points.

My recommendation: **Consumer provides the provider** — keep the library zero-dependency, document how to implement DebugAgent with any AI SDK. But this is a product direction decision I can't make alone.

---

## Metrics

| Metric              | Value                         | Previous (2026-05-10) | Delta                    |
| ------------------- | ----------------------------- | --------------------- | ------------------------ |
| Total Go lines      | 3,737                         | 2,644                 | +1,093                   |
| Non-test Go lines   | 2,160                         | ~2,644                | -484 (dead code removed) |
| Test lines          | 1,577                         | 397                   | +1,180                   |
| Go files            | 17                            | 14                    | +3 (test files)          |
| Test files          | 4                             | 1                     | +3                       |
| Test cases          | 130                           | 26                    | +104                     |
| Test pass rate      | 100% (130/130)                | 100% (26/26)          | —                        |
| Coverage (root)     | 87.0%                         | ~70% (est)            | +17%                     |
| Coverage (agent)    | 97.4%                         | 0%                    | +97.4%                   |
| Coverage (diagnose) | 54.8%                         | 0%                    | +54.8%                   |
| Build errors        | 0                             | 0                     | —                        |
| `go vet` issues     | 0                             | 0                     | —                        |
| gopls warnings      | 3 (intentional unused params) | 9                     | -6                       |
| Packages            | 3                             | 3                     | —                        |
| Dependencies        | 0 (stdlib only)               | 0                     | —                        |
| Consumers           | 0                             | 0                     | —                        |
| Commits             | 8                             | 1                     | +7                       |
| Tagged releases     | 0                             | 0                     | —                        |

---

_Arte in Aeternum_
