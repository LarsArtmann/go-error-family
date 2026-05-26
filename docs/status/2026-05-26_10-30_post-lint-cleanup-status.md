# Status Report — go-error-family

**Date:** 2026-05-26 10:30
**Version:** v0.1.2 (latest tag)
**Commits:** d8deac7 (HEAD, master)
**Unpushed:** 0

---

## Executive Summary

Structured error protocol library. Zero external deps. Go 1.26.3. **1,961 LOC production, 1,595 LOC tests.** 102 test functions. All passing. **Root 97.1%, Agent 100%, Diagnose 58.0% (↓ from 60.6%).**

Since the last status report (2026-05-21), the project:

1. Fixed SKILL.md internal contradiction (severity vs exit codes)
2. Tagged and released v0.1.2
3. Resolved all 83 golangci-lint issues
4. Refactored diagnostic rules for maintainability

The library remains **functionally complete** for v1.0. Remaining work is polish, test coverage for integration-test territory, and documentation.

---

## Fully Done ✓

| Item                          | Details                                                                                                                     |
| ----------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| **SKILL.md accuracy**         | Fixed severity/exit-code contradiction in Partial Success recipe, added ParseFamily to API reference, tightened rules table |
| **v0.1.2 release**            | Tagged and pushed                                                                                                           |
| **golangci-lint zero issues** | 83 issues resolved: errname, nilerr, unparam, funlen (2×), nestif (2×), gosec (3×), goconst (26×)                           |
| **Diagnostic rule refactor**  | FilesystemRule extracted into 4 helpers (handleStatError, suggestCreate, checkDirWritable, checkFileReadable)               |
| **Diagnostic rule refactor**  | GitRule extracted into 2 helpers (checkWorkingTree, checkRemote)                                                            |
| **runCommand cleanup**        | Removed unused stderr return, streamlined to 3 return values                                                                |
| **Linter config**             | `.golangci.yml` with 50+ enabled linters, v2 format                                                                         |
| **AGENTS.md maintained**      | Split-brain fix for ErrorBatch documented                                                                                   |
| **All tests pass**            | `go test ./...` — root, agent, diagnose all green                                                                           |

---

## Partially Done ⚠️

| Item                      | Current State     | Gap                                                                                                                     |
| ------------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------------- |
| **Diagnose coverage**     | 58.0%             | ↓ from 60.6%. Refactored code has more lines but same test surface. Shell-out rules remain integration-test territory.  |
| **DOMAIN_LANGUAGE.md**    | Template exists   | Still placeholder — no real terms filled in                                                                             |
| **CHANGELOG.md**          | Has v0.1.0–v0.1.2 | Missing entries for the lint-fix commit                                                                                 |
| **goconst fixes**         | Linter is happy   | Some "constants" (strTrue, strFalse, strHost) add indirection without real value. Readable code > linter-silenced code. |
| **AGENTS.md linter docs** | Not updated       | No mention of disabled linters (exhaustruct, gochecknoglobals) or why                                                   |

---

## Not Started ✗

| Item                                 | Impact | Notes                                                                                                               |
| ------------------------------------ | ------ | ------------------------------------------------------------------------------------------------------------------- |
| CI/CD pipeline                       | High   | release.yml exists but no test-on-push, coverage gate, or lint gate. Pre-commit hook (BuildFlow) runs locally only. |
| Real consumers / examples            | High   | Zero known consumers. Library is polished but untested in real projects.                                            |
| Integration tests for diagnose rules | High   | PostgresRule, GitRule, FilesystemRule, NetworkRule all shell out. No mock-based coverage.                           |
| `errors.Join` support                | Medium | Go 1.20+ multi-errors not handled by Classify. Would check Unwrap() []error.                                        |
| pkg.go.dev validation                | Medium | v0.1.2 may still have validation issues (external bug).                                                             |
| Example functions for godoc          | Medium | No `Example_*` test functions for key APIs.                                                                         |
| Fuzz tests                           | Medium | ParseFamily, applyContext, resolveHost should be fuzzed — they accept external input.                               |
| Benchmark suite                      | Low    | No performance baselines for Classify, HandleError, Runner.Run.                                                     |

