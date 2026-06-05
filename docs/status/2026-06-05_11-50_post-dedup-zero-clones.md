# Status Report — 2026-06-05 11:50

**Session focus:** Deduplication of code clones across the entire codebase

---

## a) FULLY DONE ✅

### Deduplication — ZERO clones at threshold 30

Started from 3 clone groups (43 total clones) at the original threshold 15 report. Eliminated ALL meaningful duplication:

| Threshold | Clone Groups Before | Clone Groups After |
|-----------|-------------------|-------------------|
| 50        | 1                 | **0**             |
| 30        | 3                 | **0**             |
| 15        | 3 (from report)   | 45 (incidental)   |

The 45 remaining groups at threshold 15 are all 2-3 item groups of 1-3 line snippets — idiomatic Go patterns (table-driven test struct shapes, `errors.Is` checks, `t.Fatalf` patterns). Per SKILL.md guidance: "Accept when idiomatic."

### Helpers extracted

| Package | Helper | Purpose |
|---------|--------|---------|
| `diagnose` | `assertDetail(t, result, key, want)` | Assert `result.Details[key] == want` |
| `diagnose/git` | `assertDetail` + `assertStatus` | Detail and status assertions for git tests |
| `diagnose/postgres` | `pgAssertDetail` + `pgAssertStatus` | Detail and status assertions for postgres tests |
| `diagnose` | `setAccessFailure(result, key, summary, fix)` | Extracted from `checkDirWritable`/`checkFileReadable` |

### Tests merged

| File | Before | After |
|------|--------|-------|
| `bridge/wrap_test.go` | 4 separate `errors.Is` tests | 1 table-driven `TestWrap_UnwrapChain` |
| `bridge/wrap_test.go` | 5 subtests in `TestWrap_Format` | 1 table-driven test |
| `family_test.go` | `TestFamilyString` standalone + 3 `testFamilyProperty` calls | All use `testFamilyProperty` with named `familyStringCase` type alias |

### Quality metrics

| Metric | Value |
|--------|-------|
| All tests (root + bridge + submodules) | **PASS** with `-race` |
| Lint issues (changed files) | **0** |
| Clone groups at threshold 30 | **0** |
| Clone groups at threshold 50 | **0** |
| Production files over 350 lines | **0** (max: `handle.go` at 342) |
| Test files over 350 lines | **1** (`diagnose/postgres/rules_postgres_test.go` at 337) |

### Test coverage

| Package | Coverage |
|---------|----------|
| Root (`errorfamily`) | 96.0% |
| `agent` | 89.4% |
| `diagnose` | 59.8% |

---

## b) PARTIALLY DONE ⚠️

### `diagnose` package coverage (59.8%)

The diagnose package has low coverage because its core rules (`FilesystemRule`, `NetworkRule`) shell out to real system commands. The git and postgres submodules have high coverage via mock injection (98.5% and 81.0%), but the base `diagnose` package still has integration-dependent code paths that are hard to unit test.

### File-split compliance

All files are under 350 lines. The closest to the limit:
- `handle.go`: 342 lines (8 lines under)
- `diagnose/postgres/rules_postgres_test.go`: 337 lines

---

## c) NOT STARTED 🔲

