# Status Report: art-dupl Deep-Sweep Triage (Session-Scoped)

**Date:** 2026-09-30 06:08 CEST
**Scope:** This session only, per instruction: the art-dupl `-t 1 --suggest-generics` deep-sweep triage plus the verification run during this report. No unrelated research was done.
**Skill compliance note:** `status-report` prescribes a styled HTML dashboard; the user explicitly requested `.md` at this path, so Markdown wins (override flagged in the closing message).

**TL;DR:** The deep sweep surfaced exactly the 3 accepted baseline clone groups — zero harmful duplication, zero new clones. The plain-mode gate is green (`art-dupl check -t 1 .` → exit 0). During report verification I additionally **proved the gate's failure path** (planted clone → exit 2) and **mapped a flag-surface gap** (`check` rejects `--suggest-generics`; the baseline records no flag provenance). No code was changed; nothing is broken by this session.

---

## Self-Review (brutal, session-scoped)

1. **What did you forget?**
   - I matched report→baseline by **file pair**, not by **hash**. The tool matches by hash; my match was backstopped by the hash-based gate (green), but the direct hash comparison (scan output vs `.art-dupl-baseline.json`) was never run programmatically. Closed as item (b)1 below / task f.2.
   - I saw the gopls warning `error_test.go:567:44 [nilness][nilpanic] panic with nil value` repeatedly in diagnostics during the session and did not mention it in my first handoff. It is now named in (d)1 and (f)5.
2. **What is stupid that we do anyway?**
   - `.art-dupl-baseline.json` records `threshold` + `recordedAt` but **not the flags used at recording time**, while scan and `check` accept different flag surfaces. Flag-sensitive hashing with no provenance is a silent-drift trap.
   - The user's deep sweep passed `--suggest-generics --type-aware` together; the tool itself warns the first overrides the second. Redundant flag noise on every deep sweep.
3. **What could you have done better?**
   - First read of `check --suggest-generics` output mislabeled an **exit 1 usage error** ("Unknown flag") as a validation failure. Caught on rerun by separating exit codes from greps; corrected before it reached any conclusion.
   - `PIPESTATUS` was empty under the shell interpreter on the first exit-code capture; should have captured exit codes cleanly the first time.
4. **What could you still improve?**
   - Make "prove a gate both ways" standard: the 30-second negative-path test materially raised confidence in the green claim.
   - Convert the two non-`strTrue` accepted-clone rationales into durable docs; they currently live only in conversation transcripts and this report.
