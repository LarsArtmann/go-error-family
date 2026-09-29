# Peripheral Surface Audit — M14 Hygiene Batch

**Date:** 2026-09-28
**Scope:** docs/ surfaces outside the living-doc set and docs/status/ — verified current
against v0.10.3 so nothing stale hides outside the annotated/archived spine.

| Surface | Verdict | Evidence |
| ------- | ------- | -------- |
| `docs/modularization/PROPOSAL.md` + `EXECUTION_PLAN.md` (2026-06-17, v0.5.0 era) | HISTORICAL — fully executed | the 7-module go.work workspace IS the shipped outcome (root, agent, bridge, diagnose, diagnose/git, diagnose/postgres, examples); no open task in the plan remains unimplemented |
| `docs/research/pro-contra-review.html` | REFERENCE — keep | decision research artifact, no claims requiring freshness |
| `docs/comparison-samber-oops.html` | REFERENCE — keep | comparative analysis backing the bridge design; complementarity is restated in AGENTS.md |
| `docs/top-5-stupidest-things.md` + `docs/resolving-top-5-stupidest-things.md` | RESOLVED — keep as pair | `ApplyFixes` deleted (agent is analysis-only per AGENTS.md); remaining items all shipped (verified in the 2026-09-27 docs-health pass) |
| `docs/feedback/sec-consumer-feedback.md` | ARCHIVED 2026-09-28 | ~~moved to `docs/feedback/archived/` with a verified-resolution banner~~ banner-only at archive time; inline per-item resolutions (PP1-PP5, IDEA1-4) added 2026-09-29 by the docs-health completeness gate — all shipped by v0.10.1 |
| `docs/planning/archived/best-of-both-worlds.html` | ARCHIVED — correctly placed | zero dangling references (repo-wide grep hits only the Pareto plan's checklist item) |
| `go.work` | ANNOTATED 2026-09-28 | floor-vs-toolchain comment added (1.26.7 = toolchain line; per-module floors are deliberate) |

**Result:** no stale, unresolved, or dangling document outside the living-doc spine.