---

## Totally Fucked Up! 🔥

| Issue                                     | Severity   | Explanation                                                                                                                                                                  |
| ----------------------------------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **//nolint:nilerr on entire GitRule.Run** | Medium     | Suppresses ALL nilerr checks in the function, not just the one legitimate diagnostic-rule pattern. A targeted fix or code restructure would be better.                       |
| **goconst "cure" in diagnose/**           | Low-Medium | Replacing `"true"`/`"false"` in map keys with `strTrue`/`strFalse` constants adds cognitive overhead. The strings are self-documenting. This is a linter tax on readability. |
| **Disabled exhaustruct globally**         | Low        | 44 false positives made it untenable. But disabling globally means future code won't be checked. Targeted exclusions would be more precise.                                  |
| **Disabled gochecknoglobals globally**    | Low        | 3 intentional data tables (familyData, defaultMessages, postgresSpec). Global disable is a sledgehammer.                                                                     |
| **Diagnose coverage dropped 2.6%**        | Low        | From 60.6% to 58.0%. Not a bug — refactoring added lines without adding tests. But the trend is wrong.                                                                       |

---

## What We Should Improve 🎯

### 1. Revert low-value goconst "fixes"

The `strTrue`, `strFalse`, `strHost`, `strPort`, `strLocalhost`, `strUnknown` constants in `diagnose/` and the error-code constants in `handle.go` don't improve the code. They make it harder to read. Better approach: configure `goconst.min-len` in `.golangci.yml` to exclude short strings, and use `//nolint:goconst` on ruleSpec declarations.

### 2. Fix over-broad //nolint:nilerr

Instead of `//nolint:nilerr` on the entire `GitRule.Run` function, either:

- Restructure the `os.Stat` check to avoid the pattern that triggers nilerr
- Use a more targeted suppression mechanism

### 3. Document linter decisions in AGENTS.md

Add a section explaining why `exhaustruct` and `gochecknoglobals` are disabled, and what the goconst policy is. Future agents (and humans) need to know this.

### 4. Add integration test strategy for diagnose rules

The 4 diagnostic rules (Postgres, Filesystem, Network, Git) shell out to system commands. They're inherently integration-test territory. Consider:

- A `//go:build integration` tag for real system tests
- Mock-based unit tests for the `Applicable()` and result-assembly logic
- A `DIAGNOSE_TEST_MODE` env var that uses mock commands

### 5. CI pipeline

The `.github/workflows/release.yml` only triggers on tags. Add a `ci.yml` that runs on every push:

- `go test ./...`
- `go vet ./...`
- `golangci-lint run`
- Coverage report with threshold (e.g., 95% root, 100% agent, 55% diagnose)

---

## Top 25 Priorities (Sorted by Impact × Ease)

### Tier 1: Quick Wins (minutes each)

1. **Revert strTrue/strFalse in diagnose/** — Replace constants with literals. Add `//nolint:goconst` on spec vars.
2. **Revert strHost/strPort/strLocalhost/strUnknown in diagnose/** — Same rationale.
3. **Revert codeFileNotFound etc. in handle.go** — Error code constants add indirection without preventing real bugs.
4. **Configure goconst.min-len in .golangci.yml** — Set to 4 or 5 to avoid flagging "dir", "git", "true", "false".
5. **Fix //nolint:nilerr scope** — Make it targeted, not function-wide.
6. **Document linter decisions in AGENTS.md** — Add "Linter Configuration" section.
7. **Update CHANGELOG.md** — Add v0.1.2 entries.
8. **Fill DOMAIN_LANGUAGE.md** — Define: Family, Code, Context, Classification, Template, DiagnosticRule.

### Tier 2: Medium Impact (1–2 hours each)

9. **Add CI pipeline** — test-on-push + `go vet` + `golangci-lint` + coverage gate.
10. **Add mock-based unit tests for diagnose rules** — Test Applicable(), result assembly, edge cases without shelling out.
11. **Add `errors.Join` handling in Classify** — Check `Unwrap() []error` interface.
12. **Add Example\_\* test functions** — godoc-rendered examples for New, Wrap, Classify, HandleError.
13. **Extract confidence constants** — Name the magic numbers (0.1, 0.3, 0.4, 0.7, 0.8, 0.85, 0.9).
14. **Add `//go:build integration` tests for shell-out rules** — Real system tests for PostgresRule, GitRule, etc.

### Tier 3: Strategic (half-day+)

15. **Consolidate DiagnosticFinding → DiagnosticResult** — Eliminate type duplication. Make DiagnosticFunc return []DiagnosticResult or define a shared struct.
16. **Real-world example project** — Small CLI tool using the library end-to-end.
17. **Fuzz tests for ParseFamily, applyContext** — Input is external, should be fuzzed.
18. **Benchmark suite** — Classify, HandleError, Runner.Run baselines.
19. **API stability review** — Audit all exported types for breaking change potential before v1.0.0.
20. **Add HandleBatchError recipe to README** — Make partial success pattern discoverable.
21. **Consider Status as string enum** — Eliminates default branches, trivial serialization.

### Tier 4: Long-term / Nice-to-have

22. **OpenAPI/JSON schema for HandleResult** — For HTTP/gRPC consumers.
23. **Structured logging bridge** — `slog` integration for error context.
24. **Prometheus metrics bridge** — Error family/code counters.
25. **`go generate` for message templates** — Extract templates from YAML/JSON.

---

## Metrics

| Metric                        | Value                           | Δ from 2026-05-21              |
| ----------------------------- | ------------------------------- | ------------------------------ |
| Production LOC                | 1,961                           | +170 (refactoring + constants) |
| Test LOC                      | 1,595                           | +14                            |
| Test functions                | 102                             | —                              |
| Total checks (incl. subtests) | ~270                            | —                              |
| Packages                      | 3 (root, diagnose, agent)       | —                              |
| Production files              | 13                              | —                              |
| Test files                    | 4                               | —                              |
| Largest production file       | handle.go (~290 lines)          | +28 lines                      |
| Largest test file             | errorfamily_test.go (655 lines) | —                              |
| External dependencies         | 0                               | —                              |
| Go version                    | 1.26.3                          | +0.0.1                         |
| golangci-lint issues          | **0**                           | **↓83**                        |
| Root coverage                 | 97.1%                           | —                              |
| Agent coverage                | 100%                            | —                              |
| Diagnose coverage             | **58.0%**                       | **↓2.6%**                      |

---

## File Inventory

```
errorfamily/
  error.go           160 lines   Error struct, fmt.Formatter
  family.go          200 lines   Family enum, Audience, Tone, string constants
  interfaces.go       41 lines   Coded, Classified, Contextual, Retryable
  constructors.go     99 lines   New/Wrap + 10 family shortcuts
  classify.go        101 lines   Classify, RegisterClassification
  handle.go          290 lines   HandleError, templates, MessageTemplate, error-code constants

diagnose/
  diagnose.go        300 lines   Runner, DiagnosticRule, ruleSpec, helpers, string constants
  context.go          40 lines   runCommand (3-return), commandExists
  rules_postgres.go  135 lines   PostgresRule, IsPostgresRunning
  rules_filesystem.go 155 lines  FilesystemRule (4 helpers)
  rules_network.go    96 lines   NetworkRule
  rules_git.go       130 lines   GitRule (2 helpers)

agent/
  agent.go           144 lines   DebugAgent, Config, AgentResult, FixStep
```

---

## Unanswered Questions

**Top question:** How should the library test its shell-out diagnostic rules (Postgres, Git, Filesystem, Network) without either (a) making tests fragile to system state, or (b) adding so much mock infrastructure that the tests become worthless? The `runCommand` function is unexported, so external mocking isn't possible. Options:

1. Export `runCommand` and accept the API surface expansion?
2. Add a test-only `testCommand` field to each rule struct?
3. Use build tags (`//go:build integration`) and accept that diagnose coverage will always be low in unit tests?
4. Extract the command execution into an interface that rules depend on?

The current approach (option 3, implicit) leaves diagnose at 58% coverage forever. Is that acceptable for a library, or should we invest in option 2 or 4?
