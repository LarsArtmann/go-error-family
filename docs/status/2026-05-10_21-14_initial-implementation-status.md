# Status Report — go-error-family

**Date:** 2026-05-10 21:14
**Repo:** `github.com/larsartmann/go-error-family`
**Branch:** master
**Commit:** `917f583` — feat: initial implementation of go-error-family
**Status:** Clean working tree, 26/26 tests passing, zero build errors

---

## Executive Summary

Created `go-error-family` from scratch — a structured error protocol library for Go with behavioral classification, BSD exit codes, deterministic diagnostic rules, and a configurable AI debug agent. The library is the direct output of a deep first-principles design session that audited five incompatible error systems across the ecosystem, analyzed cockroachdb/errors and uniflow/errors, and incorporated the Wix UX error message framework.

**The library is a prototype. It compiles, tests pass, and the architecture is sound. But it has zero consumers, zero integration tests for the diagnostic rules, and no CI pipeline.**

---

## A. FULLY DONE

### Core Protocol Package (`errorfamily`)

| Component | File | Lines | Status |
|---|---|---|---|
| `Family` int enum | `family.go` | 160 | Complete — 5 families with String/IsRetryable/ExitCode/Tone/Audience |
| Consumer interfaces | `interfaces.go` | 38 | Complete — Coded, Classified, Contextual, Retryable (all embed `error` for Go 1.26 `errors.AsType`) |
| Reference `Error` struct | `error.go` | 185 | Complete — Is/Unwrap/Format/Context/Summary/MatchesContext |
| Classification engine | `classify.go` | 99 | Complete — Classify/IsRetryable/ExitCode/RegisterClassification/RegisterClassifications |
| Constructors | `constructors.go` | 91 | Complete — New/Newf/Wrap/Wrapf + 5 family-specific constructors each |
| CLI boundary handler | `handle.go` | 268 | Complete — HandleError/HandleErrorDetailed/MessageTemplate/Wix framework |
| Tests | `errorfamily_test.go` | 397 | Complete — 26 tests covering all core functionality |

**Test coverage:**
- Family: String, ParseFamily, IsRetryable, ExitCode, Tone
- Error: Error(), Unwrap(), Is(), ErrorCode(), ErrorFamily(), ErrorContext(), context isolation
- Error: Format (%s, %v, %+v), Summary, MatchesContext, MatchesContextValue
- Constructors: all 10 family-specific + New/Newf/Wrap/Wrapf + nil handling
- Classification: nil → Rejection, classified interface → uses Family, retryable interface → infers, registered sentinels → lookup, default → Transient
- Chain traversal: errors.Is through wrapped chain, errors.AsType[Coded] through chain
- External type: verified that custom structs implementing our interfaces work with Classify/AsType

### Diagnostic System (`diagnose`)

| Component | File | Lines | Status |
|---|---|---|---|
| Runner + interfaces | `diagnose.go` | 260 | Complete — concurrent rule execution, confidence sorting |
| System snapshot | `context.go` | 117 | Complete — OS/Arch/Hostname/Env with secret redaction |
| PostgresRule | `rules_postgres.go` | 139 | Complete — pg_isready, TCP fallback, platform-aware fix suggestions |
| FilesystemRule | `rules_filesystem.go` | 173 | Complete — path existence, permissions, writability, auto-fix for mkdir |
| NetworkRule | `rules_network.go` | 99 | Complete — DNS resolution, TCP connectivity |
| GitRule | `rules_git.go` | 130 | Complete — repo check, dirty state, merge conflicts, remote reachability |

### AI Debug Agent (`agent`)

| Component | File | Lines | Status |
|---|---|---|---|
| Full agent scaffold | `agent.go` | 374 | Complete — 4 involvement levels, risk classification, command allow/deny, deterministic fallback |

### Documentation

| Doc | Status |
|---|---|
| README.md | Complete — quick start, constructors, classification, custom types, diagnostics, agent, architecture |
| Design document (in docs/ repo) | Complete — 1,119 lines covering audit, analysis, first principles, final design |

---

## B. PARTIALLY DONE

### Diagnostic Rules — No Integration Tests

The four diagnostic rules (Postgres, Filesystem, Network, Git) have zero test files. They compile and the logic is structurally sound, but:

- **PostgresRule** — never tested against a real or mock PostgreSQL instance
- **FilesystemRule** — never tested with real filesystem operations (temp dirs)
- **NetworkRule** — never tested against real DNS resolution
- **GitRule** — never tested against real git repos

These rules use `os.Stat`, `exec.Command`, `net.Dial` directly — no interfaces, no dependency injection. Testing them properly requires either:
1. Extracting command/connection logic behind interfaces
2. Using temp directories and mock servers in tests
3. Accepting that some tests only pass in environments with the tools installed

### AI Debug Agent — Scaffold Only

The agent has:
- Full type system (Involvement, RiskLevel, Config, FixStep, AgentResult)
- Involvement-based approval logic (shouldApply)
- Deterministic analysis fallback (deterministicAnalyze)
- Prompt building (buildPrompt)

