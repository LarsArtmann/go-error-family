# Comprehensive Status Report — 2026-06-01

**Project:** `github.com/larsartmann/go-error-family`
**Branch:** `master`
**Date:** 2026-06-01 06:52 UTC
**Version:** v0.3.0-dev (uncommitted)

---

## A) FULLY DONE

### Core Library (v0.2.0 + v0.3.0-dev)

- Five Families: Rejection, Conflict, Transient, Corruption, Infrastructure — with exit codes, retry semantics, Audience, Tone
- Consumer interfaces: `Coded`, `Classified`, `Contextual`, `Retryable` — all embed `error` for Go 1.26 `errors.AsType`
- `Error` struct: `fmt.Formatter` (%v, %+v, %s), `WithContext` chain, `WithCause`, `WithTimestamp`, `Summary()`
- `Classify()` with multi-error (`errors.Join`), registered sentinels, deadlock-safe snapshot
- `HandleError()` / `HandleErrorWithContext()` — CLI boundary handler with Wix-style templates
- `HandleErrorDetailed()` / `HandleErrorDetailedWithConfig()` — structured result for HTTP/gRPC
- Template system: `RegisterTemplate()`, consumer overrides, built-in defaults, family fallback
- `RegisterClassification()` / `RegisterClassifications()` — third-party sentinel mapping
- `Compose()` — `errors.Join` wrapper for partial-success patterns
- Dead code removed: SystemSnapshot, ApplyFixes, Involvement, RiskLevel, AutoFix (v0.2.0)

### Diagnostic System

- `CommandRunner` interface + `DefaultCommandRunner` — injectable command execution
- `ContextKey` typed constants: `KeyHost`, `KeyPort`, `KeyPath`, `KeyDBHost`, etc. (23 constants)
- `RuleSpec` with typed `ContextKeys []ContextKey` field
- `DiagnosticResult.Context` — surfaces error context that triggered the rule
- `ErrorContext(err)` — exported helper for extracting context from any error
- Zero-dep core rules: `FilesystemRule`, `NetworkRule` in `DefaultRunner()`
- Submodule rules: `GitRule` (diagnose/git), `PostgresRule` (diagnose/postgres) — both with `CommandRunner` injection
- All rules populate `DiagnosticResult.Context`

### Agent

- Deterministic analysis-only `DebugAgent` — proposes, never executes
- `FixStep{Description, Command, Rationale}` — consumer decides what to do

### Testing & Quality

- All tests pass with `-race`, 0 lint issues across 5 modules
- Root: 95.9% | Agent: 100% | Diagnose core: 67.1% | Git: 98.5% | Postgres: 81.0%
- 16 benchmarks + 5 fuzz tests
- Mock `CommandRunner` in git/postgres tests
- 5 package-level `Example` functions
- 3 runnable examples (CLI, HTTP, custom rule)

### Infrastructure

- Multi-module workspace (go.work) with 3 go.mod files
- Nix flake (devShell, formatter, format check)
- CI: test + lint + build (GitHub Actions)
- Zero external dependencies in root module

---

## B) PARTIALLY DONE

### 1. godoc Quality

- Root package symbols audited and fixed (20 issues)
- **But:** `diagnose` package exported helpers (`HasContextKey`, `ContextValue`, `ResolveContextKey`, etc.) lack individual examples
- `agent` package has minimal doc — no usage examples in godoc

### 2. SKILL.md

- Updated for v0.2.0 APIs but **not yet updated for v0.3.0 additions** (`HandleErrorWithContext`, `CommandRunner`, `ContextKey`, `DiagnosticResult.Context`, `Compose`, `WithTimestamp`)

### 3. Context Propagation

- `HandleErrorWithContext` properly passes context to `DiagnosticFunc`
- **But:** `FilesystemRule.Run` accepts `ctx` but never uses it for `os.Stat`/`os.Create` calls
- **But:** `Runner.Run` doesn't enforce context cancellation — hung rules block forever
- **But:** `NetworkRule.Run` uses hardcoded 3s timeout instead of respecting context deadline

---

## C) NOT STARTED

