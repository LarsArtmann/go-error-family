# Modularization Proposal: Extract agent/ and diagnose/ from Root Module

**Date:** 2026-06-17
**Project:** go-error-family v0.5.0
**State:** Partial split (4 modules, go.work, root is god-module)

---

## Executive Summary

The root module contains four concerns: stable classification core, experimental diagnostics, experimental agent, and examples. Extracting `agent/` and `diagnose/` into their own modules unblocks v1.0 of the classification core. The change is low-risk — both packages are already self-contained with clean dependency directions.

---

## Phase 1 — Current State

### Module Landscape

| Module | Path | Packages | Internal Deps | External Deps | Replace | State |
|---|---|---|---|---|---|---|
| root | `github.com/larsartmann/go-error-family` | core + agent/ + diagnose/ + examples/ | — | zero | — | **God-module** (stable + experimental mixed) |
| bridge | `github.com/.../bridge` | bridge/ | root (v0.3.0) | samber/oops | — | Clean |
| git | `github.com/.../diagnose/git` | git/ | root (v0.3.0) | zero | — | Clean |
| postgres | `github.com/.../diagnose/postgres` | postgres/ | root (v0.3.0) | zero | — | Clean |

### Starting State: Workspace mode with one god-module

The project already has `go.work` coordinating 4 modules. The problem is that the **root module is a god-module** — it mixes the stable classification core with experimental packages. The submodules (bridge, git, postgres) are clean.

### Key Problem

**Stale version pins.** All three submodules pin root at v0.3.0. After v0.5.0 breaking changes, these must be bumped to v0.5.0.

---

## Phase 1.5 — Re-modularization Assessment

### Existing Module Scores

| Module | Cohesion (1-5) | Coupling (1-5, lower=better) | Independent? | Action |
|---|---|---|---|---|
| **root** | **2** — contains core + diagnostics + agent + examples | 1 — zero external deps | No — experimental packages force version coupling with stable core | **Split** |
| **bridge** | 5 — single purpose (oops integration) | 2 — depends on root + oops | Yes | **Keep** (bump version pin) |
| **git** | 5 — single purpose (git diagnostics) | 1 — depends on root only | Yes | **Keep** (bump version pin) |
| **postgres** | 5 — single purpose (pg diagnostics) | 1 — depends on root only | Yes | **Keep** (bump version pin) |

### Proposed Remodel

| Old | Action | New Module(s) | Rationale |
|---|---|---|---|
| root (core) | **Keep** | `github.com/.../error-family` (v1.0 stable) | Core is ready for v1.0 |
| root/diagnose/ | **Extract** | `github.com/.../diagnose` (v0.x) | Experimental, zero-dep, depends on core only |
| root/agent/ | **Extract** | `github.com/.../agent` (v0.x) | Experimental, depends on core + diagnose |
| root/examples/ | **Keep in root** | Stays in root module | Leaf nodes, compiled by CI, share deps with root |
| bridge | **Keep** | No change | Clean module |
| git | **Keep** | No change (update deps: add diagnose) | Clean module |
| postgres | **Keep** | No change (update deps: add diagnose) | Clean module |

---

## Phase 2 — Research & Analysis

### Internal Dependency Graph

```
core (root package)
  ↑
  ├── diagnose/ ──────→ core (errorfamily.Classify, errorfamily.Family, errorfamily.Contextual)
  │     ↑
  │     ├── agent/ ───→ diagnose (DiagnosticResult, StatusFailed) + core (Classify)
  │     ├── git/ ─────→ diagnose (CommandRunner, RuleSpec) [via root module today]
  │     └── postgres/ → diagnose (FamilyIs, CommandRunner) + core (Transient) [via root module today]
  │
  ├── bridge/ ────────→ core (Family, IsRetryable) + oops
  └── examples/ ──────→ core (NewRejection, HandleError, etc.) [leaf nodes]
```

### Coupling Points

1. **agent → diagnose**: agent imports `diagnose.DiagnosticResult` and `diagnose.StatusFailed`. After extraction, agent module must depend on diagnose module. **Correct direction** — agent consumes diagnostic results.

2. **git/postgres → diagnose**: These submodules currently import `diagnose.CommandRunner` and `diagnose.RuleSpec` through the root module (because diagnose/ is in the root go.mod). After extraction, they must add an explicit `require` for the diagnose module.

3. **diagnose → core**: diagnose imports `errorfamily.Classify`, `errorfamily.Family`, `errorfamily.Contextual`, `errorfamily.Coded`. These are all stable core types. **Clean dependency.**

4. **No internal/ usage**: All packages are public. After extraction, `diagnose/mock.go` should be evaluated for `internal/` placement.

### Error Type Accessibility

All error types (`Family`, `Error`, interfaces) live in the core package. No sentinel errors cross module boundaries. `errors.Is`/`errors.As` work through the core types. **No accessibility risk.**

### God-Package Analysis

