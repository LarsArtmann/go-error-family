# Status Report: Module Extraction & Architecture Session

**Date:** 2026-06-17 12:23
**Project:** go-error-family
**Session Span:** v0.4.0 → v0.6.0 (module extraction complete)

---

## a) FULLY DONE

### v0.5.0 — Pro/Contra Review Execution (committed: `9edc7fb`)

| CON | Fix                                                     | Verification                           |
| --- | ------------------------------------------------------- | -------------------------------------- |
| 03  | `WithContext`/`WithCause`/`WithTimestamp` copy-on-write | Race-clean, 3 new isolation tests      |
| 09  | Dropped `Compose`                                       | Pointed at `errors.Join`               |
| 07  | Template syntax `{{.key}}` → `{key}`                    | All tests pass                         |
| 02  | Injectable `Registry` type                              | 9 isolation tests, backward compatible |
| 06  | README retry wording                                    | No longer implies backoff policy       |
| 10  | Experimental stability notices                          | All non-root packages marked v0.x      |

### v0.6.0 — Module Extraction (committed: `4ba7843`, `a83f311`)

| Task                                                    | Status                        |
| ------------------------------------------------------- | ----------------------------- |
| Extract `diagnose/` into own module (`diagnose/go.mod`) | Done — build, test, lint pass |
| Extract `agent/` into own module (`agent/go.mod`)       | Done — build, test, lint pass |
| Update `go.work` (6 modules)                            | Done                          |
| Local replace directives for unpublished modules        | Done — root, diagnose, agent  |
| CI updated with test/lint steps for diagnose and agent  | Done                          |
| CHANGELOG v0.6.0 entry                                  | Done                          |
| SKILL.md architecture overview updated                  | Done                          |
| AGENTS.md workspace modules list + version              | Done                          |

### Architecture Analysis Artifacts (committed: `eda4116`, `97848d6`)

- Architecture review HTML (strengths, weaknesses, recommendations)
- D2 diagrams: current-state + ideal-state (rendered to SVG)
- Modularization proposal with 15-question brutal self-review
- Execution plan with Pareto breakdown and mermaid graph

### Current Module Landscape (6 modules)

| Module               | go.mod    | External Deps | Tests (race) | Lint     | Coverage |
| -------------------- | --------- | ------------- | ------------ | -------- | -------- |
| root (`errorfamily`) | yes       | zero          | pass         | 0 issues | 98.4%    |
| `diagnose`           | yes (NEW) | zero          | pass         | 0 issues | 77.3%    |
| `agent`              | yes (NEW) | zero          | pass         | 0 issues | 89.4%    |
| `bridge`             | yes       | samber/oops   | pass         | 0 issues | —        |
| `diagnose/git`       | yes       | zero          | pass         | 0 issues | 98.5%    |
| `diagnose/postgres`  | yes       | zero          | pass         | 0 issues | 80.3%    |

---

## b) PARTIALLY DONE

### Documentation for new module structure

- **CHANGELOG:** v0.6.0 entry written ✓
- **SKILL.md:** Architecture overview updated ✓
- **AGENTS.md:** Workspace modules list updated ✓
- **README.md:** NOT updated for module extraction — still says root contains everything
- **DOMAIN_LANGUAGE.md:** Not checked for stale references post-extraction

### Plan items from execution plan

The plan (`docs/planning/2026-06-17_11-51_module-extraction-and-polish.md`) has 17 tasks. Tasks 1-8 (module extraction + CI + version pins) are complete. Tasks 9-17 (API polish + docs) remain.

---

## c) NOT STARTED

| #   | Task                                                   | Impact | Effort |
| --- | ------------------------------------------------------ | ------ | ------ |
| 9   | `Registry.Clone()` method                              | Medium | 20min  |
| 10  | `Registry.RegisterTemplates()` batch                   | Low    | 15min  |
| 11  | DRY `resolveSuggestedFix` / `renderCLI`                | Medium | 30min  |
| 12  | Update README for new module structure                 | Medium | 30min  |
| 13  | Update AGENTS.md build commands per module             | Medium | 20min  |
| 14  | Update SKILL.md details (beyond architecture overview) | Low    | 20min  |
| 16  | Check/update DOMAIN_LANGUAGE.md                        | Low    | 10min  |
| 17  | Final full verification + release prep                 | Low    | 10min  |

---

## d) TOTALLY FUCKED UP

**Nothing.** All 6 modules build, test (race), and lint clean. No stale references. The replace directives in go.mod files are temporary but correct — they'll be removed once root publishes a version that no longer contains agent/ and diagnose/ as sub-packages.

One annoyance: BuildFlow's `go-mod-tidy` step is slow because the replace directives create a chain (root → diagnose → root). This is a known Go limitation during module extraction and resolves on first publish.

---

## e) WHAT WE SHOULD IMPROVE

1. **The replace directive chain is fragile** — root replaces diagnose, diagnose replaces root. This works but is confusing. The real fix is publishing a root version that deletes the `agent/` and `diagnose/` directories from the module. Until then, `go.work` handles development correctly.

