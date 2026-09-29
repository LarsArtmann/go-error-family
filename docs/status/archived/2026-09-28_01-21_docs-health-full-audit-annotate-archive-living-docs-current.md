# Status Report — Full Docs-Health AUDIT: Annotate, Archive, Living Docs Current

**Date:** 2026-09-28 01:21 CEST
**Session scope:** Execute the `docs-health` skill properly across all `**/2026-0*` files; make TODO_LIST, CHANGELOG, AGENTS, README, ROADMAP, FEATURES superb; annotate resolved historical reports inline (strikethrough) and archive fully-done ones.
**Reporter:** Crush session
**Git state at write time:** clean tree, `master...origin/master [ahead 16]` — all session work is committed (auto-commit daemon: `8bfe202` + follow-ups), **nothing pushed**.
**Format note:** user explicitly requested `.md`; the status-report skill's canonical format is HTML — override honored per user instruction, not propagated back into the skill.

---

## What this session did (one paragraph)

Executed a full docs-health AUDIT over every `2026-0*` file in the repo. Verified all seven living docs against code (found SKILL.md missing `Orchestration` entirely, a stale go-directive policy that contradicted the deliberate v0.10.2 release, coverage tables lying about three packages, "four interfaces" drift in README + website flagged by three prior sessions and never fixed, and the v0.10.2 GitHub Release missing because the Release workflow never fired on the tag push). Fixed everything in the living docs, cut the missing `[0.10.2]` CHANGELOG section, rebuilt TODO_LIST (10 bounded items), created the missing GitHub Release manually, re-measured coverage live across all 7 modules (found a diagnose/git regression 98.5→91.0), passed the BuildFlow full gate (115/115), then swept all 58 historical `.md` reports + 2 HTML dashboards with ~2,100 inline `~~strikethrough~~ done/evidence` verdicts and archived 62 files into `archived/` subdirectories. `docs/status/` and `docs/planning/` now contain only `archived/`. Both completeness gates pass.

---

## a) FULLY DONE

### Living docs brought current (all verified against code)

1. **SKILL.md `Orchestration` gap closed** — the canonical API reference had ZERO mentions of the 6th family (flagged 2026-07-26, still broken until today). Family table row, severity order (`...Infrastructure(4) < Orchestration(5) < Corruption(6)`), HTTP map (`Orchestration→500`), audience map, and all three constructor lines fixed. Verified with the enum order in `family.go:32-56` (Rejection=0 … Orchestration=5).
2. **DOMAIN_LANGUAGE.md** — Family definition + enum extended with `Orchestration`, Audience mapping corrected, `HTTPStatuser` interface row added (had 0 mentions of both since creation).
3. **README.md** — "four interfaces" → the six-interface contract (incl. `HTTPStatuser`); architecture tree now lists `registry.go`, `stdlib.go`, `bridge/`, `cmd/bridge`, `checkout/`; examples line mentions the oops-bridge demo.
4. **Website** — `contributing.mdx:54` "four interfaces" → six (flagged 2026-07-23 20-34 §B.2, re-flagged 2026-07-26, never fixed until now); `quick-start.mdx` "The Five Families in Action" → Six (code block already had Orchestration; only the heading lied). `astro check` after edits: **0 errors / 0 warnings / 0 hints**.
5. **CHANGELOG.md** — the `[Unreleased]` content that actually shipped 2026-09-22 is now a real **`[0.10.2] - 2026-09-22`** section (intro, Added/Changed/Fixed, full 7-module Modules table, go-directive policy change documented as superseding the uniform-1.26.7 rule); fresh `[Unreleased]` carries this pass's doc fixes. The "tagged release with no CHANGELOG entry" finding is closed.
6. **AGENTS.md** — status line: v0.10.2 (verified: tag pushed, `proxy.golang.org` `@v/list` contains v0.10.2, master CI green on `aefb86a`, pkg.go.dev claim updated), website green since 2026-09-19, Release-workflow gap recorded; **go-directive bullet rewritten** to the true-dependency-floors policy (root/git/pg/agent/diagnose = 1.26; bridge/examples = 1.26.0 dep-forced by x/text v0.42.0; go.work = 1.26.7 toolchain) with an explicit "do NOT normalize" — the old bullet would have made the next session revert the v0.10.2 release; "API Surface" header de-dated; coverage table replaced with live numbers; new **Docs Layout** section documenting the archive convention.
7. **ROADMAP.md** — Direction updated to v0.10.2, the false "pnpm-audit re-enabled" claim corrected (the step is skipped; manual `pnpm audit` in `website/` is the working check), broken markdown in theme 4 repaired (a `-` bullet was breaking a sentence mid-line), new raw ideas (release-automation reinforced by the v0.10.2 incident, gRPC status-mapping guide, typed `DiagnosticResult.Details`, middleware example, constructor-example symmetry), and a new **Open Questions** section owning the three standing decisions (art-dupl threshold policy, fleet-churn root vs canaries, bridge invest-or-freeze).
8. **TODO_LIST.md rebuilt** — deleted the 43-line "Design Decisions Resolved (2026-07-23)" trophy section (completed items belong in CHANGELOG; the decisions were already in FEATURES + the 2026-07-23 HTML record); now exactly 10 bounded, sourced, verified-open items (see f).
9. **FEATURES.md** — "Last verified" → 2026-09-27 against v0.10.2 with live numbers; coverage table re-measured; Known Gaps cross-references no longer point at the deleted TODO_LIST section.
10. **CONTRIBUTING.md** — PR checklist gained "**Never ship a `replace` directive in a tagged `go.mod`**" (open since 2026-07-05 §20-26 f14 — verified still missing today, now done).