The root package itself (`errorfamily`) is NOT a god-package — it has 8 files totaling ~1300 LOC, all cohesive around error classification. The `diagnose/` sub-package has 6 files (~700 LOC), all cohesive around diagnostic rules. The `agent/` sub-package is a single file (199 LOC). **No god-packages — the problem is module-level, not package-level.**

---

## Phase 3 — Proposed Module Structure

### Module 1: Core (v1.0 STABLE)

| Field | Value |
|---|---|
| **Path** | `github.com/larsartmann/go-error-family` |
| **Purpose** | Behavioral error classification for Go (Family, Classify, ExitCode, HandleError) |
| **Deps (prod)** | Zero |
| **Deps (test)** | Zero |
| **Public API** | Family, Error, Coded, Classified, Contextual, Retryable, Classify, IsRetryable, ExitCode, Registry, NewRegistry, DefaultRegistry, HandleError, HandleErrorWithContext, HandleErrorDetailed, HandleErrorDetailedWithConfig, HandleConfig, HandleResult, MessageTemplate, DiagnosticFinding, DiagnosticFunc, RegisterClassification, RegisterTemplate, constructors (New, Wrap, NewRejection, etc.) |
| **External deps** | None |
| **LOC** | ~1300 (8 source files) |
| **Version target** | v1.0.0 |

### Module 2: Diagnose (v0.x EXPERIMENTAL)

| Field | Value |
|---|---|
| **Path** | `github.com/larsartmann/go-error-family/diagnose` |
| **Purpose** | Deterministic diagnostic rules that investigate errors by checking live system state |
| **Deps (prod)** | core (errorfamily) |
| **Deps (test)** | core |
| **Public API** | DiagnosticRule, DiagnosticResult, Runner, RuleSpec, DefaultRunner, FilesystemRule, NetworkRule, CommandRunner, DefaultCommandRunner, MockCommandRunner, ContextKey, ErrorContext, RunCommand, CommandExists, ResolveRunner, FamilyIs, HasContextKey, ContextValue, ResolveContextKey, HasContextSubstring, ErrorCodeContains, Status, StatusFailed, etc. |
| **External deps** | None |
| **LOC** | ~700 (6 source files) |
| **Version target** | v0.1.0 |

### Module 3: Agent (v0.x EXPERIMENTAL)

| Field | Value |
|---|---|
| **Path** | `github.com/larsartmann/go-error-family/agent` |
| **Purpose** | Deterministic root-cause synthesizer that produces fix suggestions from diagnostic results |
| **Deps (prod)** | core + diagnose |
| **Deps (test)** | core + diagnose |
| **Public API** | DebugAgent, Config, AgentResult, FixStep, New, Analyze |
| **External deps** | None |
| **LOC** | 199 (1 source file) |
| **Version target** | v0.1.0 |

### Module 4: Bridge (existing, bump version)

| Field | Value |
|---|---|
| **Path** | `github.com/larsartmann/go-error-family/bridge` |
| **Deps** | core (v0.5.0 → v1.0.0) + samber/oops |
| **Action** | Bump root pin from v0.3.0 to v1.0.0 |

### Module 5: Git (existing, add diagnose dep)

| Field | Value |
|---|---|
| **Path** | `github.com/larsartmann/go-error-family/diagnose/git` |
| **Deps** | core (v1.0.0) + **diagnose (v0.1.0) — NEW** |
| **Action** | Add diagnose module dep, bump root pin |

### Module 6: Postgres (existing, add diagnose dep)

| Field | Value |
|---|---|
| **Path** | `github.com/larsartmann/go-error-family/diagnose/postgres` |
| **Deps** | core (v1.0.0) + **diagnose (v0.1.0) — NEW** |
| **Action** | Add diagnose module dep, bump root pin |

### DAG Verification

```
core (v1.0)
  ↑
  ├── diagnose (v0.1) → core
  │     ↑
  │     └── agent (v0.1) → diagnose + core
  │
  ├── bridge → core + oops
  ├── git → diagnose + core
  └── postgres → diagnose + core
```

**Acyclic.** All arrows point toward core. No cycles. ✓

### Workspace Strategy

`go.work` at repo root. All modules listed. No `replace` directives in any go.mod. Consumers of published modules use versioned imports.

### Versioning Strategy

| Module | Strategy | Rationale |
|---|---|---|
| core | **v1.0.0** — frozen, semver guarantees | Stable classification core, no breaking changes |
| diagnose | **v0.1.0** — independent semver | Experimental, may change |
| agent | **v0.1.0** — independent semver | Experimental, may change |
| bridge | Continue at current version + bump root dep | Already independent |
| git, postgres | Continue at current version + update deps | Already independent |

---

## Phase 4 — Brutal Self-Review

