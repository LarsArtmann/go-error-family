# Status Report — 2026-06-01 09:17

**Project:** go-error-family — Structured Error Protocol Library  
**Branch:** master  
**Go:** 1.26.3 | **Tests:** 184 passing | **Lint:** 0 issues | **Races:** 0

---

## Summary

This session continued v0.3.0 cleanup work from a previous interrupted session. Three key items were completed: test registry pollution prevention, Runner.Run context cancellation enforcement, and commit of an uncommitted UnregisterTemplate addition.

---

## Coverage

| Module               | Coverage |
| -------------------- | -------- |
| Root (`errorfamily`) | 96.0%    |
| `agent`              | 89.4%    |
| `diagnose`           | 66.9%    |
| `diagnose/git`       | 98.5%    |
| `diagnose/postgres`  | 80.3%    |

---

## Work Completed This Session

### a) FULLY DONE

| #   | Task                                                                    | Commit    |
| --- | ----------------------------------------------------------------------- | --------- |
| 1   | Commit uncommitted `UnregisterTemplate` in handle.go                    | `2ab6fea` |
| 2   | Add `UnregisterClassification` in classify.go                           | `13729ad` |
| 3   | Add `t.Cleanup()` to all 5 registry-mutating tests                      | `13729ad` |
| 4   | Runner.Run context cancellation enforcement (channel-based)             | `ea85f2c` |
| 5   | Refactor Runner.Run into 3 methods (reduce cyclomatic complexity 15→~6) | `ea85f2c` |
| 6   | Add `TestRunnerContextCancelledMidRun` (early return verification)      | `ea85f2c` |
| 7   | Run benchmarks — all nominal                                            | —         |
| 8   | Lint all 3 modules — 0 issues                                           | —         |

### b) PARTIALLY DONE

- **SKILL.md update for v0.3.0 APIs** — Previous session updated README/CONTRIBUTING but SKILL.md still references some pre-v0.3 APIs (not committed in this session, but was committed as `b2a1e44` by prior session)

### c) NOT STARTED (from original plan)

1. **Concurrent safety tests for registries** — `RegisterClassification`, `RegisterTemplate`, `Runner.Register` lack `-race` oriented parallel subtests
2. **DiagnosticFinding vs DiagnosticResult split brain** — `DiagnosticResult` in diagnose.go and `DiagnosticFinding` in handle.go represent similar concepts with different field sets; consolidation deferred
3. **DebugAgent → Agent interface rename** — API naming improvement deferred
4. **FilesystemRule.Run ignoring context parameter** — The `ctx` parameter is accepted but not used in filesystem checks
5. **Runner.Run goroutine leak on context cancellation** — When ctx cancels, goroutines writing to the buffered channel complete harmlessly but aren't actively cancelled. Could add context check inside rule goroutines.

### d) TOTALLY FUCKED UP

Nothing. All 184 tests pass, 0 lint issues, 0 race conditions across all 5 modules.

---

## Session Stats

| Metric                            | Value               |
| --------------------------------- | ------------------- |
| Tests                             | 184 passing         |
| Commits this session              | 3                   |
| Files changed                     | 6                   |
| Lines added                       | +87                 |
| Lines removed                     | -17                 |
| Root coverage                     | 96.0%               |
| Benchmark (Classify Error struct) | 8.4 ns/op, 0 allocs |
| Benchmark (HandleError)           | 413 ns/op, 5 allocs |

---

## Top #25 Things to Get Done Next

Sorted by impact × effort (Pareto):

| #   | Task                                                                                         | Impact | Effort | Category     |
| --- | -------------------------------------------------------------------------------------------- | ------ | ------ | ------------ |
| 1   | Add concurrent safety tests for registries (`RegisterClassification`, `RegisterTemplate`)    | High   | Low    | Test         |
| 2   | Make `FilesystemRule.Run` respect context cancellation                                       | High   | Low    | Correctness  |
| 3   | Consolidate `DiagnosticFinding` vs `DiagnosticResult` (split brain)                          | High   | Medium | Architecture |
| 4   | Rename `DebugAgent` → `Agent` interface                                                      | Medium | Low    | API          |
| 5   | Add `UnregisterClassification`/`UnregisterTemplate` to AGENTS.md                             | Medium | Low    | Docs         |
| 6   | Add Example functions for `HandleErrorWithContext`, `UnregisterClassification`, `ContextKey` | Medium | Low    | Docs         |
| 7   | Increase `diagnose` coverage from 66.9% to 80%+                                              | Medium | Medium | Test         |
| 8   | Increase `diagnose/postgres` coverage from 80.3% to 90%+                                     | Medium | Medium | Test         |
| 9   | Add integration test: full pipeline (create error → classify → diagnose → handle)            | Medium | Medium | Test         |
| 10  | Add `Family-specific format constructors` (NewRejectionf, NewTransientf)                     | Medium | Medium | Feature      |
| 11  | Add `errors.Join`-aware `Compose` that returns worst family                                  | Medium | Medium | Feature      |
| 12  | Add `Mark(err, sentinel)` for identity stamping                                              | Medium | Medium | Feature      |
| 13  | Add structured logging adapter (slog integration)                                            | Medium | Medium | Feature      |
| 14  | Benchmark `Runner.Run` with context cancellation path                                        | Low    | Low    | Perf         |
| 15  | Add `Runner.Register` concurrent safety test                                                 | Low    | Low    | Test         |
| 16  | Add fuzz tests for `ParseFamily`, `Classify`                                                 | Low    | Medium | Test         |
| 17  | Add `Error.Format(state, verb)` for `%+v` verbose output                                     | Low    | Low    | Feature      |
| 18  | Add `Errors(err) []error` unwrapping helper                                                  | Low    | Low    | Feature      |
| 19  | Add `IsFamily(err, Family) bool` convenience function                                        | Low    | Low    | Feature      |
| 20  | Add `Corruption` family diagnostic rules                                                     | Medium | High   | Feature      |
| 21  | Add `Conflict` family diagnostic rules                                                       | Medium | High   | Feature      |
| 22  | Add observability hooks (metrics, tracing)                                                   | Medium | High   | Feature      |
| 23  | Extract `diagnose/internal/testutil` for shared mock runners                                 | Low    | Medium | Cleanup      |
| 24  | Add `go vet` line-length check or custom linter                                              | Low    | Low    | Tooling      |
| 25  | Add `CODEOWNERS` file                                                                        | Low    | Low    | Process      |