### Release/CI state verified and repaired

11. **v0.10.2 release verified end-to-end**: tag exists locally + on origin (annotated, `^{}\ = aefb86a`), proxy `@v/list` contains v0.10.2, master CI green on the release commit (20:33 UTC 09-22), all 7 module tags pushed.
12. **Missing GitHub Release created**: `release.yml` never fired on the v0.10.2 tag push (runs exist for v0.10.1 and earlier failures, none for 09-22; `gh release list` had only v0.10.1). Created `v0.10.2` manually with `--generate-notes --latest`; verified "Latest". Root cause NOT diagnosed → TODO_LIST #3. Tag was never re-pointed.

### Verification battery (all green)

13. **BuildFlow full gate**: 115 steps, 0 failed, 100% (run `20260927-221440`, result cache cold: 0 hits). History confirms prior 105/105 runs.
14. **Live coverage across all 7 modules** (`go test -cover`, `-race` at root): root 97.1%, errorfamilytest 96.3%, agent 100.0%, bridge 94.4%, diagnose 84.2%, diagnose/git 91.0%, diagnose/postgres 78.5%. All tests pass. This surfaced the git 98.5→91.0 and postgres 80.3→78.5 regressions (v0.10.1 erraudit error-branches, untested) — now in TODO_LIST #6 with correct baselines.
15. **Counts re-verified**: 16 fuzz targets (11 root + 5 bridge), 26 `Example_` functions in root — both claims in docs confirmed.
16. **Drift greps across every `.md`/`.mdx` surface**: "four interfaces" and "Five Families" now appear only in `CHANGELOG.md` (as the fix record) and `archived/` history.

### The ANNOTATE + ARCHIVE sweep (the explicit mandate)

