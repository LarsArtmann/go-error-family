# Comprehensive Status Update — 2026-05-26 11:41

**Project:** `github.com/larsartmann/go-error-family`
**Branch:** `master`
**Commit:** `87e96c0`
**Status:** Modularization COMPLETE. Multi-module workspace operational. Zero lint issues.
**Author:** Crush (AI Assistant) with Lars Artmann

---

## A) FULLY DONE

### 1. Modularization of Diagnostic Rules

- **GitRule** extracted to `diagnose/git/` submodule with own `go.mod`
- **PostgresRule** extracted to `diagnose/postgres/` submodule with own `go.mod`
- **go.work** workspace file created linking root + 2 submodules
- All tests, imports, and references updated across all modules
- Tests moved from `diagnose/diagnose_test.go` to dedicated test files in each submodule

### 2. API Surface Expansion

The following previously-unexported helpers are now **public** (exported from `diagnose`):

- `RuleSpec` / `RuleSpec.Matches`
- `HasContextKey`, `ContextValue`, `ResolveContextKey`
- `HasContextSubstring`, `FamilyIs`, `ErrorCodeContains`
- `RunCommand`, `CommandExists`

This enables third-party rule authors to build on the framework without forking.

### 3. DefaultRunner() Breaking Change

- Now includes **only** zero-dependency rules: `FilesystemRule`, `NetworkRule`
- GitRule and PostgresRule removed from default set
- Consumers opt in via explicit import + `NewRunner(&git.GitRule{}, &postgres.PostgresRule{})`

### 4. Test & Lint Hygiene

- **All tests pass** in all 3 modules (root, diagnose/core, diagnose/git, diagnose/postgres)
- **0 golangci-lint issues** across all modules
- **0 go vet issues** across all modules
- Updated `.golangci.yml` exclusion rules for `diagnose/git` and `diagnose/postgres` paths

### 5. Documentation Updates

- `README.md`: architecture tree, built-in vs opt-in rules, import examples
- `SKILL.md`: updated diagnostic rules table, helper API, module paths
- `AGENTS.md`: updated coverage table, submodule guidance
- `CHANGELOG.md`: Unreleased section with breaking change notice
- `.gitignore`: override global `go.work` ignore for this repo

### 6. errors.Join Support (Previous Session)

- `Classify` now handles `Unwrap() []error` before `errors.AsType`
- First non-Transient wins (fail-closed)
- 11 test cases covering edge cases

### 7. Lint Cleanup (Previous Session)

- All 83 golangci-lint issues resolved
- Disabled `exhaustruct` and `gochecknoglobals` (false positives)
- Refactored `FilesystemRule.Run` into 4 helpers, `GitRule.Run` into 2 helpers

---

## B) PARTIALLY DONE

### 1. Test Coverage — Mixed Bag

| Module               | Coverage   | Status                                             |
| -------------------- | ---------- | -------------------------------------------------- |
| root (`errorfamily`) | **97.2%**  | Excellent                                          |
| `agent`              | **100.0%** | Perfect                                            |
| `diagnose` (core)    | **66.8%**  | Improved (was 60.6%)                               |
| `diagnose/git`       | **23.1%**  | Poor — integration-test territory, but gap is real |
| `diagnose/postgres`  | **44.8%**  | Below target — network mocks needed                |

**Gap analysis:**

- Git and Postgres rules shell out to system commands (`git`, `pg_isready`) and network calls. Pure unit testing is hard.
- But: `resolveHost`, `resolvePort`, `suggestStartFix`, `Applicable`, `IsPostgresRunning` (non-shell paths) ARE testable and NOT fully covered.

### 2. Nix Flake — Partially Broken

- `nix flake check` **fails** on `checks.build`: Go build cache permission denied (`/homeless-shelter/.cache/go-build`)
- `nix build` **fails**: no `defaultPackage` attribute defined
- Dev shell works fine
- Apps (`nix run .#test`, `nix run .#lint`) work

### 3. Documentation — 90% Complete

- Missing: migration guide for consumers upgrading from v0.1.x to v0.2.0
- Missing: explicit "how to write a custom diagnostic rule" tutorial in README
- Missing: go.work usage notes for contributors

### 4. Release Workflow

- `.github/workflows/release.yml` tests `go test -race ./...` and `golangci-lint run ./...`
- But: it does **NOT** test submodules (`diagnose/git`, `diagnose/postgres`) individually
- No coverage threshold enforcement in CI

---

## C) NOT STARTED