1. **TODO_LIST.md** — No project-level TODO list exists
2. **FEATURES.md** — No feature inventory exists
3. **ROADMAP.md** — No roadmap exists
4. **diagnose coverage improvement** — Could add more mock-based tests for `FilesystemRule` and `NetworkRule`
5. **agent coverage improvement** — 89.4%, could target 95%+
6. **handle.go proactive split** — At 342 lines, it's 8 lines under the 350 limit. Could split now before it grows
7. **Benchmark coverage** — `benchmark_test.go` and `diagnose/benchmark_test.go` have benchmarks but no coverage tracking
8. **Fuzz test expansion** — Only `bridge/fuzz_test.go` has fuzz tests; root package fuzz tests not verified for completeness
9. **Examples coverage** — All 3 example programs report 0% coverage (expected — they're `main` packages)
10. **`.golangci.yml` indentation reformat** — Uncommitted whitespace change exists

---

## d) TOTALLY FUCKED UP 💥

### Nothing catastrophic this session.

**Minor issue:** During dedup work, the git test files were being simultaneously split into `mock_test.go`/`scenario_test.go`/`integration_test.go` by a parallel process (commit `4219fdd`). My edits to the old `rules_git_test.go` effectively deleted it. I had to re-discover the new file structure and re-apply helpers to the split files. No data loss — all changes were preserved.

---

## e) IMPROVEMENTS WE SHOULD MAKE 📈

### High impact

1. **Create `TODO_LIST.md`** — No structured task tracking exists. Every session starts from AGENTS.md memory, which is fine but not a substitute for a proper TODO list.
2. **Create `FEATURES.md`** — AGENTS.md lists APIs but doesn't track feature status (DONE/PARTIAL/PLANNED).
3. **Split `handle.go` (342 lines)** — Proactively split before it hits the 350-line limit. Natural boundaries: `HandleError`, `HandleErrorWithConfig`, `HandleErrorDetailedWithConfig`, template logic.
4. **Improve `diagnose` coverage (59.8% → 80%+)** — Extract testable logic from `FilesystemRule.Run` and `NetworkRule.Run` into pure helper functions, then unit test the helpers.
5. **Postgres test file at 337 lines** — Close to 350-line limit. Consider splitting `rules_postgres_test.go` into `mock_test.go` + `unit_test.go` + `integration_test.go` (following the git pattern).

### Medium impact

6. **Add `diagnose/postgres` mock helpers** — The postgres tests already have `pgAssertDetail`/`pgAssertStatus`, but mock setup is still manual. Consider a `newDefaultPgMock()` that pre-configures `pg_isready=true`, `brew=false`, `systemctl=false`, `service=false`.
7. **Consistent test helper naming** — Git has `assertDetail`/`assertStatus`, postgres has `pgAssertDetail`/`pgAssertStatus`, diagnose has `assertDetail`. Consider a `testhelpers` package or consistent prefixing.
8. **Benchmark coverage tracking** — Add `-bench` flags to CI to track performance regressions.
9. **Fuzz corpus seeding** — Add seed corpora for the 5 fuzz functions in `bridge/fuzz_test.go`.

### Low impact

10. **Auto-format `.golangci.yml`** — The uncommitted indentation change is cosmetic but should be committed for consistency.
11. **Remove `var _ = fmt.Sprintf` in git mock_test.go** — Blank identifier assignment to prevent unused import; consider if still needed.
12. **Consolidate `diagnose/diagnose.go` + `diagnose/helpers.go` split** — The split was done for file size but `helpers.go` at 108 lines could arguably stay merged.

---

## f) Top 25 Things We Should Get Done Next 🎯

### Priority 1 — High Impact (Do First)

1. **Create `TODO_LIST.md`** — Centralized task tracking
2. **Create `FEATURES.md`** — Honest feature inventory with status
3. **Split `handle.go` (342 → ~170+172)** — Proactive before it grows past 350
4. **Improve `diagnose` coverage from 59.8% to 70%+** — Extract testable helpers from rules
5. **Split `diagnose/postgres/rules_postgres_test.go` (337 lines)** — Before it exceeds limit
6. **Commit `.golangci.yml` formatting fix** — Clean up uncommitted change

### Priority 2 — Medium Impact

7. **Create `ROADMAP.md`** — Long-term direction for v0.4.0+
8. **Add `newDefaultPgMock()` helper** — Reduce postgres test boilerplate
9. **Unify assert helper naming** — Consistent across packages
10. **Add `diagnose/NetworkRule` unit tests** — Improve coverage via mock injection
11. **Add `diagnose/FilesystemRule` unit tests** — Improve coverage via mock injection
12. **Update `AGENTS.md`** — Reflect latest file split structure, coverage numbers, helper functions
13. **Run `go mod tidy` on all modules** — Ensure clean dependency state
14. **Add integration test CI config** — Document how to run full suite
15. **Seed fuzz corpora** — Add initial seed inputs for fuzz functions

### Priority 3 — Nice to Have

16. **Add `docs/DOMAIN_LANGUAGE.md` update** — Capture any new terms from recent work
17. **Review `bridge/classify.go` (90 lines)** — Check if bridge classify logic can be simplified
18. **Add `diagnose` benchmark tests** — Track performance of rule execution
19. **Consider `testify` or custom assertion package** — For cross-package assert helper reuse
20. **Add `Makefile` or `justfile` → `flake.nix` migration** — If any build scripts exist
21. **Review `handle.go` for template extraction** — 342 lines suggests possible template helper extraction
22. **Add error context propagation tests** — Verify context flows through all HandleError variants
23. **Document the `familyStringCase` type alias pattern** — Useful for other table-driven tests
24. **Consider `diagnose/mock.go` → `diagnose/internal/mocks/`** — Keep test infrastructure separate
25. **Add `CHANGELOG.md`** — Track releases since v0.3.0

---

## g) Top #1 Question I Cannot Figure Out 🔥

**Should `diagnose` coverage (59.8%) be improved by mocking `os.Stat`/`os.Open`/`os.Create` in `FilesystemRule`, or by extracting pure logic functions that can be tested without touching the filesystem?**

The current design intentionally shells out to real filesystem operations. Mocking `os.Stat` requires either an interface wrapper or extracting the decision logic. The git/postgres packages solved this via `CommandRunner` injection, but `FilesystemRule`/`NetworkRule` use stdlib directly. The architectural decision matters: if we add a `Filesystem` interface to `diagnose`, it affects the public API. If we extract pure logic, it's internal but may feel like testing implementation details.

---

## Project State Summary

| Metric | Value |
|--------|-------|
| Version | v0.3.0 |
| Go version | 1.26.3 |
| Branch | master |
| Last commit | `d71792c` — docs(status): file-split compliance report |
| Production files | 19 |
| Test files | 19 |
| Production LOC | 2,888 |
| Test LOC | 3,613 |
| Clone groups (t≥30) | **0** |
| Clone groups (t≥50) | **0** |
| Lint issues | **0** (1 pre-existing gci in unmodified `diagnose.go`) |
| Test result | **ALL PASS** with `-race` |
| Race conditions | **0** |