1. **`Compose` doc fix** — comment claims it returns "worst Family" but it just calls `errors.Join`
2. **`extractCommand` in agent** — looks for `$ ` and `Run: ` prefixes but no rule produces those formats. `FixStep.Command` is always empty in practice
3. **`NetworkRule.resolveHost`** — naive URL parsing breaks on IPv6, user:pass@host, query strings
4. **Test pollution cleanup** — global registries mutated in tests without `t.Cleanup`
5. **Family-specific format constructors** — `NewRejectionf`, `WrapTransientf` etc. missing
6. **`applyContext` template substitution** — no unit tests for edge cases
7. **Mock runner deduplication** — identical mocks in git_test and postgres_test
8. **`context.go` naming** — misleading filename for command execution helpers
9. **Concurrent safety tests** — for `RegisterTemplate`, `RegisterClassification`

---

## D) TOTALLY FUCKED UP

### 1. `Compose` Is a Lie

`classify.go:80` — The doc comment says "returns the 'worst' Family among them (highest exit code)" but the function body is literally `return errors.Join(errs...)`. It computes nothing. This is fraudulent documentation.

### 2. `extractCommand` Never Works

`agent/agent.go:151-162` — Searches for `$ ` and `Run: ` prefixes in `SuggestedFix`. But ALL diagnostic rules produce suggestions like `"Start PostgreSQL:\n  brew services start postgresql"`. None use `$ ` or `Run: `. So `FixStep.Command` is always `""`. This is a dead feature that looks like it works.

### 3. Test Registry Pollution

`RegisterClassification`, `RegisterClassifications`, `RegisterTemplate` all mutate global state in tests. Entries persist across test runs. No `t.Cleanup()` to undo. This can cause flaky tests if test ordering changes.

### 4. `FilesystemRule.suggestCreate` — Broken File/Dir Detection

`rules_filesystem.go:120-124` — Uses `strings.Contains(path, ".")` to guess if a path is a file or directory. Breaks on `.config`, `my.project/`, `Makefile`, `Dockerfile`.

---

## E) WHAT WE SHOULD IMPROVE

### Immediate Impact (Fix broken things first)

1. Fix `Compose` — either implement what the doc says, or change the doc to match reality
2. Fix `extractCommand` — make it match the actual fix format our rules produce, or remove it
3. Fix test pollution — add `t.Cleanup()` for global registry mutations
4. Fix `FilesystemRule.suggestCreate` — stop using `.` detection for file vs directory
5. Fix `NetworkRule.resolveHost` — use `net/url.Parse` instead of string hacking

### Short-Term (Architecture quality)

6. Deduplicate mock runners — extract to shared test helper in `diagnose` package
7. Rename `context.go` → `command.go` — file contains `RunCommand`/`CommandExists`, not context helpers
8. Add missing `ContextKey` constants for `"postgres_port"` and `"PGHOST"` / `"PGPORT"`
9. Update `gitSpec` to use `diagnose.KeyGit` instead of raw string `"git"`
10. Add `Runner.Run` context cancellation enforcement

### Medium-Term (Polish)

11. Update SKILL.md for v0.3.0 APIs
12. Add `RegisterClassification`/`RegisterTemplate` concurrent safety tests
13. Add `applyContext` unit tests for template edge cases
14. Consider `net/url.Parse` for `NetworkRule.resolveHost`
15. Use `net.Dialer` with `DialContext` in `NetworkRule` instead of `DialTimeout`

### Long-Term (Architecture evolution)

16. Consider consolidating `DiagnosticFinding` (root) and `DiagnosticResult` (diagnose) — they're split brains
17. Consider renaming `DebugAgent` interface → `Agent` in the agent package
18. Consider adding `KeyPort` constant usage in `PostgresRule.resolvePort`
19. Consider family-specific format constructors (`NewRejectionf`, etc.)
20. Consider extracting a shared test utilities package for mock runners

---

## F) TOP #25 THINGS TO GET DONE NEXT

Sorted by impact × effort (highest first):