5. **Did you lie to you?** No. One early statement was wrong and was corrected in-session (the "validation error" interpretation). The green claim was verified twice.
6. **How can we be less stupid?** Pin canonical command lines and exit-code semantics in AGENTS.md (task f.1, f.4); add flags provenance to the baseline (f.3).
7. **Ghost systems?** None created; nothing was wired or unwired this session (zero code changes).
8. **Scope creep?** Held the line: did NOT "fix" the accepted `strTrue` clone (AGENTS.md: do not fix again), did NOT touch the unrelated gopls warning beyond reporting it, did NOT run HARVEST mid-session (user said WAIT).
9. **Removed something useful?** No removals of any kind.
10. **Split brains?** One noticed, pre-existing: accepted-clone rationale lives in **two places that can drift** — the AGENTS.md bullet (documents only `strTrue`) and the baseline JSON (hashes only, no rationale). Task f.1 unifies.
11. **Tests?** No code changed → no test suite run (per "test after changes"). Instead: a tooling test — negative-path gate check in `/tmp` (planted clone → exit 2, then `trash` cleanup). Repo suite state is the release state (green, per AGENTS.md 2026-09-29).

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | Deep-sweep triage: all 3 reported clone groups identified as the accepted baseline groups | `.art-dupl-baseline.json` (3 entries, hash `29b9…`/`c4cf…`/`f5aa…`) ↔ report file pairs, all 6 code locations read: `error.go:117-141`, `bridge/bridge.go:224-263`, `diagnose/rules_network.go:31-40`, `diagnose/postgres/rules_postgres.go:58-67`, `diagnose/git/rules_git.go:203-206`, `diagnose/helpers.go:101-108` |
| 2 | Per-clone dispositions with rationale (Accept ×3) | Format methods: parallel `fmt.Formatter` for distinct types across modules, verbose branches differ, generic extraction needs a dependency root can't have. `resolveHost/Port` pair: 2-line idiomatic setup, cross-module extraction costs more than it saves. `strTrue/strFalse`: AGENTS.md disposition (unexported, can't cross module boundary) |
| 3 | Plain-mode gate verified green | `art-dupl check -t 1 .` → "✅ No new clones detected (baseline: 3 groups)", exit 0 |
| 4 | Gate failure path proven (not assumed) | Planted duplicate funcs in `/tmp/artdupl-gate-test` → "🔴 1 new clone group(s) detected (baseline had 3)", exit 2; dir removed via `trash` |
| 5 | Flag-surface boundary mapped | `art-dupl check -t 1 --suggest-generics .` → exit 1, "Unknown flag: --suggest-generics" — `check` runs plain-mode only |
| 6 | Skills loaded and followed | `deduplicate-code` (triage turn); `status-report` + `brutal-self-review` + `docs-health` + section-quality-guide (this turn) |

## b) PARTIALLY DONE

| # | Item | Works | Open | Effort |
|---|------|-------|------|--------|
| 1 | Hash-level baseline correspondence | File pairs + current line numbers verified against code | Scan-vs-baseline **hash** diff not run mechanically (paste carried no hashes; rerun with extraction pending) | S |
| 2 | Durable disposition docs | AGENTS.md covers `strTrue` rationale + "3 accepted groups" | Rationales for clones 1–2 exist only in transcripts/this report | S |
| 3 | Baseline flags provenance | Problem precisely scoped: `check` can't take the flag at all; baseline JSON has no `flags` field | Fix undecided: AGENTS.md doc note (S) vs upstream feature request (M/L, verify-before-filing gate first) | S–M |

## c) NOT STARTED

| # | Item | Why not started | Priority |
|---|------|-----------------|----------|
| 1 | Upstream candidate: baseline `flags` field + `check` flag parity (art-dupl) | Out of session scope; needs repro packaging + verify-before-filing | Medium |
| 2 | gopls `[nilness][nilpanic]` triage at `error_test.go:567:44` | File outside session scope (user: no unrelated research); warning is diagnostic-level only | Low-Medium |
| 3 | HARVEST of section (f) into `TODO_LIST.md`/`ROADMAP.md` | Deliberately deferred: user said "THEN WAIT FOR INSTRUCTIONS" | Next-session first step |

## d) TOTALLY FUCKED UP

**Nothing from this session is broken.** Zero code changes, no damage, gate green both ways (0 when clean, 2 when a clone is planted). The closest things, named precisely rather than hidden:

1. **Pre-existing, noticed but not triaged:** gopls `nilness` warning `error_test.go:567:44` — "panic with nil value". Severity unknown (diagnostic warning; CI green per AGENTS.md release state). Root cause: uninvestigated. Workaround: n/a until triaged. It may be a deliberate nil-panic test — that is exactly what task (f)5 resolves.
2. **Tooling sharp edge (workflow risk, not breakage):** baseline flag-provenance gap. A future baseline recorded in suggest-generics mode could not be checked at all (flag unsupported), and nothing in the JSON would reveal the mismatch. No data loss; confusion cost only.

## e) WHAT WE SHOULD IMPROVE

