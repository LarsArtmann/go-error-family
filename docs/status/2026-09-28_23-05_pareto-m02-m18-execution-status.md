# Status Report — Pareto Plan Execution M01–M18 (docs-health → release → ecosystem)

**Generated:** 2026-09-28 23:05 CEST
**Session scope:** executed the ENTIRE Pareto execution plan
(`docs/planning/2026-09-28_01-50_superb-pareto-execution-plan.md`): M02–M18 plus
final verification. M01 (push + CI validation) was completed at session start.
**Format override:** skill default is styled HTML; the user explicitly requested
`.md` — honored, flagged here per skill contract.

**Headline:** all 18 plan tasks executed; a coordinated release (v0.10.3) shipped
with retractions verified against the origin; one real code fix escaped the
standing-claims battery; CI + Website Check + Deploy Website all green on the
final push (`8ff919f`).

---

## a) FULLY DONE (evidence-carried)

| # | Item | Evidence |
| - | ---- | -------- |
| 1 | **M02 — release.yml no-fire ROOT-CAUSED** | Zero `refs/tags/v0.10.2` PushEvent in the events API; workflow byte-identical at both tags; tag SHA matches local → GitHub's combined branch+tag push silently dropped the tag webhook. Findings note: `docs/status/2026-09-28_02-10_release-yml-no-fire-root-cause-v0.10.2.md` |
| 2 | **M02 — mitigation landed AND live-tested** | `workflow_dispatch` fallback in release.yml (explicit `tag_name`, force-fetch/checkout of input tag); dispatched on v0.10.3 → run 36478697983 `success`; runbook (separate tag/branch pushes, ~2-min trigger check) in AGENTS.md |
| 3 | **M03 — v0.10.2 GitHub Release curated** | Notes published in v0.10.1 style (7-module table, highlights); G4 recommendation recorded: keep 0.x releases marked FULL, prerelease flag = literal alpha/beta/rc only |
| 4 | **M04 — full tag audit: 60+ tags** | Found **5 broken tags, not the 2 the TODO assumed**: root `v0.5.0`, `v0.5.1`, `v0.6.0` (local-dir replaces; v0.6.0 + phantom pseudo-version) + `diagnose/v0.1.0` + `agent/v0.1.0`. `v0.6.1` verified CLEAN (zero requires) and deliberately NOT retracted |
| 5 | **M04 — retraction shipped as coordinated v0.10.3** | Tags `diagnose/v0.2.5` → `agent/v0.2.5` → `v0.10.3` (release commit `7ffd883`), each pushed separately per the new runbook; Release workflow FIRED first try — validating the M02 root cause in practice |
| 6 | **M04 — retraction verified end-to-end** | `go list -m -retracted` via `GOPROXY=direct` (origin): all 5 return the exact rationale; `v0.6.1`/`v0.10.3`/`v0.2.5`s clean; `@latest` resolves v0.10.3; proxy `.info` serves v0.10.3 |
| 7 | **M04 — docs synced** | CHANGELOG `[0.10.3]` (+ factual fix to 0.10.2's Modules claim: submodules pin root v0.10.1 at tag time); website `changelog.mdx` gained **missing 0.10.1 AND 0.10.2** sections plus 0.10.3; ROADMAP theme 3 updated |
| 8 | **M05 — website guard canary live** | `.github/workflows/website-check.yml`: TS-major==6 canary, `pnpm install --frozen-lockfile`, `astro check`, `astro build`; validated on its own first push (run 36483254236 `success`, 28s); PR-triggered too, so Dependabot TS bumps fail BEFORE merge |
| 9 | **M05 — Dependabot decision verified, not guessed** | GitHub docs confirm: no per-directory security-update toggle; `open-pull-requests-limit: 0` does NOT disable security updates. Decision: leave enabled, red security job = the alert signal, remediate via `pnpm audit` (recorded in AGENTS.md) |
| 10 | **M06 — coverage lift, all targets exceeded** | diagnose core 84.2→**97.4%** (target 88), diagnose/git 91.0→**98.7%** (95), postgres 78.5→**89.2%** (83); all with `-race`; new `diagnose/command_test.go` + git remote-failure branches + postgres run-error/TCP branches; FEATURES + AGENTS tables updated with Δ |
| 11 | **M07 — battery caught a real regression + fixed** | erraudit 0 findings restored: Sep-22 erraudit binary added a blank-identifier-on-`recover` rule AFTER the Sep-15 zero-claim → `safeCauseString` now carries the documented `//nolint:legacyerrors` + rationale; all 7 modules 0 findings |
| 12 | **M07 — rest of battery green (after one fix)** | structure-linter exit 0 (+ applied its `*.png binary` advisory); `buildflow --build-mode full` no-cache: the 1 failure was the treefmt FORMAT check (golines wanted new long lines wrapped) → `nix fmt`, format check green; website build via `nix run .#build` (15 pages) — claim corrected: flake exposes `apps.build`, NOT `packages.default` |
| 13 | **M09 — art-dupl suppression + policy** | `baseline`/`check` subcommands exist; `.art-dupl-baseline.json` committed (3 accepted groups, threshold 1, check exits 0); policy recorded: routine gate `-t 5`, deep sweeps `-t 1` against baseline (resolves ROADMAP OQ1 as a default) |
| 14 | **M10 — both BuildFlow diagnoses source-verified + filed** | (a) `lockfilePatterns` globs `**/pnpm-lock.yaml` while the detector runs at `rootDir` (js_tools.go:351/205) → added as evidence comment on existing issue **#19** (no duplicate filed); (b) phantom analyzer: `IsIgnored` wired in roleak/doanalyzerv2, ZERO calls in `pkg/phantom/` (7 files) despite the core ENUM declaring `phantom` → filed **BuildFlow#23**; both linked from `.buildflow.yml` |
| 15 | **M11 — v0.11.0 scoped** | CHANGELOG `[Unreleased]` skeleton (discoverability release, no API changes); release checklist written into TODO_LIST #8 |
| 16 | **M12 — supply-chain knob applied** | `minimumReleaseAgeStrict: true` in `website/pnpm-workspace.yaml` (docs-verified: pnpm 11 already defaults a 1-day cooldown; strict = fail-loud); frozen install passes the trustLockfile check ("492 entries in 2.2s"); `astro check` 0 issues |
| 17 | **M13 — AGENTS.md trim, target beaten** | 39,865 → **18,020 bytes** (55% cut, target was <30 KB); every still-true rule kept, all narratives archived VERBATIM in `docs/history/agents-archive.md`; every internal citation path verified to exist |
| 18 | **M14 — hygiene batch** | SEC feedback archived with verified-resolution banner (all PP1–5/IDEA1–4 shipped by v0.10.1); `go.work` floor-vs-toolchain comment; `best-of-both-worlds.html` confirmed correctly archived (zero dangling refs); peripheral-surface audit recorded (`docs/status/2026-09-28_03-00_m14-peripheral-surface-audit.md`) |
| 19 | **M15 — errorfamilytest + diagnose examples** | 7 assertion-helper examples (package's first); executable `ExampleRuleSpec` (data-driven rule + Runner + `ResolveContextKey`); both modules lint 0 issues |
| 20 | **M16 — gRPC guide shipped** | `website/src/content/docs/guides/grpc.mdx`: family→gRPC table (per Google's official HTTP↔gRPC mapping), interceptor, retry guidance (Infrastructure also = Unavailable but single-attempt); sidebar entry; `astro check` 0 issues, build **16 pages** |
| 21 | **M17 — constructor symmetry** | `ExampleNewConflict/Corruption/Infrastructure/Orchestration` (root now **30 runnable examples**); one example fixed my wrong assumption (Infrastructure `IsRetryable()` = false) — the example now documents true behavior |
| 22 | **M18 — triage + decision records** | 6 superseded Dependabot PRs (#6–#11, root 0.10.1→0.10.2) closed with rationale (v0.10.3 supersedes; Dependabot re-files); CSP + uptime-monitor both formally DECLINED with reasoning + revisit triggers (`docs/status/2026-09-28_03-30_decision-records-csp-uptime.md`) |
| 23 | **Final verification — everything green** | All 7 modules `go test -race -cover` OK; golangci-lint 0 issues ×7; `GOWORK=off` build OK; `GOEXPERIMENT= go build` OK (CI-parity reflex); `nix build .#checks.x86_64-linux.format` green; final push `8ff919f` → CI `success` (1m2s), Website Check `success` (28s), Deploy Website `success` (54s) |
| 24 | **M08 — announcement drafted + self-reviewed** | see b) — the publish is a deliberate user gate |

## b) PARTIALLY DONE

| # | Item | State | What remains |
| - | ---- | ----- | ------------ |
| 1 | **M08 Bridge announcement** | Drafted at `docs/planning/2026-09-28_bridge-patterns-announcement-draft.md`; self-reviewed against guide content (no overclaiming; zero-consumers honesty; "19 tests" verified) | **USER GATE:** review, pick channel (GitHub Discussion vs r/golang), publish, cross-link from README/related-tools |
| 2 | **Retraction propagation on proxy.golang.org** | Verified via `GOPROXY=direct` (origin truth); proxy `.info` already serves v0.10.3 | The proxy's cached `@latest` still returned v0.10.2 at last check — retraction semantics on the PROXY (not origin) propagate when that cache refreshes; re-verify `go list -m -retracted` WITHOUT `direct` later |
| 3 | **v0.11.0 release** | Scope frozen, CHANGELOG skeleton written, checklist in TODO_LIST #8; the gRPC guide (its main content item) already landed and deployed | Cut the actual release (bump CHANGELOG date, sync website changelog, tags per runbook, curated notes) |
| 4 | **TODO_LIST item 7 (website chores)** | Decision made + applied (`minimumReleaseAgeStrict`) | Item text still carries an inline "DONE 2026-09-28" marker instead of being removed — TODO_LIST convention says remove-on-ship |
| 5 | **gRPC guide as shipped artifact** | Live on the website (16 pages, deployed) | The interceptor code sample is documentation-only — never compiled/CI-verified (no grpc dep in any module, by design) |

## c) NOT STARTED (deliberately out of this session's scope)

1. ROADMAP Open Question **G2** (fleet churn: root-fix vs canaries-forever) — unblocked input arrived (BuildFlow #19/#23), decision pending
2. ROADMAP Open Question **G3** (bridge: invest vs freeze) — follows announcement reception
3. G1 got only a recorded **default** (art-dupl `-t 5` routine / `-t 1`+baseline sweeps); formal user confirmation still open
4. Release automation script (ROADMAP theme 3; runbook exists, automation doesn't)
5. `docs/status/` index file (current vs archived)
6. CI markdown-link check (lychee) — battery ran it; not a CI gate; 2 known dead consumer-repo links in ARCHIVED feedback docs (frozen records, left alone)
7. CI gate for `art-dupl check -t 1` (baseline committed but not enforced in CI)
8. `pnpm audit` in `website-check` workflow (manual cadence still manual)
9. Backlog (ROADMAP fuel, demand-gated): OpenAPI schema · `httperror` RFC · redis/docker/kubectl submodules · typed `DiagnosticResult.Details` · `Code()`/`ErrorCode()` convergence · benchmark suite · consumer survey refresh · Echo/Gin guides
10. pkg.go.dev spot-checks (30 root examples, new errorfamilytest examples) — only visible after v0.11.0

## d) TOTALLY FUCKED UP (honest list)

1. **First tag audit was WRONG.** I read the ROOT `go.mod` at submodule tags instead of `<subdir>/go.mod` — briefly "concluded" `bridge/v0.2.0` had a poisoned module path (false alarm; bridge/v0.2.0 is fine). Caught it before any action; redone correctly. Lesson: submodule tags serve the SUBDIRECTORY go.mod via the proxy.
2. **The TODO's own scope was under-inclusive and I almost shipped it as-is.** "Retract the v0.6.x family" would have left root `v0.5.0`/`v0.5.1` AND two submodule `v0.1.0` tags broken-and-gettable. Only the extended audit caught the real inventory. Scope-then-verify, not verify-the-assumption.
3. **The standing-claims battery exposed a STALE claim I would have re-asserted.** "erraudit 0 findings (re-verified 2026-09-27)" was false against the Sep-22 erraudit binary — the 2026-09-27 pass evidently didn't actually exercise root module erraudit output. Process gap: claims recorded without tool version stamps decay invisibly.
4. **buildflow full failed on formatting** — I violated a rule AGENTS.md itself documents (golines wraps long lines; run `nix fmt` before claiming clean). One wasted no-cache full run (~4 min) to rediscover it.
5. **Repeated test-first-writing mistakes** (I wrote expectations from memory instead of reading the code under test): 3 git tests missing `initGitRepo`; a `MockResponse{Err: ...}` without `ExitCode: -1` modeling an impossible exec state; two wrong `stripHost` expectations; timeout-test expectation contradicting the actual ExitError contract; two composite-literal parse errors (`X{}` in if-conditions). Each cost a test run. The fixes are correct and now document real behavior — but the loop count was sloppy.
6. **Daemon commit-race fumbling.** Two of my `git add -A && git commit` turns hit "nothing to commit" because the auto-commit daemon had already committed; my semantic commit messages then landed on partial file sets (e.g. the "godoc examples wave" commit carried only 4 files). History is fragmented across heuristic daemon commits and my semantic ones. No content lost — but `git log` reads worse than the work deserves.
7. **This very report was LATE.** The previous instruction asked for it "RIGHT NOW"; I only ran `date` + the CI check before the session cut. It exists now.
8. **Minor:** `multiedit` was rejected twice on `command_test.go` (daemon formatter touched the file between read and edit) — re-read via `view` was required; I first retried blind.

## e) WHAT WE SHOULD IMPROVE

1. **Run `nix fmt` (or buildflow format) BEFORE every commit/push** — format failures are free to prevent, expensive to discover in a no-cache buildflow run.
2. **Read the function under test before writing its test** — half my test iterations came from invented expectations.
3. **Full inventory before scoping a "family" fix** — the retraction TODO underspecified reality by 3 tags; audits beat assumptions.
4. **Stamp tool versions + dates into standing-claims records** (`erraudit <version>`, golangci pin, art-dupl build) — version drift silently invalidates claims (this bit twice: erraudit, and the `nix build` vs `apps.build` shape).
5. **For submodule tag work: always read `<subdir>/go.mod` at the tag**, never the repo-root go.mod.
6. **Add the release runbook's proxy check**: after tags, verify `proxy.golang.org/@latest` propagation, not just origin (`GOPROXY=direct`) — the two can disagree for hours.
7. **TODO_LIST discipline**: remove shipped items; inline "DONE" markers rot.
8. **Commit hygiene under the daemon**: for releases/atomic sequences, commit immediately after each logical unit (I did for tags) and accept daemon interleave elsewhere; alternatively investigate a session-scoped daemon pause — config change, needs a decision.
9. **Enforce the art-dupl baseline in CI** (`art-dupl check -t 1 .`) — a committed baseline nobody runs is prose, not a gate.
10. **Compile-verify documentation code samples** for new guides (gRPC interceptor) — e.g. a tiny `examples/cmd/grpcboundary` or a build-tagged test, else they rot silently.
11. **Restart the golangci LSP after bulk edits** — stale diagnostics (the 14 phantom `testableexamples` warnings) cost attention that the real linter didn't.
12. **Keep AGENTS.md at the new 18 KB weight** — new rules require equal-size trims; the archive exists precisely for this.
13. **Battery should include `pnpm audit` in `website/`** (the manual cadence) and the website `nix run .#build` with its CORRECT invocation.

## f) Top 50 things we should get done next

> Brainstorm, not commitment (per status-report skill): items 1–12 are
> TODO_LIST-ready; 13–50 are ROADMAP fuel to be routed through docs-health
> HARVEST with normal rigor. Sorted by impact.

1. **User review + publish the bridge announcement** (gate; draft ready) and cross-link from README/related-tools
2. **Cut v0.11.0** per TODO #8 checklist (content is already on master; sync website changelog, date the CHANGELOG section, tags per runbook, curated notes)
3. **Re-verify retraction via the proxy** (`go list -m -retracted` without `direct`) once `@latest` cache refreshes; record the date
4. **Confirm Dependabot re-files the 6 submodule bumps against v0.10.3**; triage those PRs (expected green, merge or hold for v0.11.0)
5. **Add `art-dupl check -t 1 .` to CI** so the baseline is enforced, not decorative
6. **Add `pnpm audit` to `website-check`** (covers the BuildFlow `pnpm-audit` skip locally until upstream lands #19)
7. **Compile-verify the gRPC interceptor sample** (extract to `examples/cmd/grpcboundary` or build-tagged test)
8. **Release automation script** (ROADMAP theme 3): tag sequence + 2-min Release-run check + dispatch fallback + proxy propagation check
9. **Resolve G1 formally** (confirm or override the art-dupl threshold default)
10. **Track BuildFlow #19/#23 to fix; remove the two `.buildflow.yml` skips when they land** (each removal re-enables real coverage)
11. **Write `docs/status/` index** (living vs archived pointer table)
12. **TODO_LIST hygiene pass**: remove item 7's inline-DONE, renumber; keep ≤10 bounded items
13. Add retraction-aware consumer-simulation gate to CI (`go list -m -retracted` on pinned broken versions → expect failure)
14. lychee md-link check as a CI step with an explicit allowlist for archived docs' dead external links
15. pkg.go.dev spot-check after v0.11.0: 30 root examples render; errorfamilytest examples visible
16. Verify `errorfamily.lars.software` live-serves the gRPC guide + synced changelog (URL fetch)
17. SKILL.md sync: 30-example count, gRPC guide mention, v0.10.3 status line
18. README: add gRPC guide cross-link in the boundary section (website sidebar done; README not yet)
19. HTTP guide: add reciprocal "See Also" link to the gRPC guide (gRPC guide links out; reverse link missing)
20. ROADMAP: mark theme 3's automation idea as partially shipped (runbook + dispatch fallback exist)
21. ROADMAP: point OQ1 at the AGENTS art-dupl policy line (recorded default)
22. Decide G2 (fleet churn strategy) once BuildFlow #19/#23 progress is visible
23. Decide G3 (bridge invest-or-freeze) after announcement reception data
24. Post-v0.11.0: re-cut Dependabot expectation list and record the new baseline pin state
25. erraudit version stamp + dated tool table for the claims battery (process fix from d-3)
26. Add battery item: website `nix run .#build` (correct invocation, with the `apps.build` note)
27. Investigate daemon-pause option for release sequences (commit-boundary hygiene, d-6)
28. `docs/history/agents-archive.md`: add a header pointer FROM AGENTS.md only (done) and ensure HARVEST never routes new narratives there without a rule reference
29. Retract-scope precedent: write the "full inventory before family-fix" lesson into the project docs (AGENTS one-liner) or `references/lessons.md` in crush-config (cross-project)
30. Consider `--diff-report` in deep art-dupl sweeps for triage UX
31. diagnose/git: the `resolveRepoPath` `"."` fallback branch stays uncovered (os.Getwd failure) — accept explicitly or refactor for testability (gated, tiny)
32. postgres `IsPostgresRunning` runErr branch uncovered without pg_isready-present+error env — same accept-or-refactor decision (touches published API shape → gated)
33. `examples/cmd/http`: optional bridge-pattern HTTP variant (adoption gap for the enrichment layer)
34. `errorfamilytest`: consider `AssertJSON` (consumer-demand-gated)
35. Website: consider an `apps.check` in website flake bundling check+build (matches the corrected `apps.build` shape)
36. Website: cache astro build artifacts in CI if deploy time ever matters (now 54s — skip unless it regresses)
37. `go.work.sum` pruning step in the release checklist (documented incident class from the diagnose v0.2.2 re-point)
38. Record the two dead consumer links (archived feedback docs) in the future lychee allowlist (item 14 dependency)
39. `docs/DOMAIN_LANGUAGE.md`: no gRPC terms needed (website-only guide) — record that decision to close the question
40. Consider a `retract` policy line in AGENTS release runbook (when to retract: any tag that fails a clean-room `GOWORK=off` consumer build)
41. Add a clean-room consumer build smoke test (`GOWORK=off`, temp module, `go get root@latest`) to the release checklist — catches replace/require rot (the v0.6.x class) pre-tag
42. pkg.go.dev: after v0.11.0, check the gRPC guide's discoverability from README (GitHub render) — README badge/links
43. ROADMAP: benchmark-suite idea — scope it (benchstat across v0.10.x→v0.11.0) as a bounded spike
44. Consumer survey refresh (ROADMAP) — schedule after the announcement has a week of signal
45. Dependabot: revisit the declined npm ecosystem entry ONLY if GitHub ships per-directory security toggles (tracked via community discussion 69580)
46. Consider shipping the bridge announcement ALSO as a GitHub Discussion so it is linkable from the repo README regardless of Reddit outcome
47. `related-tools.mdx`: add the gRPC guide link if its structure lists guides
48. AGENTS.md: add the "read `<subdir>/go.mod` at submodule tags" gotcha (d-1) as a one-liner — costs one line, prevents a repeat
49. Celebrate-scan: verify no `//nolint` added this session lacks a rationale comment (erraudit/nolintlint contract) — quick grep
50. Next session start: re-run the standing-claims battery WITH version stamps (items 25/26) to re-anchor post-release state

## g) Questions I can NOT figure out myself

1. **Bridge announcement channel + publish consent:** GitHub Discussion on `go-error-family`, r/golang, or both? It goes out under your name — draft is ready (`docs/planning/2026-09-28_bridge-patterns-announcement-draft.md`), I will not publish without your go.
2. **v0.11.0 timing:** cut it now (all content already on master, checklist ready, ~30 min per runbook) or hold it to bundle whatever comes out of the announcement/BuildFlow #19/#23? I can't infer your release-cadence appetite.
3. **G1 art-dupl policy:** confirm the recorded default (routine `-t 5`; deep sweeps `-t 1` against the committed baseline; CI enforcement pending item 5) or state your preferred standing threshold — the ROADMAP Open Question stays open until you do.

---

*Point-in-time snapshot; will go stale. Section (f) items 1–12 are the HARVEST
candidates for TODO_LIST; 13–50 are ROADMAP fuel — not yet routed, per the
"report, then wait" instruction.*