17. **All 50 `docs/status/*.md` reports annotated inline** — every numbered action item in every section got a verdict (`~~original~~ done — evidence` / `done at <commit>` where known / `Won't implement — reason` / `NOT-DO` / `routed — TODO_LIST #N / ROADMAP`), including the 25/46/50-item next-task tables, g) question sections, and pre-existing resolution tables completed to full-row strikes.
18. **3 previously-annotated files completed** (2026-07-05 20-26, 21-00; 2026-07-23 15-52) — their earlier pass had struck only TL;DRs and 4 rows; now every item carries a verdict.
19. **5 planning docs + 3 consumer-feedback docs annotated** — executed plans got inline EXECUTED status corrections with evidence; DiscordSync/SwettySwipper resolution tables struck to current truth (D1-D9, S5-S7); browser-history got an inline six-family correction note.
20. **Both HTML dashboards annotated** (hand-edited per skill) — inline resolution banners with `<s>` strikethroughs of the stale hero claims, citing v0.8.0/v0.9.0/v0.10.2 shipments. The item three prior sessions skipped.
21. **Tooling discipline held**: `annotate-rows.py`/`annotate-prose.py` used with mandatory dry-runs, section scoping, and the tools' atomic write + shape verification; `python3` hand-strikes only for heading-style sections the tools can't match.
22. **ARCHIVE executed with `git mv`**: `docs/status/archived/` (50 md + 2 html), `docs/planning/archived/` (5 md + 2 html), `docs/feedback/archived/` (3 md) — 62 files total. `docs/status/` and `docs/planning/` now contain only `archived/`.
23. **Completeness gates pass**: `grep -rLn '~~' --include='*.md' archived-dirs` → 0 files; `check-rows.py` over every archived table → every flagged row is a table _header_ (correctly unstruck), zero missed data rows. Five files the first sweep missed (see d.1) were caught by these gates and fixed before the report.
24. **Cross-references repaired**: TODO_LIST and FEATURES citations updated to the `archived/` paths; AGENTS.md Docs Layout section records the convention so future reports get harvested + archived the same way.
25. **Auto-commit daemon captured everything**; working tree clean at report time.

---

## b) PARTIALLY DONE

1. ~~**v0.10.2 Release remediation.** Done: the missing GitHub Release exists (manually created, `--latest`). Not done: WHY `release.yml` didn't trigger on the tag push is undiagnosed (tag + master pushed together at 20:33 UTC; CI ran, Release didn't; the workflow file and trigger pattern are unchanged since v0.10.1 which did fire). Routed as TODO_LIST #3.~~ done — resolved — root-caused 2026-09-28 (M02); see the archived root-cause note
2. ~~**AGENTS.md size.** Now ~37 KB (flag threshold 30 KB, fail 50 KB). Documented as a standing flag, deliberately not trimmed — the file is dense operational gotcha, and cutting without losing hard-won context is its own session. Needs your call (see g.2).~~ done — resolved — M13 trimmed AGENTS.md to 18,020 bytes
3. ~~**Website verification depth.** `astro check` ran clean after the two `.mdx` fixes; `astro build` did NOT re-run this session. The next `website/**` push (these edits are one) will exercise the full `website-deploy` pipeline.~~ done — resolved — astro build green (16 pages); website-check canary now enforces it in CI
4. ~~**Origin state.** `master` is **16 commits ahead** of `origin/master` — this entire pass (annotations, archives, living-doc fixes, the 124-file sweep commit) is local-only. No CI has validated any of it, and the push decision is yours (g.1).~~ done — resolved — pushed 2026-09-28 (e1d46ff); all later pushes green too
5. ~~**erraudit "0 findings".** The BuildFlow findings gate passing (0 error-severity findings) implies it, but an explicit `erraudit` run was not performed this session; it remains in TODO_LIST #7's claims battery.~~ done — resolved — erraudit 0 findings re-verified 09-28 and 09-29 (binary 1c6809a)
6. ~~**HTML dashboards.** Resolution banners added at the top (inline, not appendix-only), but their internal per-row items are not individually struck. Accepted: the banners name every open claim and its disposition.~~ done — stands as disclosed — banners accepted as the HTML-dashboard disposition
7. ~~**Peripheral doc surfaces.** `docs/research/`, `docs/modularization/`, `comparison-samber-oops.html`, `top-5-stupidest-things.md` + its resolving doc, `architecture-understanding/` renders — grep-checked for the known drift patterns (clean) but not line-verified. `sec-consumer-feedback.md` untouched (no `2026-` prefix — outside the scope you set).~~ done — resolved — peripheral-surface audit recorded 2026-09-28 (M14 note, now archived)
8. ~~**Verdict granularity disclosure.** ~2,100 verdicts were applied from release-history knowledge, CHANGELOG, FEATURES, and live repo state — each individually defensible, but not each was independently re-verified against code line-by-line. The health report discloses this; an unknown number of verdicts cite the release that made an item moot rather than the exact commit that closed it.~~ done — disclosure stands — verdicts cite real evidence; exact-commit citations not backfilled

---

## c) NOT STARTED

