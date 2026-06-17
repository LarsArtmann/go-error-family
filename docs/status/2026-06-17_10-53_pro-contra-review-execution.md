# Status Report: Pro/Contra Review Execution Session

**Date:** 2026-06-17 10:53
**Session Goal:** Execute the actionable items from `docs/research/pro-contra-review.html`
**Module:** go-error-family v0.5.0 (root, bridge, diagnose/git, diagnose/postgres)

---

## a) FULLY DONE

### CON 03 — `WithContext`/`WithCause`/`WithTimestamp` copy-on-write (data race fix)

**Problem:** All three chaining methods mutated the receiver in place and returned the same pointer. A shared error (sentinel, struct field) could be mutated by one goroutine while another reads it — a data race. The library preached immutability but contradicted itself.

**Fix:** All three methods now use a shared `clone()` helper that creates a new `*Error` with a deep-copied context map. The original error is never mutated.

**Files changed:** `error.go` (3 methods refactored + `clone()` helper added), `error_test.go` (3 new copy-on-write isolation tests)

**Verified:** Race detector clean. Benchmark shows 0 allocs/op for `WithContext` (escape analysis keeps the clone on the stack for the common case).

### CON 09 — Dropped `Compose` (dead weight)

**Problem:** `Compose(errs...)` was literally `return errors.Join(errs...)` with a doc comment admitting it exists "for API discoverability." It added API surface to learn, document, and maintain for zero added logic.

**Fix:** Removed the function. Updated README to point at `errors.Join` directly. Updated SKILL.md architecture overview and code examples. Removed the AGENTS.md lint configuration entry.

**Files changed:** `classify.go` (removed), `README.md`, `SKILL.md`, `AGENTS.md`

### CON 07 — Template syntax `{key}` instead of `{{.key}}`

**Problem:** `MessageTemplate` used `{{.key}}` placeholders resolved by naive `strings.ReplaceAll`. The syntax collides with Go's `text/template`, misleading users into expecting pipelines, conditionals, and HTML escaping. The AGENTS.md itself flagged this as an XSS concern.

**Fix:** Changed all placeholder syntax from `{{.key}}` to `{key}`. Updated `applyContext` in `handle.go`, all test files, README, SKILL.md, and DOMAIN_LANGUAGE.md.

**Files changed:** `handle.go`, `handle_test.go`, `template_test.go`, `handle_context_test.go`, `README.md`, `SKILL.md`, `docs/DOMAIN_LANGUAGE.md`

### CON 02 — Injectable `Registry` type (global mutable registries eliminated)

**Problem:** Two package-level mutable maps (`registry.entries` for sentinels, `templateRegistry.entries` for templates) created hidden coupling between packages, couldn't be scoped per-test without `t.Cleanup(Unregister...)`, and parallel tests could interfere.

**Fix:** Created `Registry` type (`registry.go`) holding both sentinels and templates. Package-level `DefaultRegistry` preserves backward compatibility — all convenience functions (`Classify`, `RegisterClassification`, `RegisterTemplate`, etc.) delegate to it. `HandleConfig.Registry` allows passing a custom registry. `NewRegistry()` for test isolation (no cleanup needed — registry goes out of scope). Also fixed `HandleErrorDetailedWithConfig` to check registry templates for `SuggestedFix` (was only checking built-in defaults).

**Files changed:** `registry.go` (new, 160 lines), `classify.go` (rewritten to delegate), `handle.go` (threaded registry through, removed `templateRegistry`), `.golangci.yml` (Registry exhaustruct exclusion), `registry_test.go` (new, 9 isolation tests)

**Coverage:** Root coverage rose from 96.5% → 98.4%

### CON 06 — Tightened README retry wording

**Problem:** README said "schedule a retry with backoff" — implying the library provides more than a binary retry/no-retry signal. Real retry logic (backoff, jitter, idempotency, circuit-breaking) is the consumer's responsibility.

**Fix:** Updated README Quick Start and Classification sections. Updated SKILL.md partial-success recipe comment. Now clearly states "backoff, jitter, idempotency are yours to implement."

**Files changed:** `README.md`, `SKILL.md`

### CON 10 — Experimental stability notices + CHANGELOG + docs

**Problem:** Pre-1.0 library with repeated breaking changes across four minor versions. Adopters don't know which parts are stable vs experimental.

**Fix:** Added `Stability: experimental (v0.x)` notices to package docs for `agent`, `diagnose`, `diagnose/git`, `diagnose/postgres`, and `bridge`. Root package documented as the stable classification core. Added comprehensive v0.5.0 CHANGELOG entry with all breaking changes called out.

**Files changed:** `doc.go`, `agent/agent.go`, `diagnose/diagnose.go`, `diagnose/git/doc.go`, `diagnose/postgres/doc.go`, `bridge/bridge.go`, `CHANGELOG.md`, `AGENTS.md` (version → v0.5.0, coverage updated, Registry Pattern section added)

### Cleanup items found during self-review

- Removed stale `.golangci.yml` exclusions for deleted anonymous registry structs (lines for `classify.go` and `handle.go` anonymous exhaustruct)
- Updated SKILL.md Template Resolution Order to reflect `Registry.lookupTemplate` instead of removed `lookupTemplate`
- Updated AGENTS.md Lint Configuration section with Registry-specific notes

---

## b) PARTIALLY DONE

Nothing is partially done — all items below were either fully completed or explicitly deferred.

---

## c) NOT STARTED (explicitly deferred by user)

### CON 05 + CON 04 — `extractCommand` prose parsing + "agent" naming

