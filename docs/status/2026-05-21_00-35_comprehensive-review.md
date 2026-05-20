# Status Report — go-error-family

**Date:** 2026-05-21 00:35
**Version:** v0.1.1 (latest tag)
**Commits:** 4e982c9 (HEAD, master)
**Unpushed:** 0

---

## Executive Summary

Structured error protocol library. Zero external deps. Go 1.26+. **1,791 LOC production, 1,581 LOC tests.** 102 test functions, 270 total checks (including subtests). All passing. **Root 97.1%, Agent 100%, Diagnose 60.6%.** Open-sourced under MIT.

The library is **functionally complete** for v1.0. The remaining work is polish, documentation accuracy, and CI.

---

## Fully Done ✓

| Item                       | Details                                                                                                       |
| -------------------------- | ------------------------------------------------------------------------------------------------------------- |
| **Family enum**            | 5 families (Rejection/Conflict/Transient/Corruption/Infrastructure) with exit codes, tone, audience, defaults |
| **Error struct**           | Reference implementation with code, message, family, context, cause, timestamp                                |
| **Consumer interfaces**    | Coded, Classified, Contextual, Retryable — all embed `error` for `errors.AsType[T]()`                         |
| **Constructors**           | New/Wrap/Newf/Wrapf + family shortcuts (10 constructors)                                                      |
| **Classification**         | 4-level precedence: Classified → Retryable → Registered sentinels → Transient default                         |
| **CLI boundary**           | HandleError, HandleErrorWithConfig, HandleErrorDetailed — Wix-style messages                                  |
| **Template system**        | defaultMessages map + RegisterTemplate() + HandleConfig.TemplateOverride                                      |
| **Diagnostic framework**   | Runner, DiagnosticRule interface, ruleSpec matching, concurrent execution                                     |
| **4 diagnostic rules**     | Postgres, Filesystem, Network, Git — all data-driven via ruleSpec                                             |
| **Agent**                  | DebugAgent interface, deterministic analyzer, timeout enforcement                                             |
| **Root coverage**          | 97.1% (every function covered except minor branches)                                                          |
| **Agent coverage**         | 100%                                                                                                          |
| **Thread safety**          | sync.RWMutex for classification and template registries, snapshots for reads                                  |
| **Open source**            | MIT license, tagged v0.1.1, pushed to origin                                                                  |
| **Architecture**           | Clean dependency graph: root → stdlib, diagnose → root, agent → root+diagnose                                 |
| **File sizes**             | All production files under 282 lines (handle.go is largest)                                                   |
| **Partial success recipe** | Documented in SKILL.md — composition pattern, not library type                                                |

---

## Partially Done ⚠️

| Item                   | Current State         | Gap                                                                                      |
| ---------------------- | --------------------- | ---------------------------------------------------------------------------------------- |
| **Diagnose coverage**  | 60.6%                 | Rules that shell out (git, pg_isready) are hard to unit test; integration-test territory |
| **DOMAIN_LANGUAGE.md** | Template exists       | Not filled in for this project — placeholder glossary, no real terms                     |
| **CHANGELOG.md**       | Has v0.1.0 and v0.1.1 | Missing entries for recent doc/cleanup commits                                           |

---

## Not Started ✗

| Item                           | Impact | Notes                                                                 |
| ------------------------------ | ------ | --------------------------------------------------------------------- |
| CI/CD pipeline                 | High   | release.yml exists but no test-on-push, coverage report, or lint gate |
| Real consumers / examples      | High   | Zero known consumers — library is untested in real projects           |
| `go vet` / `staticcheck` in CI | Medium | Both pass locally but not enforced                                    |
| pkg.go.dev documentation       | Medium | v0.1.1 has a validation error (external bug)                          |
| Versioned Go docs              | Low    | Go doc comments are good but not reviewed for godoc rendering         |

---

## Split Brain Found & Fixed ✓

| Split Brain                                                                                                                                                                                                                  | Status             |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| **AGENTS.md referenced `ErrorBatch`/`BatchResult[T]`** which were deleted in commit 4e982c9. Caused by `git checkout -- AGENTS.md` which restored the stale batch docs. **Fixed:** replaced with composition recipe pointer. | FIXED this session |

---

## Architecture Review

### What's Excellent

1. **Data-driven patterns** — `familyData` array, `defaultMessages` map, `ruleSpec` structs. Adding a new Family = one const + one array entry. Adding a diagnostic rule = one struct + one `ruleSpec`. Zero boilerplate.

2. **Interface composition** — Each consumer interface is tiny (1 method + `error` embedding). Types implement whichever subset makes sense. `errors.AsType[T]()` for clean extraction.

3. **Dependency graph is clean** — Root package depends on nothing. `DiagnosticFunc` is a function type (not interface) to avoid circular imports. Consumer wires it.

4. **Consistent conventions** — Context values always `string`, error codes dot-notation, timestamps UTC, nil-safe constructors.

### What Could Be Better

1. **`DiagnosticResult` vs `DiagnosticFinding` split brain** — `diagnose.DiagnosticResult` (in diagnose/) and `errorfamily.DiagnosticFinding` (in handle.go) represent the same concept with different fields. `DiagnosticFinding` is a trimmed copy (missing Details, Confidence, Duration). This is a **real type duplication** that should be consolidated. The `DiagnosticFunc` bridge type means the consumer must manually convert between them.