But it does NOT have:
- An actual AI provider integration (no OpenAI/Anthropic/Crush SDK calls)
- Command execution sandbox (ApplyFixes marks steps as applied but doesn't execute)
- Any tests

### CLI Boundary Handler — Basic Templates Only

`HandleError()` works end-to-end but the template system is primitive:
- `codeToWhat()` and `codeToFix()` use `strings.Contains` on code names — fragile
- No `MessageTemplate` overrides registered by default
- No integration with the diagnostic runner (HandleConfig.DiagnosticRunner is declared but not wired)
- `HandleErrorDetailed()` returns structured data but no consumer uses it yet

---

## C. NOT STARTED

### Integration with Any Consumer Repo

Zero repos import `go-error-family`. The library exists in isolation.

| Repo | What needs to happen | Status |
|---|---|---|
| **go-cqrs-lite** | Extract Family/Classify/RegisterClassification into go-error-family import | Not started |
| **go-finding** | Add ErrorCode()/ErrorContext() methods to FindingError | Not started |
| **hierarchical-errors** | Add ErrorCode() to its Error struct | Not started |
| **auto-deduplicate** | Add ErrorFamily()/ErrorContext(), move Present() to CLI layer | Not started |
| **docs-organizer** | Add Is(), ErrorCode(), ErrorContext() to DocsError | Not started |

### CI/CD Pipeline

No GitHub Actions, no linting pipeline, no release automation.

### Go Module Publishing

`go.mod` exists but the module has never been pushed to GitHub. No tagged releases.

### Linting

No `.golangci.yml` configuration. LSP shows hints and warnings:
- `agent/agent.go`: stringsseq, stringscutprefix, QF1012 (WriteString → Fprintf) hints
- `diagnose/context.go`: unused function `formatCommandFix`
- `diagnose/rules_filesystem.go`: unused function `isDirectory`
- `handle.go`: unchecked `fmt.Fprintln` return value

### Diagnose Package Tests

Zero test files in `diagnose/` and `agent/`.

### Nix Flake

No `flake.nix` — the LarsArtmann ecosystem standard for build automation.

### ADR (Architecture Decision Records)

No formal ADRs in `docs/adr/`. The design document captures the reasoning but not in ADR format.

---

## D. TOTALLY FUCKED UP

### Nothing Is Broken

Build is clean. Tests pass. No compile errors. No panics.

### But These Are Real Problems:

1. **Diagnostic rules are untestable as written** — `runCommand()`, `commandExists()`, `net.Dial`, `os.Stat` are called directly with no interfaces. This means: no unit tests possible without refactoring, or tests that only work if you have PostgreSQL/Git installed. This is a structural testing gap, not a bug.

2. **`HandleError()` doesn't wire to diagnostics** — `HandleConfig` has a `DiagnosticRunner` field and `Diagnose bool` field, but `HandleErrorWithConfig()` never calls the runner. The diagnostic system and the CLI handler are two separate systems that don't talk to each other yet.

3. **No `Is()` on `*Error` for sentinel matching** — `Error.Is()` matches by Code+Family, which is good for typed errors. But there's no equivalent of cockroachdb's `Mark()` for stamping identity onto arbitrary errors. `RegisterClassification` exists for third-party sentinels, but if you want to mark your own error as matching a specific target, you can't do it without creating a new `*Error` with the same code+family.

4. **Agent `ApplyFixes` doesn't actually execute** — It marks steps as applied and returns them. No command execution. No sandboxing. This is intentional for safety in the prototype, but it means the autonomous involvement level is a lie — it approves everything and does nothing.

5. **`classify.go` has a data race** — `RegisterClassification` is called from `init()` in other packages, which is fine. But `Classify()` reads `registry.entries` in a loop, and if someone calls `RegisterClassification` concurrently (outside init), there's a potential race on the map. The RLock/RWMutex protects the map itself, but `errors.Is` calls within the locked section could deadlock if the target's `Is()` method tries to classify. This is a theoretical concern, not a practical one.

---

## E. WHAT WE SHOULD IMPROVE

### Architecture

1. **Extract command execution behind interfaces in diagnose** — `CommandRunner` interface with a real and mock implementation. Makes rules testable without PostgreSQL/Git.
2. **Wire HandleError to the diagnostic runner** — HandleConfig.DiagnosticRunner should actually run diagnostics and include findings in the output.
3. **Add `Mark(err, sentinel)` function** — Inspired by cockroachdb/errors. Stamps identity onto errors without requiring RegisterClassification.

### Quality

4. **Write diagnose package tests** — Use temp directories for FilesystemRule, mock connections for PostgresRule/NetworkRule, temp git repos for GitRule.
5. **Write agent package tests** — Test involvement level logic, deterministic analysis, Config validation.
6. **Fix all LSP hints/warnings** — Remove unused functions, use Fprintf instead of WriteString+Sprintf, simplify string operations.
7. **Add .golangci.yml** — Match the ecosystem standard from go-cqrs-lite/auto-deduplicate.
8. **Add HandleError tests** — Test the CLI boundary handler with various error types, templates, and families.

### Ecosystem

9. **Push to GitHub and tag v0.1.0** — Makes it importable by other repos.
10. **Add go-error-family to LIBRARY_GUIDE.md** — So AI assistants know to use it.
11. **Write ADR for the Family design** — Formal record of why 5 families, why int not string, why embed error in interfaces.

---

## F. TOP 25 THINGS TO DO NEXT

### Critical — Quality (Do First)

| # | Task | Effort | Impact |
|---|---|---|---|
| 1 | Write diagnose package tests (FilesystemRule with temp dirs) | 2h | Rules are currently untested |
| 2 | Fix all LSP hints/warnings (unused funcs, string hints) | 30min | Code hygiene |
| 3 | Write agent package tests (involvement levels, deterministic analysis) | 1h | Agent is untested |
| 4 | Write HandleError tests (all families, templates, context substitution) | 1h | CLI handler is untested |
| 5 | Wire HandleError to diagnostic runner | 30min | Two systems don't talk to each other |

### High — Testability

| # | Task | Effort | Impact |
|---|---|---|---|
| 6 | Extract CommandRunner interface for diagnose rules | 1h | Makes all rules mockable |
| 7 | Add ConnectionTester interface for PostgresRule/NetworkRule | 30min | Makes network rules mockable |
| 8 | Write PostgresRule integration test with mock server | 1h | Validates pg_isready logic |
| 9 | Write GitRule integration test with temp repos | 1h | Validates git status logic |

### High — Ecosystem Integration

| # | Task | Effort | Impact |
|---|---|---|---|
| 10 | Push repo to GitHub | 15min | Importable by consumers |
| 11 | Tag v0.1.0-alpha | 5min | Signals API stability expectations |
| 12 | Add go-error-family to docs/LIBRARY_GUIDE.md | 30min | Discoverability |
| 13 | Add go-error-family to projects-management-automation go.work | 15min | Workspace integration |

### Medium — Feature Completeness

| # | Task | Effort | Impact |
|---|---|---|---|
| 14 | Add `Mark(err, sentinel)` function | 30min | Identity stamping without global registry |
| 15 | Add golangci.yml configuration | 30min | Consistent linting |
| 16 | Write ADR-001: Why Family int over string categories | 30min | Architecture documentation |
| 17 | Add Nix flake.nix for build/test automation | 1h | Ecosystem standard |
| 18 | Add CI pipeline (GitHub Actions: build, test, vet, lint) | 1h | Automated quality gates |

### Medium — First Consumer Migration

| # | Task | Effort | Impact |
|---|---|---|---|
| 19 | Migrate go-cqrs-lite: import go-error-family for Family/Classify | 2h | Proves the protocol works |
| 20 | Add interfaces to docs-organizer (Is, ErrorCode, ErrorContext) | 30min | Second consumer |
| 21 | Add interfaces to go-finding (ErrorCode, ErrorContext) | 30min | Third consumer |

### Lower — Polish

| # | Task | Effort | Impact |
|---|---|---|---|
| 22 | Wire AI agent to a real provider (Crush SDK or OpenAI) | 3h | Agent actually works |
| 23 | Add message template overrides for common error codes | 1h | Better default UX |
| 24 | Add IsPostgresRunning standalone helper to go-cqrs-lite | 15min | Useful utility |
| 25 | Write examples/ directory with runnable Go examples | 1h | GoDoc integration |

---

## G. TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should the diagnostic rules accept interfaces for their external dependencies (command execution, network connections, filesystem operations), or should they stay as direct os/exec/net calls and only be tested via integration tests?**

The tradeoff:

| Approach | Pro | Con |
|---|---|---|
| **Interfaces** (CommandRunner, ConnectionTester, FileSystem) | Unit-testable without tools installed. Mock-able. CI-friendly. | More code. More abstractions. Rules become less readable. |
| **Direct calls** + integration tests | Simple, readable rules. No abstraction overhead. | Tests require PostgreSQL/Git installed. CI must provision services. Can't test edge cases (network timeouts) easily. |

The rest of the LarsArtmann Go ecosystem (go-cqrs-lite, go-finding) uses concrete implementations in production and integration tests. But auto-deduplicate uses samber/do for DI. There's no consistent pattern.

My recommendation: **interfaces** — the rules are small, the interfaces would be tiny (one or two methods each), and testability is worth the small abstraction cost. But this is a style decision that should match the ecosystem convention, which I can't determine alone.

---

## Metrics

| Metric | Value |
|---|---|
| Total lines of Go code | 2,644 |
| Total files | 14 Go files + README + go.mod |
| Test files | 1 (errorfamily_test.go) |
| Test cases | 26 |
| Test pass rate | 100% (26/26) |
| Build errors | 0 |
| LSP errors | 0 |
| LSP warnings | 9 (hints + unused funcs) |
| Packages | 3 (errorfamily, diagnose, agent) |
| Dependencies | 0 (stdlib only) |
| Consumers | 0 |

---

_Arte in Aeternum_