**User decision:** "This we should think about." This is the one item the user explicitly asked to defer. The fix would make `DiagnosticResult` emit a structured `{summary, command, rationale}` triple instead of prose, and rename the agent to match its deterministic nature (RCA/Synthesizer/DiagnosticAnalyzer).

**Why deferred:** This is a deeper architectural change that affects the diagnostic rule interface, all existing rules (FilesystemRule, NetworkRule, GitRule, PostgresRule), the agent, and all their tests. It deserves its own design discussion.

---

## d) TOTALLY FUCKED UP

Nothing. All changes are verified: build ✓, tests ✓ (race), lint 0 issues across all 4 modules.

---

## e) WHAT WE SHOULD IMPROVE

1. **`WithContext` now allocates** — The benchmark shows 0 allocs/op (escape analysis is smart), but the clone creates a new `*Error` and copies the map. For hot paths doing many `WithContext` calls, this is slightly more expensive. The tradeoff is correct (safety > micro-optimization), but worth documenting.

2. **Registry doesn't have a `Clone()` method** — If a service wants to start from `DefaultRegistry` and add a few sentinels, it can't. It must re-register everything. A `Clone()` method would enable "inherit and extend" patterns.

3. **No `Registry.RegisterTemplates(map[string]MessageTemplate)` batch method** — Sentinels have batch registration, templates don't. Inconsistency.

4. **`resolveSuggestedFix` duplicates the resolution chain** — It manually walks override → registry → default → fallback, which is the same chain as `renderCLI`. Could extract a shared `resolveTemplate(code, cfg, reg)` function.

5. **The `{key}` syntax has no escaping** — If a context value contains `{something}`, it could be interpreted as a placeholder in a subsequent substitution. Unlikely but theoretically possible. A single-pass replacement (not iterative) mitigates this.

---

## f) TOP 25 THINGS TO GET DONE NEXT

### High impact, low effort

1. **CON 05 + CON 04:** Design structured `DiagnosticResult` triple + rename agent (the deferred item)
2. **Add `Registry.Clone()`** — enables inherit-and-extend patterns
3. **Add `Registry.RegisterTemplates()` batch method** — consistency with `RegisterClassifications`
4. **Extract shared template resolution** — DRY `resolveSuggestedFix` and `renderCLI`
5. **Ship v1.0 of the classification core** — the stable root package is ready; mark it v1.0.0 to signal commitment

### High impact, medium effort

6. **Add `Registry.Merge(other *Registry)`** — compose registries (e.g., service-specific + shared)
7. **Add context-value escaping** — single-pass replacement or use a delimiter that can't appear in keys
8. **Add `Error.WithContextMap(map[string]string)`** — batch context addition without repeated chaining
9. **Fuzz test the new `{key}` template substitution** — verify no injection or double-substitution
10. **Add `Error.Clone()` as a public method** — consumers may want to branch an error
11. **Document the `Registry` pattern in SKILL.md with a full integration example**
12. **Add `HandleConfig.Validate()` method** — catch nil writers, conflicting options early

### Medium impact, medium effort

13. **Add structured logging integration example** — slog handler that uses Family for severity
14. **Add HTTP middleware example** — translate Family to HTTP status codes
15. **Add `Family.HTTPStatus()` method** — map families to HTTP status codes (Rejection→400, Conflict→409, Transient→503, etc.)
16. **Improve diagnose core coverage** — currently 77.3%, could reach 90%+ with more integration tests
17. **Add `Error.JSON()` method** — structured JSON representation for API responses
18. **Add retry policy helper** — `Family.RetryPolicy()` returning a sensible default policy (max attempts, backoff)
19. **Add OpenTelemetry integration example** — span attributes from Family and context
20. **Add `Error.WithContextf(key, format, args...)`** — formatted context values

### Lower priority

21. **Migrate docs/status/ and docs/planning/ to a separate repo or wiki** — they clutter the library
22. **Add a CONTRIBUTING.md section on the Registry pattern**
23. **Add Go doc examples for `NewRegistry` and `Registry.Classify`**
24. **Add a comparison table update to README** — now that Registry is available
25. **Consider `errors.Join` wrapper that pre-classifies** — returns `(error, Family)` tuple

---

## g) TOP QUESTION I CANNOT FIGURE OUT MYSELF

**Should the v1.0 stable core include the `Registry` type, or should it stay v0.x experimental?**

The `Registry` is new in this session. It's well-tested (9 isolation tests, race-clean) and backward-compatible. But it's brand new — no external consumers have validated the API shape. Options:

- **A:** Include `Registry` in v1.0 core — it's the right abstraction and backward-compatible
- **B:** Keep `Registry` as v0.x experimental, freeze only `Family`/`Classify`/`Error`/interfaces as v1.0
- **C:** Ship v1.0-rc1 with `Registry` included, gather feedback, then finalize

I lean toward **A** because the package-level API is unchanged (all convenience functions still work identically), and `Registry` is additive. But this is a product/strategy decision, not a technical one.

---

## Verification Summary

| Module               | Build | Tests (race) | Lint     | Coverage |
| -------------------- | ----- | ------------ | -------- | -------- |
| root (`errorfamily`) | ✓     | ✓            | 0 issues | 98.4%    |
| `agent`              | ✓     | ✓            | 0 issues | 89.4%    |
| `diagnose` (core)    | ✓     | ✓            | 0 issues | 77.3%    |
| `diagnose/git`       | ✓     | ✓            | 0 issues | 98.5%    |
| `diagnose/postgres`  | ✓     | ✓            | 0 issues | 80.3%    |
| `bridge`             | ✓     | ✓            | 0 issues | —        |

All benchmarks within expected ranges. `WithContext` at 0 allocs/op (escape analysis).