| #   | Task                                                                 | Impact   | Effort | Category     |
| --- | -------------------------------------------------------------------- | -------- | ------ | ------------ |
| 1   | Fix `Compose` doc comment (it's a lie)                               | Critical | 2min   | Bug          |
| 2   | Fix `extractCommand` to match actual fix formats                     | Critical | 10min  | Bug          |
| 3   | Add `t.Cleanup()` for test registry pollution                        | High     | 15min  | Bug          |
| 4   | Fix `FilesystemRule.suggestCreate` file/dir detection                | High     | 10min  | Bug          |
| 5   | Fix `NetworkRule.resolveHost` with `net/url.Parse`                   | Medium   | 15min  | Bug          |
| 6   | Use `net.Dialer` + `DialContext` in NetworkRule                      | Medium   | 10min  | Bug          |
| 7   | Rename `diagnose/context.go` → `diagnose/command.go`                 | Low      | 1min   | Cleanup      |
| 8   | Deduplicate mock runners to shared helper                            | Low      | 20min  | Cleanup      |
| 9   | Use `diagnose.KeyGit` in git spec instead of raw string              | Low      | 2min   | Cleanup      |
| 10  | Add missing `ContextKey` constants for postgres_port, PGHOST, PGPORT | Low      | 5min   | Feature      |
| 11  | Use ContextKey constants in PostgresRule.resolvePort                 | Low      | 5min   | Cleanup      |
| 12  | Update SKILL.md for v0.3.0 APIs                                      | High     | 30min  | Docs         |
| 13  | Remove dead `var _ = fmt.Sprintf` from postgres tests                | Low      | 1min   | Cleanup      |
| 14  | Add `Runner.Run` context cancellation enforcement                    | Medium   | 20min  | Feature      |
| 15  | Add concurrent safety tests for registries                           | Medium   | 15min  | Test         |
| 16  | Add `applyContext` unit tests                                        | Low      | 10min  | Test         |
| 17  | Add family-specific format constructors                              | Low      | 15min  | Feature      |
| 18  | Consolidate `DiagnosticFinding` vs `DiagnosticResult` types          | High     | 60min  | Architecture |
| 19  | Consider renaming `DebugAgent` → `Agent` interface                   | Low      | 10min  | API          |
| 20  | Remove unused `IsPostgresRunning` or add real tests                  | Low      | 10min  | Cleanup      |
| 21  | Add `HandleErrorWithContext` direct tests                            | Medium   | 10min  | Test         |
| 22  | Add `Error.Is` tests through wrapped error chains                    | Low      | 5min   | Test         |
| 23  | Add godoc examples for diagnose package helpers                      | Low      | 20min  | Docs         |
| 24  | Consider removing `KeyDirectory` alias (just use `KeyDir`)           | Low      | 5min   | Cleanup      |
| 25  | Remove unused `KeyRepoPath` or document why it exists                | Low      | 5min   | Cleanup      |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should `Compose` actually compute the "worst" Family, or should it be removed?**

The doc comment says it returns the "worst Family among them (highest exit code)." But the body is just `errors.Join(errs...)`, which does no classification at all. The classification happens in `Classify()` which already handles `errors.Join` multi-errors correctly (first non-Transient wins). So `Compose` as currently documented is redundant — you can just use `errors.Join` directly and then `Classify`/`ExitCode` on the result.

Two options:

1. **Delete `Compose`** — `errors.Join` + `Classify` already does everything. No new API needed.
2. **Make `Compose` useful** — Have it actually return the worst Family/exit code alongside the joined error, like `func Compose(errs ...error) (error, Family)`.

I lean toward option 1 (delete) because Go already has `errors.Join` and our `Classify` handles multi-errors. Adding `Compose` that just wraps `errors.Join` adds no value and the misleading doc is worse than no function.

---

## Coverage Summary

| Package           | Before (v0.2.0) | After (v0.3.0-dev) |
| ----------------- | --------------- | ------------------ |
| root              | 97.2%           | 95.9%              |
| agent             | 100%            | 100%               |
| diagnose (core)   | 66.8%           | 67.1%              |
| diagnose/git      | 69.2%           | **98.5%**          |
| diagnose/postgres | 58.6%           | **81.0%**          |

## Build Status

| Check                                   | Result                      |
| --------------------------------------- | --------------------------- |
| `go build ./...`                        | ✅ Clean                    |
| `go test ./... -race`                   | ✅ All pass                 |
| `golangci-lint run ./...` (all modules) | ✅ 0 issues                 |
| Benchmarks                              | ✅ All pass, no regressions |