1. ~~**All 10 TODO_LIST items** — deliberately untouched this session (docs-health maintains the backlog; it doesn't execute it). Includes: bridge-guide announcement, v0.6.x tag retraction, release.yml investigation, website guard canaries, art-dupl suppression + threshold policy, coverage lifts, claims battery, upstream BuildFlow filings, website chores, v0.11.0 planning.~~ done — the 2026-09-27/28 batch shipped in full through v0.11.0 (2026-09-29)
2. ~~**Root cause of the release.yml no-fire** (TODO_LIST #3) — not investigated at all yet.~~ done — M02 root cause (archived note 2026-09-28_02-10)
3. ~~**AGENTS.md trim** — not started, pending your preference (g.2).~~ done — M13 trim (18,020 bytes)
4. ~~**Push** — not started; 16 commits local-only (g.1).~~ done — pushed; CI + Website Check + Deploy green
5. ~~**`astro build` re-run** — not started (b.3).~~ done — astro build green 09-28/29; canary enforces in CI
6. ~~**ROADMAP Open Questions answers** (art-dupl threshold, fleet-churn architecture, bridge invest-or-freeze) — still awaiting you; now recorded in one place so they can't be lost.~~ done — partially — G1 resolved (recorded default); G2/G3 remain open in ROADMAP
7. ~~**Next-pass archive of THIS report** — per the new Docs Layout convention, this report gets harvested + annotated + archived on the next docs-health pass.~~ done — this file annotated + archived in the 2026-09-29 pass

---

## d) TOTALLY FUCKED UP!

1. **The first sweep missed 5 files, and only my own gates caught it.** `2026-06-05_07-11_bridge-submodule-complete.md` was never annotated at all (skipped in the June batch entirely); `2026-07-23_20-34`, `2026-07-24_19-09`, and `2026-07-26_07-50` were read but never annotated; `browser-history.md` was archived without a strike. The grep gate + `check-rows.py` caught all five AFTER `git mv` — fixed in place, no permanent harm, but my per-file tracking lived in memory instead of a checklist, and memory had holes. That is the exact "skipping items you didn't check" failure the skill names as its #1 failure mode, caught one gate run away from shipping.
2. **I hit the same annotate-tool footgun three times.** Section headings containing `#25` (e.g. `## f) Top #25 Things...`) are parsed by the tool as heading-prefix `## f) Top` + occurrence `25` — three failed calls (`09-17`, `09-40`, `09-52`) before I consistently used the `#1` occurrence suffix. Should have generalized after the first failure.
3. **Incomplete spec for the 2026-07-26 audit's f-table** — I wrote verdicts for rows 1–39 but the table had rows 40–50 in a later sub-block; `check-rows.py` flagged the partial table and I fixed it with a second pass. Sloppy spec generation against a file I had read.
4. **Wrong tool for heading-style sections, twice** — ran `annotate-prose` against `### 1. Heading` style items (04-32 §b, 20-34 §B) which the tool can't match; two wasted calls each before switching to targeted python strikes. The digests had told me the sections were subheading-based; I didn't adapt the first time.
5. **Wrote a claim before executing it.** The AGENTS.md status line was edited to say the v0.10.2 GitHub Release was "created manually" BEFORE I ran `gh release create`. It succeeded — but had it failed, the doc would have lied with my name on it. Command first, doc second. Same class: TODO_LIST #6 baselines were written from the 09-22 report (83.9/80.3) before I measured; the live run changed two numbers and added a regression (git 98.5→91.0) I then had to fold in.
6. **Used bash `sed`/`cat` as a substitute for View before edits** — the edit tool rejected it twice (DOMAIN_LANGUAGE, 20-26 §g), burning rounds on a rule I know: bash-read is not a read.
7. **Never checked `git status -sb` until the report was due.** The 2026-09-15 18:19 report literally prescribes "end-of-session push-state check — 'green' is not a property of the working tree". I discovered "ahead 16" an hour into the report instead of at session start. A documented lesson, repeated.
8. **Bulk-verdict epistemics.** The 41-file digest-parallelization was the right call for coverage, but it pushed me into category-level verdicts ("this class of item shipped in v0.8.0") applied to individual rows. Every verdict cites real evidence, but the evidence strength varies; per-item code checks for all ~2,100 would have been the gold standard and was not feasible. Disclosed, not hidden — but it is a weaker guarantee than the session's headline suggests at first glance.

---

## e) WHAT WE SHOULD IMPROVE

1. **Per-file ledger checklist for bulk sweeps.** Memory tracked 50+ files and dropped 5. The digest agents should return a table I tick off row-by-row (read → classified → annotated → gate-checked → archived), not inform my memory.
2. **Run the gate mid-sweep, not only at the end.** The 5 misses existed for the whole annotate phase. Running `grep -L '~~'` + `check-rows.py` after each directory batch would have caught them while context was fresh.
3. **Measure-then-write, always.** Every number that enters a doc (coverage, counts, ahead/behind) gets its live command run BEFORE the doc edit that cites it.
4. **Command before claim.** When a doc line asserts "X was done", run X first, then write the line. (d.5.)
5. **`git status -sb` is a session start AND end reflex** — already a written lesson; it needs to be in my first and last tool call of every session in this fleet.
6. **Adapt after the first tool error, not the third.** The `#N` occurrence footgun, the heading-style sections, and the read-before-edit rejections all repeated. One failure → write the workaround down in the running session → apply everywhere.
7. **Digest-parallelize earlier.** The 4 parallel file digests were the single highest-leverage move of the session (accurate structure, huge context savings) — launched at file ~30 of 62. Next time: after file 10, not file 30.
8. **The archive convention needs the loop closed every time.** `docs/status/` is clean today because this pass harvested AND archived. The next status report that ends with "HARVEST pending" recreates the clutter. Every future report: HARVEST → annotate → archive in the following docs-health pass, no exceptions.
9. **AGENTS.md growth is unbounded.** This session added ~1 KB (Docs Layout, status line, policy rewrite). Without a trim pass it crosses 40 KB within a quarter. A "move release-era narratives to `docs/`" pass is cheap and safe; it just needs a decision (g.2).
10. **Commit-story debt is compounding.** The sweep landed as a 124-file heuristic daemon commit. Content is correct; history tells no story. The fleet accepts this by policy — but it keeps costing every future `git log`/`git blame` reader.

---

## f) Up to 50 things we should get done next

_Sources: [T] = already a TODO_LIST item (do NOT re-harvest, they're current), [S] = this session's new findings, [R] = ROADMAP raw idea (not TODO_LIST-grade)._

### Immediate (this session's loose ends)

| #     | Task                                                                                                                                                                 | Impact       | Effort | Src      |
| ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | -------- |
| ~~1~~ | ~~**Push master (16 ahead)** — CI validates the sweep; `website-deploy` fires on the mdx edits~~ done — pushed 09-28 (e1d46ff); CI + Deploy green                    | ~~Critical~~ | ~~S~~  | ~~S~~    |
| ~~2~~ | ~~Run `astro build` locally before/with the push (check alone ran this session)~~ done — astro build green; canary enforces in CI                                    | ~~High~~     | ~~S~~  | ~~S~~    |
| ~~3~~ | ~~**Investigate why release.yml skipped v0.10.2** (workflow file at tag ref? push mechanics?)~~ done — root-caused (M02, archived note)                              | ~~High~~     | ~~M~~  | ~~[T3]~~ |
| ~~4~~ | ~~Curate the v0.10.2 GitHub Release notes (auto-generated now; v0.10.1 was curated)~~ done — curated (M03)                                                           | ~~Low~~      | ~~S~~  | ~~S~~    |
| ~~5~~ | ~~Explicit `erraudit` re-run + go-structure-linter CLI + full buildflow + website `nix build` (the claims battery)~~ done — battery 09-28 + 09-29 (erraudit 1c6809a) | ~~Medium~~   | ~~M~~  | ~~[T7]~~ |
| ~~6~~ | ~~Annotate + archive `docs/feedback/sec-consumer-feedback.md` (only file outside the 2026-0* sweep)~~ done — M14 archived; inline resolutions added 09-29            | ~~Low~~      | ~~S~~  | ~~S~~    |

### TODO_LIST Active items (verified open 2026-09-27 — no changes needed, just execute)

| #      | Task                                                                                                                                    | Impact     | Effort | Src       |
| ------ | --------------------------------------------------------------------------------------------------------------------------------------- | ---------- | ------ | --------- |
| ~~7~~  | ~~Announce the Bridge Patterns guide publicly~~ done — routed to TODO_LIST #1 (channel decision still open)                             | ~~High~~   | ~~M~~  | ~~[T1]~~  |
| ~~8~~  | ~~Retract broken v0.6.x tags (`retract` directives + verify proxy)~~ done — v0.10.3 retraction (M04)                                    | ~~High~~   | ~~M~~  | ~~[T2]~~  |
| ~~9~~  | ~~Website guard canaries (TS-6 pin, frozen-lockfile + astro check, Dependabot decision)~~ done — website-check.yml (M05)                | ~~High~~   | ~~S~~  | ~~[T4]~~  |
| ~~10~~ | ~~art-dupl suppression/baseline + standing threshold policy~~ done — baseline + policy (M09)                                            | ~~Medium~~ | ~~S~~  | ~~[T5]~~  |
| ~~11~~ | ~~Coverage lifts: diagnose 84.2→90, postgres 78.5→85, git back 91.0→95+~~ done — 97.4 / 98.7 / 89.2 (M06)                               | ~~Medium~~ | ~~M~~  | ~~[T6]~~  |
| ~~12~~ | ~~File upstream BuildFlow issues (pnpm-audit subdirectory lockfiles; phantom IsIgnored)~~ done — #23 filed + #19 evidence comment (M10) | ~~High~~   | ~~S~~  | ~~[T8]~~  |
| ~~13~~ | ~~Website chores: `minimumReleaseAgeStrict` decision~~ done — minimumReleaseAgeStrict (M12)                                             | ~~Low~~    | ~~S~~  | ~~[T9]~~  |
| ~~14~~ | ~~Plan v0.11.0 scope~~ done — v0.11.0 shipped 2026-09-29                                                                                | ~~Medium~~ | ~~S~~  | ~~[T10]~~ |

### Structural / hygiene

| #      | Task                                                                                                                                                                           | Impact     | Effort | Src   |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------- | ------ | ----- |
| ~~15~~ | ~~**Trim AGENTS.md below 30 KB** — move release-era narrative bullets (2026-05/06/07 incident stories) to `docs/`, keep current-state rules~~ done — M13 (18,020 bytes)        | ~~Medium~~ | ~~M~~  | ~~S~~ |
| ~~16~~ | ~~Add a one-line comment in `go.work` (or AGENTS) explaining the deliberate `1.26.7` toolchain floor vs per-module `1.26`/`1.26.0` floors~~ done — M14 go.work comment         | ~~Low~~    | ~~S~~  | ~~S~~ |
| ~~17~~ | ~~Line-verify the peripheral doc surfaces (research/, modularization/, comparison html, top-5 docs)~~ done — M14 peripheral audit note                                         | ~~Low~~    | ~~M~~  | ~~S~~ |
| ~~18~~ | ~~Annotate `architecture-understanding/` renders as point-in-time (or date-stamp their intro)~~ done — 09-29 pass stamped the HTML render + d2 files point-in-time             | ~~Low~~    | ~~S~~  | ~~S~~ |
| ~~19~~ | ~~Decide the fate of `best-of-both-worlds.html` (archived with planning; verify it's not referenced anywhere live)~~ done — M14 confirmed archived, zero dangling refs         | ~~Low~~    | ~~S~~  | ~~S~~ |
| ~~20~~ | ~~Record the `#N` occurrence-suffix footgun of annotate-rows/prose in the docs-health skill notes (upstream-able)~~ done — skill documents the #N suffix + level-aware scoping | ~~Low~~    | ~~S~~  | ~~S~~ |
| ~~21~~ | ~~Convention: every future status report ends HARVEST-closed — harvest + annotate + archive in the next docs-health pass~~ done — convention followed by the 09-29 pass        | ~~Medium~~ | ~~—~~  | ~~S~~ |
| ~~22~~ | ~~Consider `docs/status/README.md` one-pager index (current convention + archive pointer) for human discoverability~~ done — routed to ROADMAP fuel (status index, still open) | ~~Low~~    | ~~S~~  | ~~S~~ |
| ~~23~~ | ~~Add CI md-link check so moved/archived files can't break living-doc citations silently~~ done — routed to ROADMAP fuel (lychee CI gate, still open)                          | ~~Medium~~ | ~~S~~  | ~~S~~ |
| ~~24~~ | ~~Triage the 6 open Dependabot PR branches (CI ran green on them today) — merge or close with intent~~ done — M18 closed #6-#11                                                | ~~Medium~~ | ~~S~~  | ~~S~~ |

### From the 09-18/09-22 reports (carried, unchanged priority)

| #      | Task                                                                                                                      | Impact       | Effort | Src         |
| ------ | ------------------------------------------------------------------------------------------------------------------------- | ------------ | ------ | ----------- |
| ~~25~~ | ~~CI canary: typescript major ≠ 6 fails CI (see #9)~~ done — canary (M05)                                                 | ~~Critical~~ | ~~S~~  | ~~[T4]~~    |
| ~~26~~ | ~~CI: `pnpm install --frozen-lockfile` + `astro check` for `website/**` PRs (see #9)~~ done — canary (M05)                | ~~High~~     | ~~S~~  | ~~[T4]~~    |
| ~~27~~ | ~~Decide/disable Dependabot security-updates auto-run for `/website` (see #9)~~ done — decision verified + recorded (M05) | ~~High~~     | ~~S~~  | ~~[T4]~~    |
| ~~28~~ | ~~ROADMAP answer: fleet churn root-fix vs per-repo canaries forever~~ done — routed to ROADMAP OQ2 (open)                 | ~~High~~     | ~~—~~  | ~~[R-OQ2]~~ |
| ~~29~~ | ~~ROADMAP answer: bridge & enrichment APIs — invest or freeze?~~ done — routed to ROADMAP OQ3 (open)                      | ~~High~~     | ~~—~~  | ~~[R-OQ3]~~ |
| ~~30~~ | ~~ROADMAP answer: art-dupl `-t 1` routine vs deep-sweep threshold~~ done — OQ1 resolved with recorded default             | ~~Medium~~   | ~~—~~  | ~~[R-OQ1]~~ |

### Adoption / docs backlog (ROADMAP fuel, not commitments)

| #      | Task                                                                                                                                                               | Impact     | Effort  | Src   |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------- | ------- | ----- |
| ~~31~~ | ~~Examples for `errorfamilytest` + `diagnose` subpackages (pkg.go.dev gap)~~ done — M15 examples                                                                   | ~~Medium~~ | ~~M~~   | ~~R~~ |
| ~~32~~ | ~~End-to-end `HTTPHandler` middleware example (net/http or Chi)~~ done — M17 (covered by ExampleHTTPHandler; deviation documented)                                 | ~~Medium~~ | ~~M~~   | ~~R~~ |
| ~~33~~ | ~~Example symmetry: `ExampleNewConflict/Corruption/Infrastructure/Orchestration` set~~ done — M17 constructor symmetry                                             | ~~Low~~    | ~~S~~   | ~~R~~ |
| ~~34~~ | ~~gRPC status-mapping guide (family → codes.Internal/Unavailable/InvalidArgument)~~ done — M16 gRPC guide                                                          | ~~Medium~~ | ~~M~~   | ~~R~~ |
| ~~35~~ | ~~OpenAPI/schema generation for the canonical error JSON~~ done — routed to ROADMAP backlog (OpenAPI)                                                              | ~~Medium~~ | ~~L~~   | ~~R~~ |
| ~~36~~ | ~~`httperror` subpackage RFC~~ done — routed to ROADMAP backlog (httperror RFC)                                                                                    | ~~Medium~~ | ~~L~~   | ~~R~~ |
| ~~37~~ | ~~Diagnostic submodules: redis, then docker/kubectl~~ done — routed to ROADMAP backlog (diagnostic submodules)                                                     | ~~Medium~~ | ~~M/L~~ | ~~R~~ |
| ~~38~~ | ~~Typed `DiagnosticResult.Details` (kills strTrue/strFalse at root) — next major only~~ done — routed to ROADMAP backlog (typed Details, next major)               | ~~Low~~    | ~~L~~   | ~~R~~ |
| ~~39~~ | ~~Release automation script with explicit trigger verification (ends the #3 class)~~ done — routed to ROADMAP theme 3 (automation script)                          | ~~High~~   | ~~L~~   | ~~R~~ |
| ~~40~~ | ~~Framework integration guides: Chi, Echo, Gin~~ done — routed to ROADMAP theme 4 (framework guides)                                                               | ~~Medium~~ | ~~M~~   | ~~R~~ |
| ~~41~~ | ~~Benchmark suite tracked across versions~~ done — routed to ROADMAP theme 4 (benchmark suite)                                                                     | ~~Low~~    | ~~M~~   | ~~R~~ |
| ~~42~~ | ~~CSP headers for the website (honest standing gap; declining it forever is also a decision)~~ done — declined with revisit trigger (M18 DR-1)                     | ~~Low~~    | ~~S~~   | ~~S~~ |
| ~~43~~ | ~~Uptime monitor for errorfamily.lars.software (recurring idea, never adopted — decide once)~~ done — declined with revisit trigger (M18 DR-2)                     | ~~Low~~    | ~~S~~   | ~~R~~ |
| ~~44~~ | ~~`Code()` vs `ErrorCode()` convergence proposal (next major)~~ done — routed to ROADMAP theme 1 (next-major convergence)                                          | ~~Low~~    | ~~S~~   | ~~R~~ |
| ~~45~~ | ~~Consumer survey refresh: who uses LogError/HTTPHandler/errorfamilytest today (last audit 2026-07-23)~~ done — routed to ROADMAP fuel (survey refresh)            | ~~Medium~~ | ~~M~~   | ~~R~~ |
| ~~46~~ | ~~Review `WithContextf`/`WithContextMap` example gap (only WithContextAny has one)~~ done — routed to ROADMAP fuel (WithContextf/Map examples still absent)        | ~~Low~~    | ~~S~~   | ~~S~~ |
| ~~47~~ | ~~Consider a weekly `gh run list` triage habit (red workflows persisted 3 days unnoticed once)~~ **Won't implement — standing personal cadence, not a repo task.** | ~~Medium~~ | ~~S~~   | ~~S~~ |
| ~~48~~ | ~~`minimumReleaseAgeStrict` decision is #13; this row intentionally left as a pointer~~ **NOT-DO — pointer row, intentionally empty.**                             | ~~—~~      | ~~—~~   | ~~—~~ |
| ~~49~~ | ~~Check whether `docs/research/` content is still referenced by anything (or archivable)~~ done — M14 audit: REFERENCE keep                                        | ~~Low~~    | ~~S~~   | ~~S~~ |
| ~~50~~ | ~~Next docs-health pass: harvest THIS report's f-section, then annotate + archive it~~ done — this pass harvested, annotated, and archived this file               | ~~Medium~~ | ~~S~~   | ~~S~~ |

---

## g) Questions I cannot figure out myself

1. ~~**Push now?** `master` is 16 commits ahead — the entire sweep, the living-doc fixes, and the v0.10.2 release-notes remediation are local-only, and the two `website/**` `.mdx` fixes mean the next push triggers the full `website-deploy` pipeline. Do you want me to push (and watch CI + website-deploy), do you push yourself, or does the fleet orchestrator own pushes on its own schedule?~~ done — pushed; all pipelines green on every subsequent push
2. ~~**AGENTS.md: trim or keep dense?** It is ~37 KB (flag threshold 30 KB). I can move the release-era narrative bullets (2026-05/06/07 incident stories, json/v2 saga details, lint-archaeology rationale) to `docs/` and keep only current-state rules — that would land it near 20 KB. Or is maximum density by design for this repo?~~ done — M13 trimmed to 18,020 bytes; narratives archived verbatim
3. ~~**v0.10.2 GitHub Release notes: keep or curate?** I created the missing release with auto-generated notes (matching the workflow's `generate_release_notes: true`). v0.10.1's release was hand-curated (summary + modules table). Should I curate v0.10.2 the same way — and if yes, should the 0.x-releases-prerelease convention question (from the 09-15 report, still unanswered) be decided at the same time?~~ done — M03 curated; G4 recommendation recorded (keep 0.x releases FULL)

---

_Point-in-time snapshot — stale on contact. Written as Markdown per explicit user request (status-report skill's canonical format is HTML; one-off override, not propagated). Verify claims against the repo before acting on them._
