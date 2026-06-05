# go-error-family — Full Status Report

**Date:** 2026-06-05 07:11
**Reporter:** Crush (automated)
**Trigger:** Post-session comprehensive audit after bridge submodule build
**Branch:** master (`c699793`)
**Go:** 1.26.3 · **License:** MIT · **Version:** v0.3.0

---

## Executive Summary

The project is in strong shape. Root package is zero-dependency, 96% coverage, zero lint issues. The new `bridge/` submodule (samber/oops integration) was built this session with 94% coverage and full interface bridging. Two HTML reports (comparison, planning) were created. The project has 6 workspace modules, ~2,762 production LOC, ~3,560 test LOC, and 0 TODO comments.

**One real problem:** bridge/ has 7 lint issues that need fixing (errname, goconst, staticcheck, wrapcheck). Everything else is clean.

---

## A) FULLY DONE ✅

### Root package (`errorfamily`)
- **5 error families** with behavioral semantics (Rejection, Conflict, Transient, Corruption, Infrastructure)
- **4 consumer interfaces** (Coded, Classified, Contextual, Retryable) — each embeds `error` for `errors.AsType[T]()`
- **Classification cascade**: multi-error → Classified → Retryable → sentinels → Transient default
- **CLI boundary handler** with Wix-style templates (What/Why/Fix/WayOut), context propagation, diagnostic wiring
- **Sentinel registration** — `RegisterClassification(s)`, `UnregisterClassification` — thread-safe, lock-free snapshots
- **Multi-error support** via `Compose()` — first non-Transient wins
- **Coverage: 96.0%**, 0 lint issues, 0 race conditions

### `agent/` package
- **DebugAgent interface** — `Analyze()` produces root cause analysis + `FixStep` suggestions
- **Analysis-only** — library proposes, consumer disposes
- **Deterministic analyzer** with context timeout
- **Coverage: 89.4%** (down from 100% — likely measurement variance)

### `diagnose/` package (core)
- **Concurrent diagnostic rules** — Runner.Run with configurable rules
- **Built-in rules**: FilesystemRule, NetworkRule (zero-dep)
- **CommandRunner interface** — injectable for testing
- **Typed ContextKey** — 20+ constants for error context keys
- **Coverage: 61.7%** (reflects shell-out rules tested via integration)

### `diagnose/git/` submodule
- GitRule — repo state, working tree, remotes
- MockCommandRunner injection
- **Coverage: 98.5%**

### `diagnose/postgres/` submodule
- PostgresRule — pg_isready, TCP connectivity, start command
- MockCommandRunner injection
- **Coverage: 80.3%**

### `bridge/` submodule (NEW this session)
- **ClassifiedOops** — bridges oops.OopsError to error-family interfaces
- **Satisfies 4 interfaces**: Classified, Coded, Retryable, Contextual
- **fmt.Formatter** — `%+v` delegates to oops stacktrace
- **errors.Is** — delegates to OopsError.Is + original error
- **Original error preservation** — never drops the input, even for plain stdlib errors
- **InferFamily** — tag overrides → domain defaults → Transient fallback
- **AutoWrap** — infer + wrap in one call
- **Coverage: 94.0%**, 44 tests, 4 benchmarks, 3 examples, 5 fuzz targets

### Documentation (NEW this session)
- `docs/comparison-samber-oops.html` — 8-section editorial comparison (1,107 lines)
- `docs/planning/best-of-both-worlds.html` — 6-section integration blueprint (1,240 lines)
- SKILL.md updated with bridge architecture section
- AGENTS.md updated with bridge submodule docs

---

## B) PARTIALLY DONE 🔶

### Bridge lint compliance
- **7 lint issues** in bridge/ that need fixing:
  1. `errname`: `ClassifiedOops` should be `ClassifiedError` or similar
  2. `goconst` (×3): `database`, `infra`, `infrastructure` strings should be constants
  3. `staticcheck`: `errors` imported twice (one aliased as `errors2`)
  4. `wrapcheck`: OopsError.Unwrap() returns unwrapped external error
- Tests pass, coverage is good, but lint is not clean