1. **go-git migration for GitRule** — Phase 3 of the original plan. GitRule still shells out to `git` binary. Replacing with `go-git` would eliminate runtime dependency on git CLI.
2. **Typed context keys** — `ContextKey` type + constants for all rule keys (e.g., `const KeyHost ContextKey = "host"`)
3. **Coverage thresholds in CI** — No minimum coverage gate
4. **Submodule-specific CI jobs** — GitHub Actions doesn't run `go test ./...` in `diagnose/git/` or `diagnose/postgres/`
5. **GOWORK=off CI verification** — Need to verify modules build WITHOUT the workspace file
6. **Benchmarks** — No benchmark tests for `Classify`, `HandleError`, `Runner.Run`
7. **Fuzz tests** — No fuzzing for `Classify`, `ParseFamily`, error constructors
8. **Example programs** — No `examples/` directory showing real-world usage
9. **Integration tests** — No CI job that actually starts postgres / checks a real git repo
10. **godoc comments on all exported symbols** — Some helpers lack full godoc
11. **go.mod version tagging for submodules** — Submodules depend on `v0.1.2` of root; when root bumps, submodules need updating
12. **CHANGELOG migration guide** — Breaking change needs explicit "how to migrate" section
13. **Release automation for submodules** — No automated tagging for `diagnose/git/vX.Y.Z` etc.
14. **Performance profiling** — No `pprof` integration or performance baselines
15. **Code duplication cleanup** — Test patterns duplicated across git/postgres test files (table-driven test boilerplate)

---

## D) TOTALLY FUCKED UP

### 1. Nix Build Check (`nix flake check`)

```
failed to initialize build cache at /homeless-shelter/.cache/go-build:
mkdir /homeless-shelter: permission denied
```

**Root cause:** Nix sandbox sets `HOME=/homeless-shelter` but the builder runs as a non-root user without write access. Go tries to create its build cache at `$HOME/.cache/go-build`.

**Fix:** Set `HOME=$TMPDIR` in the build derivation.

**Deeper issue:** With `GOWORK=off`, submodules will try to download `github.com/larsartmann/go-error-family v0.1.2` from the internet. In a pure Nix build (no internet), this may fail unless the module is already in the Go module proxy cache. We need a `vendor` directory or `replace` directives for reproducible Nix builds.

### 2. `nix build` — No Default Package

```
error: flake does not provide attribute 'packages.x86_64-linux.default'
```