2. **`parentDir()` in rules_filesystem.go** reimplements `filepath.Dir()`. Should use stdlib.

3. **`stripAfter()` in rules_network.go** — only used once, trivial inline replacement.

4. **`runCommand()` signature** — returns 4 values (stdout, stderr, exitCode, err). The `err` is only non-nil for context cancellation or command-not-found (exit errors are converted to exitCode). This is correct but the contract is non-obvious — callers always ignore err with `_`.

5. **No `errors.Join` support** — Go 1.20+ has `errors.Join` for multi-errors. The library's `Classify` doesn't handle wrapped multi-errors. If someone passes an `errors.Join` result, `Classify` will check each interface against the join wrapper, not individual errors.

6. **Confidence values are magic numbers** — 0.1, 0.3, 0.4, 0.7, 0.8, 0.85, 0.9 scattered across rules. Should be constants with names.

7. **`Status` is an int but could use `string`** — The `String()` method is the real value used everywhere. Making it a string enum would eliminate the `default: "unknown"` branch and make serialization trivial.

8. **`Family.IsValid()` uses a range check** — Works because families are sequential `iota`. Fragile if someone adds a non-sequential family. Consider a set lookup.

---

## Top 25 Priorities (Sorted by Impact × Ease)

### Tier 1: Quick Wins (minutes each)

1. **Fix `parentDir()` → `filepath.Dir()`** — 1 line, eliminates reimplementation
2. **Inline `stripAfter()`** — trivial, removes unnecessary helper
3. **Extract confidence constants** — name the magic numbers (e.g., `confidenceNotRootCause = 0.3`)
4. **Fill DOMAIN_LANGUAGE.md** — define actual terms: Family, Code, Context, Classification, etc.
5. **Update CHANGELOG.md** — add recent doc/cleanup commits
6. **Add `//go:build` tags to diagnose rules** — make integration vs unit test distinction explicit

### Tier 2: Medium Impact (1-2 hours each)

7. **Consolidate `DiagnosticFinding` → use `DiagnosticResult`** — eliminate the type duplication. Make `DiagnosticFunc` return `[]DiagnosticResult` or define a shared struct.
8. **Add basic CI pipeline** — test-on-push + `go vet` + coverage gate
9. **Add `errors.Join` handling in `Classify`** — check if error implements `Unwrap() []error` and classify the first applicable error
10. **Add `Example_*` test functions** — godoc-rendered examples for key APIs
11. **Add integration test stubs for diagnose rules** — at least test the match/Applicable paths with mock commands
12. **Review all doc comments for godoc rendering** — ensure first sentence is summary, proper formatting

### Tier 3: Strategic (half-day+)

13. **Real-world example project** — a small CLI tool using the library end-to-end
14. **Benchmark suite** — `Classify`, `HandleError`, `Runner.Run` performance baselines
15. **Fuzz tests for `ParseFamily`, `applyContext`** — input is external, should be fuzzed
16. **Consider `Status` as string enum** — eliminates `default` branches, trivial serialization
17. **Add `Family.IsValid()` set-based check** — more robust than range check
18. **Versioned module path** — decide if v1.0.0 is ready, tag it
19. **API stability review** — audit all exported types for breaking change potential
20. **Add `Errors(ctx, err) []error` helper** — unwrap `errors.Join` into flat list for batch scenarios
21. **Add `HandleBatchError` recipe to README** — make the partial success pattern discoverable

### Tier 4: Long-term / Nice-to-have

22. **OpenAPI/JSON schema for `HandleResult`** — for HTTP/gRPC consumers
23. **Structured logging bridge** — `slog` integration for error context
24. **Prometheus metrics bridge** — error family/code counters
25. **`go generate` for message templates** — extract templates from YAML/JSON

---

## Metrics

| Metric                        | Value                           |
| ----------------------------- | ------------------------------- |
| Production LOC                | 1,791                           |
| Test LOC                      | 1,581                           |
| Test functions                | 102                             |
| Total checks (incl. subtests) | 270                             |
| Packages                      | 3 (root, diagnose, agent)       |
| Production files              | 13                              |
| Test files                    | 4                               |
| Largest production file       | handle.go (262 lines)           |
| Largest test file             | errorfamily_test.go (655 lines) |
| External dependencies         | 0                               |
| Go version                    | 1.26.2                          |

---

## File Inventory

```
errorfamily/
  error.go          160 lines   Error struct, fmt.Formatter
  family.go         171 lines   Family enum, Audience, Tone
  interfaces.go      41 lines   Coded, Classified, Contextual, Retryable
  constructors.go    99 lines   New/Wrap + 10 family shortcuts
  classify.go       101 lines   Classify, RegisterClassification
  handle.go         262 lines   HandleError, templates, MessageTemplate

diagnose/
  diagnose.go       282 lines   Runner, DiagnosticRule, ruleSpec, helpers
  context.go         40 lines   runCommand, commandExists
  rules_postgres.go 132 lines   PostgresRule, IsPostgresRunning
  rules_filesystem.go 143 lines  FilesystemRule
  rules_network.go   96 lines   NetworkRule
  rules_git.go      120 lines   GitRule

agent/
  agent.go          144 lines   DebugAgent, Config, AgentResult, FixStep
```

---

## Unanswered Questions

None — the codebase is small, well-understood, and fully reviewed.