| # | Question | Answer |
|---|---|---|
| 1 | What did we forget? | `examples/cmd/` uses both core AND could use diagnose in future examples. Today it only uses core — stays in root. ✓ |
| 2 | What could be improved? | Agent should depend on an interface for diagnostic results, not concrete `diagnose.DiagnosticResult`. But this is a future improvement, not a blocker for extraction. |
| 3 | Split brains? | No duplicate types. DiagnosticFinding (in core/handle.go) vs DiagnosticResult (in diagnose/) serve different purposes — the former is a minimal CLI-boundary type, the latter is a rich diagnostic engine type. Both are correctly placed. |
| 4 | Right granularity? | 3 modules + 3 submodules = 6 total. Each does one thing. No micro-modules. ✓ |
| 5 | Existing code reuse? | The existing submodule pattern (bridge, git, postgres) already proves the extraction works. We're applying the same pattern to agent and diagnose. |
| 6 | Type model quality? | Registry type (v0.5.0) already provides the injectable pattern. No type changes needed for extraction. |
| 7 | Reinventing the wheel? | No — this is standard Go multi-module workspace pattern. |
| 8 | Import paths verified? | All submodules already import `diagnose` through root. After extraction, they add a `require` for the diagnose module. The import path doesn't change. |
| 9 | Test deps isolated? | All packages have zero test-only external deps. ✓ |
| 10 | CI actually faster? | Yes — 3 independent test jobs instead of 1 combined. The current CI already parallelizes submodules. |
| 11 | Versioning realistic? | Core at v1.0, rest at v0.x — matches how the library is consumed (core is the stable interface, rest is opt-in). |
| 12 | Error types accessible? | All error types live in core. `errors.Is`/`errors.As` work through core interfaces. ✓ |
| 13 | internal/ safe? | No internal/ packages to break. Future improvement: move diagnose/mock.go behind internal/ after extraction. |
| 14 | Over-modularized? | No — 6 modules for a library with 5 distinct concerns (core, diagnostics, agent, bridge, git, postgres) is correct. |
| 15 | Consumers broken? | Import paths don't change — `github.com/.../diagnose` already works as a package import. Adding a go.mod at `diagnose/` makes it a module at the same path. ✓ |

---

## Phase 5 — Execution Plan

### Tier 1 — Core (foundational)

| Step | Task | Verification | Rollback |
|---|---|---|---|
| 1.1 | Create `diagnose/go.mod` with `module github.com/.../diagnose`, `require root v0.5.0` | `cd diagnose && go build ./...` | `rm diagnose/go.mod` |
| 1.2 | Create `agent/go.mod` with `module github.com/.../agent`, `require root v0.5.0 + diagnose v0.1.0` | `cd agent && go build ./...` | `rm agent/go.mod` |
| 1.3 | Update `go.work` to add `./diagnose` and `./agent` as workspace members | `go build ./...` from root | Revert go.work |

### Tier 2 — Untangle (high leverage)

| Step | Task | Verification | Rollback |
|---|---|---|---|
| 2.1 | Bump bridge root pin: `cd bridge && go get github.com/.../error-family@v0.5.0 && go mod tidy` | `cd bridge && GOWORK=off go build ./...` | Revert go.mod |
| 2.2 | Bump git root pin + add diagnose dep: `cd diagnose/git && go get ...@v0.5.0 && go mod tidy` | `cd diagnose/git && GOWORK=off go build ./...` | Revert go.mod |
| 2.3 | Bump postgres root pin + add diagnose dep: same pattern | `cd diagnose/postgres && GOWORK=off go build ./...` | Revert go.mod |

### Tier 3 — Infrastructure (completes the graph)

| Step | Task | Verification |
|---|---|---|
| 3.1 | Run full workspace: `go build ./... && go test ./... -race` | All green |
| 3.2 | Run standalone builds: `GOWORK=off go build ./...` in each module | All green |
| 3.3 | Update CI: add test/lint steps for diagnose/ and agent/ modules | CI passes |

### Tier 4 — Polish

| Step | Task |
|---|---|
| 4.1 | Tag core as v1.0.0, diagnose as v0.1.0, agent as v0.1.0 |
| 4.2 | Update README to reflect new module structure |
| 4.3 | Update AGENTS.md build commands per module |
| 4.4 | Update CHANGELOG with v1.0.0 entry |
| 4.5 | Consider moving `diagnose/mock.go` behind `internal/` |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Import cycle between diagnose and agent | Low | High | Already verified DAG — agent → diagnose, not reverse |
| Git/postgres submodules can't find diagnose types after extraction | Medium | Medium | They already import diagnose via root module; adding explicit require is the only change |
| go.work becomes stale after module creation | Medium | Low | Run `go work sync` after each step |
| External consumers confused by module path change | Low | Medium | Paths don't change — `github.com/.../diagnose` works as both package and module path |

---

## Build System Impact

The `flake.nix` build checks (`build`, `build-standalone`) run `go build ./...` which uses go.work. After extraction, these continue to work — go.work includes all modules. The `build-standalone` check (GOWORK=off) must be updated to run in each module directory.

CI already parallelizes per-submodule. Add two new jobs: `diagnose/` and `agent/`.