This is expected (it's a library, not a binary), but the error message is unfriendly. We should either add a `packages.default` that builds the library or document that `nix build` is not applicable.

### 3. Test Duplication

jscpd reports **6.99% duplicated lines in Go code**. Much of this is:

- Identical table-driven test patterns across `diagnose/git/rules_git_test.go`, `diagnose/postgres/rules_postgres_test.go`, and `diagnose/diagnose_test.go`
- Boilerplate: `tests := []struct{ name string; err error; want bool }` followed by identical `t.Run` loops

**Not catastrophic** but worth a shared test helper if we add more submodules.

### 4. `report/jscpd-report.json` is a Generated File in Git

This file is 692 lines, 47.98% self-duplicated (it's a JSON report of its own clones). It should be in `.gitignore` and generated on demand.

---

## E) WHAT WE SHOULD IMPROVE

### Immediate (This Week)

1. **Fix `flake.nix` build check** — Set `HOME=$TMPDIR` in the derivation
2. **Add `report/` to `.gitignore`** — Generated reports don't belong in git
3. **Add CI jobs for submodules** — `go test ./...` in `diagnose/git` and `diagnose/postgres`
4. **Add GOWORK=off CI check** — Verify modules build standalone without workspace

### Short-Term (This Month)

5. **Raise test coverage for git/postgres** — Mock `RunCommand` with an interface for testability
6. **Add benchmarks** — `BenchmarkClassify`, `BenchmarkRunnerRun`, `BenchmarkHandleError`
7. **Add fuzz tests** — `FuzzClassify`, `FuzzParseFamily`
8. **Typed context keys** — `type ContextKey string` + exported constants
9. **go-git migration** — Replace `exec.Command("git", ...)` with `go-git` library
10. **Add `examples/` directory** — Real-world usage patterns

### Medium-Term (Next Quarter)

11. **Coverage thresholds in CI** — Fail build if coverage drops below 70% (core) / 50% (submodules)
12. **Integration test suite** — Docker-based postgres tests, temp-git-repo tests
13. **Performance baselines** — Track benchmark regressions in CI
14. **Automated submodule versioning** — Tag `diagnose/git/v0.2.0` when root tags `v0.2.0`
15. **Migration guide** — Explicit "Upgrading from v0.1.x" document

---

## F) TOP #25 THINGS TO GET DONE NEXT

### P0 — Blockers for v0.2.0 Release

1. **Fix `nix flake check` build failure** (`HOME=$TMPDIR`)
2. **Add `report/` to `.gitignore`**
3. **Add CI tests for submodules** (`.github/workflows/release.yml`)
4. **Verify GOWORK=off builds** in CI
5. **Update release workflow** to run `go test ./...` in each module directory

### P1 — Quality & Polish

6. **Mock `RunCommand`/`CommandExists`** for git/postgres tests (raise coverage to 80%+)
7. **Add benchmark suite** (`BenchmarkClassify`, `BenchmarkRunnerRun`, `BenchmarkHandleError`)
8. **Add fuzz tests** for `Classify` and `ParseFamily`
9. **Add `examples/` directory** with 3 real-world examples
10. **Add migration guide** to CHANGELOG or separate `MIGRATING.md`
11. **Add `go.work` usage notes** to CONTRIBUTING.md

### P2 — Features

12. **Migrate GitRule to go-git** — Eliminate runtime `git` CLI dependency
13. **Typed context keys** — `ContextKey` type + constants for all standard keys
14. **Add `DefaultRunnerMinimal()`** or `DefaultRunnerWith(opts ...RuleOption)` — More flexible runner construction
15. **Add `Timeout` to `DiagnosticResult`** — Already tracked but not exposed in API docs
16. **Add `Context()` to `DiagnosticResult`** — Surface the context used for the diagnosis

### P3 — Architecture & Long-Term Health

17. **Add `internal/` package** for shared test helpers to reduce duplication
18. **Add coverage badge** to README (shields.io)
19. **Add `go vet` to CI** (already in flake, add to GitHub Actions)
20. **Add `go mod verify` to CI**
21. **Add binary size analysis** — Track growth of root package
22. **Add API compatibility check** — `apidiff` or similar for breaking change detection
23. **Add `go-test-coverage` threshold** in CI
24. **Automated submodule tagging** — Script to tag `diagnose/git/vX.Y.Z` when root tags `vX.Y.Z`
25. **Consider `pkg/` structure** — The go-structure-linter complains about root package files. Not required for a library this size, but worth evaluating.

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

### How do we make `nix flake check` work correctly for a multi-module Go workspace?

**The problem:**

1. Submodules (`diagnose/git`, `diagnose/postgres`) have `require github.com/larsartmann/go-error-family v0.1.2` in their `go.mod`
2. The `flake.nix` `checks.build` sets `GOWORK=off` (correct for verifying standalone module builds)
3. With `GOWORK=off`, Go tries to download the root module from the internet to satisfy submodule dependencies
4. In a pure Nix build (sandbox, no network), this fails unless:
   - The exact version is in the Go module proxy cache (flake doesn't set this up), OR
   - We use `replace` directives in submodules (skill says: "Never mix replace AND go.work"), OR
   - We vendor all dependencies, OR
   - We build WITH `go.work` enabled but `go.work` uses relative paths that break when Nix copies source to `/build/`

**What I've tried:**

- Setting `GOWORK=off` causes the submodule to fetch root from internet → fails in sandbox
- Keeping `GOWORK=on` means `go.work` relative paths (`./diagnose/git`) must resolve from the build directory → may or may not work depending on copy structure
- Adding `replace` directives to submodules would fix Nix but violates the go-modularize skill's rule about mixing replace + go.work

**What I think the answer might be:**

- Use `GOWORK=off` but add `replace` directives ONLY for the build check derivation (patch go.mod during build)
- Or: use `buildGoModule` from nixpkgs which handles Go module vendoring properly
- Or: accept that `nix flake check` for a multi-module Go library is inherently tricky and document the limitation

**But I genuinely don't know which approach is idiomatic for Nix + Go workspaces. This is a real gap in my knowledge and the go-modularize skill doesn't cover Nix specifics.**

---

## Metrics at a Glance

| Metric                       | Value                                      |
| ---------------------------- | ------------------------------------------ |
| Go files                     | 19                                         |
| Total lines of Go            | ~3,619                                     |
| Modules                      | 3 (root, diagnose/git, diagnose/postgres)  |
| Tests passing                | 100% (all modules)                         |
| Lint issues                  | 0 (all modules)                            |
| Coverage (root)              | 97.2%                                      |
| Coverage (agent)             | 100%                                       |
| Coverage (diagnose core)     | 66.8%                                      |
| Coverage (diagnose/git)      | 23.1%                                      |
| Coverage (diagnose/postgres) | 44.8%                                      |
| External dependencies        | 0 (root), 1 each (submodules: root module) |
| Open TODO/FIXME/HACK         | 0                                          |
| Commits since last status    | 5                                          |

---

## Recent Commits

```
87e96c0 feat(diagnose)!: modularize GitRule and PostgresRule into submodules
585a2eb docs(planning): modularize diagnostic rules decision record
67aecd3 feat(classify): add errors.Join support with fail-closed classification
fcadb10 docs(status): comprehensive status report — post-lint-cleanup
d8deac7 chore: add flake.lock for Nix flake dependency lockfile
```

---

## Next Action (Recommended)

**Fix `flake.nix` build check** — This is the only broken thing in the project right now. Everything else is "nice to have."