2. **`agent/go.mod` doesn't have explicit `diagnose` require** — it has a replace but not a require. `go mod tidy` in the agent module can't resolve diagnose without the replace. This is correct for unpublished modules but needs cleanup on first publish.

3. **README is stale** — still describes the old single-module structure. Needs updating to reflect 6-module workspace.

4. **No Registry.Clone() or RegisterTemplates()** — inconsistency: sentinels have batch registration (`RegisterClassifications`), templates don't. `Clone()` would enable "inherit DefaultRegistry and extend" patterns.

5. **`resolveSuggestedFix` duplicates `renderCLI`'s template chain** — both walk override → registry → default → fallback. Should extract shared helper.

6. **CON 05+04 still deferred** — the `extractCommand` prose parsing + agent renaming. User explicitly deferred this for design discussion.

7. **v1.0 tagging not done** — the root module is ready for v1.0 semver commitment, but this is a product decision requiring explicit user approval.

---

## f) TOP 25 THINGS TO GET DONE NEXT

### High impact, low effort

1. **Update README for 6-module structure** — module landscape, import examples, what's stable vs experimental
2. **Add `Registry.Clone()`** — enables inherit-and-extend patterns
3. **Add `Registry.RegisterTemplates()` batch** — consistency with `RegisterClassifications`
4. **DRY template resolution** — extract shared helper from `resolveSuggestedFix` and `renderCLI`
5. **Update AGENTS.md Quick Start** — per-module build/test commands

### High impact, medium effort

6. **CON 05+04: Design structured `DiagnosticResult` triple** — `{summary, command, rationale}` instead of prose. Kills `extractCommand` heuristic at root.
7. **CON 05+04: Rename "agent" to match what it does** — RCA, Synthesizer, or DiagnosticAnalyzer
8. **Ship v1.0 of the root module** — semver commitment for the classification library
9. **Publish root version that deletes agent/ and diagnose/ dirs** — removes replace directive chain
10. **Add `Error.WithContextMap(map[string]string)`** — batch context without repeated chaining
11. **Fuzz test for `{key}` template substitution** — verify no injection or double-substitution
12. **Add `Family.HTTPStatus()` method** — map families to HTTP status codes (Rejection→400, Conflict→409, Transient→503)

### Medium impact, medium effort

13. **Add slog integration example** — severity from Family
14. **Add HTTP middleware example** — Family → HTTP status code translation
15. **Add `Error.JSON()` method** — structured JSON for API responses
16. **Improve diagnose coverage from 77.3%** — more integration tests
17. **Add retry policy helper** — `Family.RetryPolicy()` returning sensible defaults
18. **Add OpenTelemetry integration example** — span attributes from Family + context
19. **Add `Error.WithContextf(key, format, args...)`** — formatted context values
20. **Add Go doc examples for `NewRegistry`** — discoverable via `go doc`

### Lower priority

21. **Migrate docs/status/ and docs/planning/ to separate directory** — they clutter the library
22. **Add CONTRIBUTING.md section on the Registry pattern**
23. **Add comparison table update to README** — now that Registry is available
24. **Consider `errors.Join` wrapper that pre-classifies** — returns `(error, Family)` tuple
25. **Add benchmark for `Registry.Classify` vs `Classify`** — measure indirection cost

---

## g) TOP QUESTION I CANNOT FIGURE OUT MYSELF

**When should we publish a root version that deletes the `agent/` and `diagnose/` directories?**

The replace directive chain works for development, but it's a hack. The clean fix is:

1. Tag current root as `v1.0.0` (with agent/ and diagnose/ still present as packages)
2. Delete `agent/` and `diagnose/` directories from the root module
3. Tag as `v1.1.0` (breaking: agent/ and diagnose/ now require explicit module imports)
4. Remove all replace directives

But this is a **publishing strategy decision** — it affects every external consumer. I can't determine the right timeline without knowing:

- Are there external consumers today?
- Is v1.0 the right version number, or should it be v0.7.0 first?
- Should we keep backward compatibility (re-export from root) or make a clean break?

---

## Verification Summary

| Module               | Build | Tests (race) | Lint     | Coverage |
| -------------------- | ----- | ------------ | -------- | -------- |
| root (`errorfamily`) | ✓     | ✓            | 0 issues | 98.4%    |
| `diagnose`           | ✓     | ✓            | 0 issues | 77.3%    |
| `agent`              | ✓     | ✓            | 0 issues | 89.4%    |
| `bridge`             | ✓     | ✓            | 0 issues | —        |
| `diagnose/git`       | ✓     | ✓            | 0 issues | 98.5%    |
| `diagnose/postgres`  | ✓     | ✓            | 0 issues | 80.3%    |

**Git log (this session):**

```
a83f311 docs: update CHANGELOG and SKILL.md for module extraction
4ba7843 feat(modularize)!: extract agent/ and diagnose/ into independent modules
f873148 docs(planning): module extraction and library polish execution plan
9cd0d45 style: auto-format architecture docs
97848d6 docs: replace meaningless "core" with honest names
eda4116 docs: architecture review, D2 diagrams, and modularization proposal
9edc7fb feat!: copy-on-write errors, injectable Registry, drop Compose, fix template syntax
```

All changes pushed to `origin/master`.