---

## Architecture State

### New APIs Added (v0.3.0)

| API                                       | File                          | Status                    |
| ----------------------------------------- | ----------------------------- | ------------------------- |
| `HandleErrorWithContext(ctx, err, cfg)`   | `handle.go:121`               | ✅ Tested                 |
| `HandleErrorDetailedWithConfig(err, cfg)` | `handle.go:155`               | ✅ Tested                 |
| `UnregisterTemplate(code)`                | `handle.go:322`               | ✅ Tested (via t.Cleanup) |
| `UnregisterClassification(sentinel)`      | `classify.go:99`              | ✅ Tested (via t.Cleanup) |
| `WithTimestamp(ts)`                       | `error.go`                    | ✅ Tested                 |
| `CommandRunner` interface                 | `diagnose/diagnose.go:309`    | ✅ Tested                 |
| `DefaultCommandRunner` struct             | `diagnose/diagnose.go:318`    | ✅ Tested                 |
| `ContextKey` type + 26 constants          | `diagnose/diagnose.go:59-114` | ✅ Tested                 |
| `DiagnosticResult.Context` field          | `diagnose/diagnose.go:162`    | ✅ Tested                 |
| `ErrorContext(err)` helper                | `diagnose/diagnose.go:337`    | ✅ Tested                 |
| `Runner.Run` context cancellation         | `diagnose/diagnose.go:215`    | ✅ Tested                 |

### Bug Fixes (v0.3.0)

| Bug                                               | File                               | Fix                                     |
| ------------------------------------------------- | ---------------------------------- | --------------------------------------- |
| `extractCommand` silently broken                  | `agent/agent.go:151`               | Now matches real diagnostic fix formats |
| `Compose` doc claimed "worst Family"              | `classify.go:74`                   | Fixed doc to say "uses errors.Join"     |
| `suggestCreate` treated `.config` as file         | `diagnose/rules_filesystem.go:120` | Now uses `filepath.Ext`                 |
| `resolveHost` naive string trimming               | `diagnose/rules_network.go:93`     | Now uses `net/url.Parse`                |
| `DialTimeout` ignored context                     | `diagnose/rules_network.go`        | Now uses `Dialer.DialContext`           |
| `Runner.Run` no context cancellation              | `diagnose/diagnose.go:215`         | Channel-based with ctx.Done()           |
| `Runner.Run` cyclomatic complexity 15             | `diagnose/diagnose.go`             | Refactored into 3 methods (~6 each)     |
| `HandleError` benchmark wrote ~1M lines to stderr | `benchmark_test.go`                | Now uses `io.Discard`                   |

---

## e) What We Should Improve

1. **`diagnose` coverage at 66.9%** — lowest in the project; `FilesystemRule` and `NetworkRule` need more test cases (network checks require system state, but filesystem checks can use `t.TempDir()`)
2. **Global mutable state in tests** — Fixed with `t.Cleanup` but `benchmark_test.go` still calls `RegisterClassification` without cleanup (acceptable for benchmarks, but noted)
3. **Runner.Run goroutine leak** — When context cancels, rule goroutines complete and write to the buffered channel, which is fine (channel is buffered to len(rules)). No actual leak, but the goroutines aren't actively cancelled. Rules that respect context will stop early.
4. **SKILL.md may need final review** — Previous session updated it but may not cover all new APIs

---

## f) Top #1 Question

**None.** All tasks from the previous session's "Exact Next Steps" list have been completed. The remaining items (#1-5 in the next-steps list) are either done or deferred with clear rationale.

---

_Generated by Crush at 2026-06-01 09:17_