1. **Pin canonical commands** in the AGENTS.md art-dupl bullet: gate = `art-dupl check -t 1 .`; deep sweep = `art-dupl --sort total-tokens -t 1 --suggest-generics --timing --rich-text` (drop `--type-aware`; overridden, tool warns each run). Impact: stops per-session flag roulette. 
2. **One-line rationale per accepted group** next to the baseline reference in AGENTS.md. Impact: dispositions survive sessions; kills the AGENTS/JSON split brain.
3. **Baseline `flags` provenance field** (doc note now, upstream later). Impact: makes hash mismatch detectable instead of mysterious.
4. **Match by hash, not by eye.** Rerun the scan, extract hashes, diff against the baseline JSON. Impact: turns eyeball triage into a mechanical check.
5. **Prove gates both ways.** Negative-path test before trusting any green. Impact: ~30 s, converts "gate passed" into "gate works".
6. **Document exit-code semantics** (0 = green, 1 = usage error, 2 = validation failure) in AGENTS.md or upstream README. Impact: this session nearly misread 1 as 2.

## f) Next Tasks (30 real items; the 50 cap would force 20 filler rows)

HARVEST input — route bounded items to `TODO_LIST.md`, vague/long-term to `ROADMAP.md`. `[dup]` = this session's thread; `[noticed]` = pre-existing, from session context (no new research).

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Extend AGENTS.md art-dupl bullet: per-group rationales (clones 1–2) + canonical gate/sweep commands `[dup]` | High | S | Documentation |
| 2 | Rerun deep sweep, extract group hashes, diff vs baseline JSON mechanically `[dup]` | Medium | S | Quality |
| 3 | Decide flags-provenance fix: AGENTS.md note now; upstream `flags` field after verify-before-filing `[dup]` | Medium | S–M | Quality |
| 4 | Record exit-code semantics (0/1/2) in AGENTS.md art-dupl bullet `[dup]` | Low | S | Documentation |
| 5 | Triage gopls `[nilness][nilpanic]` at `error_test.go:567:44`: deliberate or real? `[noticed]` | Medium | S | Bug |
| 6 | Find where the art-dupl gates are enforced (CI/BuildFlow/pre-commit); if nowhere, wire `art-dupl check -t 5 .` into the routine gate `[dup]` | High | S | Quality |
| 7 | Add "considered and rejected: generic Format extraction / cross-module resolveHost helper" line to AGENTS.md so future agents don't re-litigate `[dup]` | Low | S | Documentation |
| 8 | Verify `art-dupl check` respects generated/test exclusion patterns same as scan `[dup]` | Low | S | Quality |
| 9 | Spot-audit the `-t 1` "221 non-actionable + 96 suppressed" classes once — confirm nothing actionable hides there `[dup]` | Medium | M | Quality |
| 10 | Set re-baseline trigger: after any cross-module refactor, re-record deliberately (never silently) `[dup]` | Low | S | Quality |
| 11 | Ship next release: README Discussion-#12-withdrawal fix kills the dead pkg.go.dev link (TODO_LIST #2) `[noticed]` | High | M | Release |
| 12 | Post-release: ride Dependabot root-pin PRs (submodules pinned previous root at tag time) `[noticed]` | Medium | S | Release |
| 13 | pnpm-audit BuildFlow skip → upstream with subdirectory-lockfile repro `[noticed]` | Medium | M | Upstream |
| 14 | branching-flow `IsIgnored` → upstream `pkg/phantom` repro `[noticed]` | Medium | M | Upstream |
| 15 | go-structure-linter: run CLI directly (embedded snapshot ignores project config) and diff against `.go-structure-linter.yaml` intent `[noticed]` | Low | M | Quality |
| 16 | Re-run erraudit battery after next toolchain bump (last green 2026-09-29) `[noticed]` | Medium | S | Quality |
| 17 | Keep golangci-lint v2.13.2 pin in sync (ci.yml, release.yml, local) at next upgrade `[noticed]` | Low | S | Quality |
| 18 | Website: verify no `package.json`-without-lockfile drift exists right now (standing rule) `[noticed]` | Low | S | Quality |
| 19 | Website: keep untracked `bun.lock` uncommitted (standing canary) `[noticed]` | Low | S | Quality |
| 20 | Website: confirm `website-check.yml` ran green on the latest `website/**` push `[noticed]` | Low | S | Quality |
| 21 | Bridge adoption (zero external consumers): write one reference-implementation showcase post/README section `[noticed]` | Medium | L | Feature |
| 22 | Under-adopted APIs (LogError ~3, HTTPHandler ~5, errorfamilytest ~3): add usage examples to README `[noticed]` | Medium | M | Documentation |
| 23 | Define a fuzzing cadence for the 16 fuzz targets (not just ad-hoc) `[noticed]` | Medium | M | Quality |
| 24 | Re-verify jsonv2 guard trio (CI `GOWORK=off` build + depguard canary + `GOEXPERIMENT=` reflex) `[noticed]` | Medium | S | Quality |
| 25 | Re-run md-go-validator after any doc edit (114/114 baseline; mind system-binary lag gotcha) `[noticed]` | Low | S | Quality |
| 26 | Re-sweep FEATURES.md coverage table (7 modules ≥ 89% `-race -cover`) after next significant change `[noticed]` | Low | M | Quality |
| 27 | Ask upstream whether `check` should accept `--type-aware` for hash-compatible deep sweeps (same verify-before-filing gate as #3) `[dup]` | Low | M | Upstream |
| 28 | If #6 wires the gate into CI, add the negative-path check as a fixture instead of ad-hoc `/tmp` tests `[dup]` | Low | S | Quality |
| 29 | After next release tag, confirm baseline still green against the new tree (recordedAt is 2026-09-28) `[dup]` | Low | S | Quality |
| 30 | Turn the copywriting rule "announcements never enter CHANGELOG" into a release-checklist line if not already there `[noticed]` | Low | S | Documentation |

## g) Questions I Cannot Answer Myself

1. **Baseline intent:** Should future deep sweeps stay hash-comparable with the baseline (i.e., always run the sweep WITHOUT `--suggest-generics`, pinning plain-mode hashes), or is the baseline only ever consumed by the plain gate? I tried: reading the baseline JSON (no `flags` field), running `check` with the flag (rejected, exit 1). The answer decides between "pin commands in AGENTS.md" and "push flags-provenance upstream".
2. **Durable dispositions:** Do you want the two non-`strTrue` accepted-clone rationales written into AGENTS.md (one line each — my recommendation), or is the current "3 accepted clone groups" bullet sufficient?
3. **Next-session priority:** HARVEST section (f) into `TODO_LIST.md`/`ROADMAP.md` first, jump straight to a specific thread (e.g., #6 gate enforcement), or triage the gopls warning?

---

## Evidence Appendix

```
$ art-dupl check -t 1 .
Found total 0 clone groups.
✅ No new clones detected (baseline: 3 groups).        exit=0

$ cd /tmp/artdupl-gate-test  # planted duplicate func pair + copied baseline
🔴 1 new clone group(s) detected (baseline had 3). Run `art-dupl baseline` to update.
  Validation error: 1 new clone group(s) introduced.   exit=2
$ trash /tmp/artdupl-gate-test

$ art-dupl check -t 1 --suggest-generics .
  ERROR
  Unknown flag: --suggest-generics.                    exit=1

Baseline (.art-dupl-baseline.json, recordedAt 2026-09-28T20:34:22Z, threshold 1):
  29b9879a4345c98f  diagnose/rules_network.go + diagnose/postgres/rules_postgres.go
  c4cf23c2580ecbf2  error.go + bridge/bridge.go
  f5aa721ca6b28b15  diagnose/git/rules_git.go + diagnose/helpers.go
```

*End of report. Section (f) is HARVEST-ready; HARVEST intentionally deferred per "WAIT FOR INSTRUCTIONS".*