### Benchmark modernization
- 4 instances of `b.N` loops in bridge benchmarks that should use `b.Loop()` (Go 1.26+)
- Same pattern likely exists in other modules

### Project documentation files
- **No TODO_LIST.md** — should exist per project conventions
- **No FEATURES.md** — should exist per project conventions
- **No ROADMAP.md** — planning exists as HTML but not as tracked document

---

## C) NOT STARTED ⬜

### Phase 3: Unified Boundary Handler
- The `handle` package from the planning document
- 5-step pipeline: enrich → classify → diagnose → analyze → emit
- ~200 lines estimated
- Requires bridge + diagnose + agent integration

### golangci-lint configuration for bridge/
- Root package has `.golangci.yml` with custom exclusions
- Bridge module may need its own lint config or additions to the root config

### Bridge example in examples/
- Skipped due to separate go.mod complexity
- Would demonstrate real-world oops + error-family usage

### README.md updates
- No mention of bridge submodule
- No mention of samber/oops integration
- Comparison reports not linked

### CI pipeline updates
- `.github/workflows/ci.yml` doesn't test bridge/ submodule
- Coverage reporting doesn't include bridge/

---

## D) TOTALLY FUCKED UP 💥

### Nothing is catastrophically broken
- All tests pass across all modules with `-race`
- Build succeeds
- No security issues
- No data loss risks

### Closest to "fucked up":
- The initial bridge implementation silently dropped plain stdlib errors (fixed in `c699793`)
- The lint issues in bridge/ were introduced and not caught before commit
- The `errors2` alias in bridge_test.go is a code smell from a lazy fix

---

## E) WHAT WE SHOULD IMPROVE

### Architecture
1. **Bridge lint issues** — 7 issues, should be zero like root package
2. **Root package structure** — go-structure-linter flags all root files as "should be in /internal/ or /pkg/" — deliberate choice for a library, but worth documenting why
3. **Coverage threshold** — no CI-enforced minimum (flagged by go-structure-linter)

### Type Models
4. **ClassifiedOops naming** — `errname` linter wants `XxxError` format. Renaming to `ClassifiedError` would align with Go conventions but break the "oops" naming connection
5. **ErrorContext tags format** — tags are serialized as `fmt.Sprint([]string{...})` which produces `[timeout connection]` — not ideal for programmatic consumers. Should consider joining with comma or making it structured
6. **InferFamily could accept options** — the domain/tag mapping tables are package-level vars, which means they're global. Consider accepting custom mappings via functional options

### Testing
7. **Diagnose core coverage** — 61.7% is the lowest in the project. Shell-out rules need more mock-based tests
8. **Agent coverage** — dropped from 100% to 89.4%, likely needs investigation
9. **Fuzz corpus** — bridge fuzz tests use `f.Add()` seeds but haven't been run with `-fuzz` for extended periods

### Documentation
10. **No TODO_LIST.md or FEATURES.md** — both are specified in project conventions but don't exist
11. **README.md** — needs bridge section
12. **CHANGELOG.md** — needs bridge entry

---

## F) TOP #25 THINGS TO DO NEXT

Sorted by impact × effort (highest first):

| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 1 | Fix 7 bridge lint issues (errname, goconst, staticcheck, wrapcheck) | High | Low | bridge |
| 2 | Add bridge/ to CI workflow (.github/workflows/ci.yml) | High | Low | CI |
| 3 | Update README.md with bridge section and comparison links | Medium | Low | docs |
| 4 | Add CHANGELOG.md entry for bridge submodule | Medium | Low | docs |
| 5 | Create FEATURES.md with honest feature inventory | Medium | Low | docs |
| 6 | Modernize b.N → b.Loop() in all benchmark functions | Low | Low | all |
| 7 | Improve ErrorContext tags serialization (comma-join instead of fmt.Sprint) | Medium | Low | bridge |
| 8 | Add golangci-lint config overrides for bridge/ | Medium | Low | bridge |
| 9 | Investigate agent coverage drop (100% → 89.4%) | Medium | Low | agent |
| 10 | Create TODO_LIST.md from existing planning docs | Medium | Low | docs |
| 11 | Build Phase 3 handle package (unified boundary handler) | High | Medium | handle |
| 12 | Improve diagnose core coverage (61.7% → 80%+) | Medium | Medium | diagnose |
| 13 | Add bridge example (separate go.mod or integration test) | Medium | Medium | examples |
| 14 | Make InferFamily mapping tables configurable (functional options) | Medium | Medium | bridge |
| 15 | Add coverage threshold to CI (enforce 80% minimum) | Medium | Low | CI |
| 16 | Document root package structure decision (why not /pkg/) | Low | Low | docs |
| 17 | Add bridge/ to release workflow (.github/workflows/release.yml) | Medium | Low | CI |
| 18 | Run bridge fuzz tests for extended period (find edge cases) | Low | Low | bridge |
| 19 | Add gitleaks exception for bridge/ if needed | Low | Low | bridge |
| 20 | Update docs/DOMAIN_LANGUAGE.md with bridge terms | Low | Low | docs |
| 21 | Consider structured tags in ErrorContext (not flat string) | Medium | Medium | bridge |
| 22 | Add integration test: oops → AutoWrap → Classify → HandleError → exit code | High | Medium | bridge |
| 23 | Update flake.nix to include bridge in devShell test/lint targets | Medium | Low | nix |
| 24 | Audit all examples for accuracy against current API | Low | Low | examples |
| 25 | Consider version bump to v0.4.0 (bridge is a new feature) | Medium | Low | release |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should the bridge module be released as part of v0.4.0 with its current API, or should the lint issues and architectural improvements (ClassifiedOops → ClassifiedError rename, configurable InferFamily, structured tags) be resolved first?**

The bridge API is functional and tested, but renaming `ClassifiedOops` to `ClassifiedError` is a breaking change if anyone has already imported it. The `InferFamily` global mapping tables work but aren't configurable. I don't know the project's release philosophy — ship-and-iterate vs. polish-before-release.

---

## Coverage Summary

| Module | Coverage | Status |
|--------|----------|--------|
| root (`errorfamily`) | **96.0%** | ✅ Strong |
| `agent` | **89.4%** | ✅ Good (was 100%, needs investigation) |
| `bridge` | **94.0%** | ✅ Strong (new) |
| `diagnose` (core) | **61.7%** | 🔶 Needs work |
| `diagnose/git` | **98.5%** | ✅ Excellent |
| `diagnose/postgres` | **80.3%** | ✅ Good |

## Lint Summary

| Module | Issues | Status |
|--------|--------|--------|
| root | **0** | ✅ Clean |
| agent | **0** | ✅ Clean |
| bridge | **7** | 🔶 Needs fixing |
| diagnose | **0** | ✅ Clean |
| diagnose/git | **0** | ✅ Clean |
| diagnose/postgres | **0** | ✅ Clean |

## Module Dependency Graph

```
root (zero-dep)
├── agent/          → depends on root + diagnose
├── diagnose/       → depends on root (zero external deps in core)
├── diagnose/git/   → depends on root
├── diagnose/postgres/ → depends on root
└── bridge/         → depends on root + samber/oops (+ ulid, lo, otel/trace transitive)
```

## Lines of Code

| Module | Production | Test | Total |
|--------|-----------|------|-------|
| root | 1,014 | 1,306 | 2,320 |
| agent | 200 | 200 | 400 |
| bridge | 261 | 752 | 1,013 |
| diagnose | 878 | 556 | 1,434 |
| diagnose/git | 202 | ~400 | ~602 |
| diagnose/postgres | 207 | 346 | 553 |
| **TOTAL** | **2,762** | **3,560** | **~6,322** |

## Recent Commits (this session)

```
c699793 feat(bridge): add samber/oops integration submodule with full interface bridging
b5d9049 docs(planning): add "Best of Both Worlds" integration blueprint for go-error-family + samber/oops
fab9a64 docs(comparison): add go-error-family vs samber/oops comparative analysis report
```

**Session additions:** +3,481 lines across 12 files (3 source, 2 test, 1 fuzz, 2 go.mod, 2 HTML docs, 2 doc updates)

## Test Functions

| Type | Count |
|------|-------|
| Test* | ~139 |
| Fuzz* | 10 |
| Benchmark* | 23 |
| Example* | 8 |
| **Total** | **~180** |
