# Status Report — TODO Execution (CI, Website Guide, Links) + json/v2 Regression Discovery

**Date:** 2026-09-15 10:46 · **Session scope:** execute the three Medium-Priority TODO items (examples CI steps, bridge guide, related-tools link), verify the blocked ACME item, and anything noticed en route · **Reporter:** Crush session

---

## What this session did (one paragraph)

Executed all three Medium-Priority items from the active TODO list: added examples-module test and lint steps to `ci.yml` (verified locally: tests pass with `-race`, golangci-lint 0 issues), created the Bridge Patterns guide page with sidebar entry and cross-links, and verified the built website end-to-end (`astro check` 0/0/0, `astro build` 15 pages). En route: found and fixed a broken website deploy gate (TypeScript 7 had broken `astro check` on master since a 2026-09-13 dep bump), confirmed the ACME TXT item is externally blocked, updated CHANGELOG/TODO_LIST/AGENTS.md, and — the big one — discovered that a concurrent session had committed `encoding/json/v2` imports into the root module, which does not compile without `GOEXPERIMENT=jsonv2` (set globally on this machine, masking the breakage locally). CI sets no such env: **master's Go CI jobs are red-in-waiting.** Reported with evidence; awaiting the user's revert-vs-adopt decision. The ACME DNS item remains externally blocked (placeholder Namecheap API key).

---

## a) FULLY DONE

1. **Examples CI test step** — `.github/workflows/ci.yml` test job now runs `go test -race -count=1 ./...` in `./examples` (19 bridge-reference tests + checkout tests finally run in CI, closing the gap that motivated the item). Replaced the old "Verify examples compile" build step: `go test` compiles everything it tests, and every other module's CI pattern is test-step-only. Verified locally: both test packages pass with `-race`.
2. **Examples CI lint step** — lint job now runs the pinned `golangci-lint-action@9fae48a # v7` (v2.12.2) with `working-directory: ./examples`, matching the per-module steps for diagnose/agent/bridge/git/postgres. Examples inherits the root `.golangci.yml` via golangci-lint's parent-dir config search (same mechanism as the other submodules). Verified locally: 0 issues (caveat in b.2).
3. **Bridge Patterns guide page** — `website/src/content/docs/guides/bridge.mdx`: libraries-classify/applications-enrich architecture diagram, the three patterns (pass-through, `AutoWrap`, explicit `Wrap`) with the tag→domain→Transient inference cascade, the one-error-two-representations table, the decision guide table, and links to the reference implementation, `guides/classification`, and `guides/http-and-cli`. Sidebar entry added in `website/astro.config.mjs` ("Bridge Patterns", after Custom Error Types).
4. **Reference implementation linked from `related-tools.mdx`** — the samber/oops section now links both the guide (`/guides/bridge`) and the runnable reference (`examples/cmd/bridge` on GitHub). Verified present in the built `dist/related-tools/index.html`.
5. **Website verified end-to-end** — `astro check` 0 errors/0 warnings/0 hints; `astro build` 15 pages (14 + the new bridge guide); bridge page, sidebar entry, and both related-tools links confirmed in `dist/`. This also exercised the concurrent session's unexercised http-and-cli.mdx change (its own report listed "astro build never ran" as open — now closed as a side effect).
6. **On-sight fix: website TypeScript pin** — `website/package.json` `typescript ^7.0.2` → `^6.0.0` (resolves 6.0.3), lockfile regenerated. A 2026-09-13 dep bump had moved the website to TS 7, whose native compiler drops the programmatic API `astro check` requires — failing the `website-deploy` workflow's check step on every run since. Upstream guidance (withastro/roadmap#1321) is to stay on 6.x until Astro supports the native compiler. Gate is green again.
7. **ACME TXT item verified externally blocked, not faked** — confirmed against `docs/status/2026-07-23_05-07_website-recovery.md` b.1: record staged in the separate `domains` repo (`domains/lars.software.tf`), apply blocked by placeholder Namecheap API key + missing IP whitelist, documented as "Manual step required." Left open in TODO_LIST.md with the blocker spelled out.
8. **Project docs updated** — `CHANGELOG.md` `[Unreleased]` gained Added (CI steps, bridge guide) and Fixed (TS 7) entries; `TODO_LIST.md` lost the three shipped items per its own convention and gained a blocker annotation on the ACME item; `AGENTS.md` CI line updated, Website Known Limitations entry extended with the TS-6.x pin rule and the "system `node` is a Bun shim — use `nix develop -c pnpm`" gotcha, page count 14→15.

## b) PARTIALLY DONE

1. **The new CI steps are proven locally, not in CI** — the commands were run exactly as CI will run them, but the actual workflow hasn't executed on GitHub yet. And due to c.1 (json/v2), the first CI run on master would fail in the _pre-existing_ workspace/GOWORK=off steps regardless of my additions.
2. **Examples lint verified with a different binary than CI pins** — local golangci-lint is v2.13.2; CI pins v2.12.2. High confidence (config unchanged, other modules pass on both), but not the exact pinned binary.
3. **pnpm-lock.yaml rewrite not diff-audited** — the TS 6 pin triggered a 1518-line lockfile rewrite (mostly deletions). Build + check are green after, but I did not attribute every removed resolution variant (TS 7 peer-variant cleanup vs pnpm normalization). Future lockfile diffs will be noisier to read.
4. **Git history quality — daemon interleaving** — the auto-commit daemon swept my four content files into commit 8bc15f2 together with 11 files from the concurrent session (error.go, http.go, README, SKILL.md, flake.lock, tests…), then committed my doc/lockfile changes separately (52cdfc8). My content is committed and correct, but the commits tell no story and mix two sessions' authorship. Same failure mode the 10:29 session reported for itself — now repeated with two concurrent sessions.
5. **Concurrent session's changes were read, not reviewed** — I judged the json/v2 imports and confirmed the README diff is table reformatting, but I did not line-review their error_test.go/http_test.go/SKILL.md changes; that session owns them.

## c) NOT STARTED

1. **The json/v2 regression fix** — discovered, root-caused, evidenced, reported (see d.1 for the full anatomy). NOT fixed: per the "never revert changes you didn't author — ask first" rule, this needs the user's call between (i) reverting the two imports to `encoding/json` (restores the documented zero-env guarantee and green CI) or (ii) deliberately re-adopting json/v2 and wiring `GOEXPERIMENT=jsonv2` through `ci.yml`, `release.yml`, `flake.nix`, and a AGENTS.md reversal entry. Blocked on user decision, not on knowledge.
2. **ACME TXT DNS record application** — externally blocked (see a.7): needs a real Namecheap API key and IP whitelisting in the domains repo. Nothing to do in this repo.
3. **Issue #5 GitHub loop closure** — inherited from the 10:29 session's own "not started" (comment/close with the two deviations). Not mine this session, still open.

## d) TOTALLY FUCKED UP!

1. **I verified green in a masked environment and only caught master's real breakage by accident at the end.** Every Go verification I ran (examples tests, lint, builds) executed with `GOEXPERIMENT=jsonv2` — set in this machine's global Go env — so `encoding/json/v2` imports compiled fine for me. I discovered the regression only while investigating what the daemon had committed, then proved it: `GOEXPERIMENT= go build ./...` fails ("build constraints exclude all Go files in …/encoding/json/v2"). My session's headline task was "make CI run the examples tests" — and on master as committed, CI cannot compile the root module at all. The mask was removable in one command (`GOEXPERIMENT=`, exactly what CI sees) and I ran it ~40 minutes later than I should have. Lesson now explicit: **verify in the CI-equivalent environment (env overrides unset) before claiming green.**
2. **I didn't check for concurrent session activity at session start.** A 10:29 status report and uncommitted foreign changes were sitting in the tree while I worked; I found them at 10:37 via the daemon's commit. A 10-second `git status` + `ls docs/status | tail` at the start would have revealed active concurrent work and changed my verification strategy (CI-parity checks, per-file authorship tracking) from step one instead of step ten.
3. **Fumbled the project's own toolchain invocation.** Ran `pnpm exec astro check` with the system `node` — which is a Bun shim here — and got a baffling `No such built-in module: node:sqlite`; AGENTS.md already documents `nix develop -c pnpm …` as the way. Burned a diagnostic cycle re-learning a documented fact.
4. **Pathspec bug made git history lie to me.** Ran `git log -- website/package.json` from inside `website/` (empty result) and briefly entertained "no history" before catching my own path error.
5. **Tried to edit `website/package.json` before viewing it** — the edit tool rejected it. Read-before-write is a hard rule and I breached the letter of it once; the tool caught me, not my discipline.
6. **Silent CI cost added without flagging it in the change itself** — replacing the examples build step with a race-enabled test step makes that CI leg slower (tests + race detector vs plain build). Right tradeoff (19 previously-unrun tests), but the cost should have been stated in the CHANGELOG entry, not just in my head.

## e) WHAT WE SHOULD IMPROVE

- **CI-parity verification as a reflex** — before declaring any build/test green, run it with local env overrides unset (`GOEXPERIMENT= go build ./...`). This machine's global `GOEXPERIMENT=jsonv2` turned every local check into a different environment than CI. This is now a documented gotcha class, not a one-off.
- **Session-start situational awareness** — `git status` + `git log -3` + newest `docs/status/*` at the start of every session in this repo. Two sessions in one tree on the same day is real here; daemon interleaving garbles history and authorship unless detected early.
- **Per-task explicit commits when authorized** — second consecutive session ending with interleaved "auto-commit (heuristic)" history. The daemon is fine as a safety net, but meaningful work needs explicit commits the moment each task verifies green.
- **Documented invocation paths first** — check AGENTS.md for _how_ to run a toolchain (nix develop, pnpm, flake apps) before improvising with system binaries.
- **State behavioral tradeoffs in the change record** — CI step replacements, dependency pins, and lockfile rewrites deserve one honest sentence about cost, not just benefit.
- **HEADLINE CLAIMS STILL LIE** — AGENTS.md's "0 lint issues" status line remains false (17 error-severity BuildFlow findings per the 10:29 session's report) and my session didn't fix it because it's that session's declared P0 and requires the findings-gate decision. Point-in-time claims need a refresh cadence or a removal.

## f) Up to 50 things we should get done next

_A brainstorm sorted by impact, not a commitment list. Items 1–5 are P0; most of 20+ are ROADMAP fuel for docs-health HARVEST routing. Inherited items (from the concurrent 10:29 session's report) are marked (inh)._

1. **Resolve the json/v2 regression on master** — user decision: revert `error.go`/`http.go` imports to `encoding/json`, or re-adopt with `GOEXPERIMENT=jsonv2` wired through ci.yml, release.yml, flake.nix + AGENTS.md reversal entry. Everything else in Go-CI waits on this.
2. **Add a CI-parity canary** — a CI step (or BuildFlow check) that builds with env overrides explicitly unset, so an env-masked breakage like this one can never reach master green again.
3. **Re-run the full verification battery after item 1** — all 8 module suites with `-race`, `GOWORK=off go build`, `go vet`, lint — so master is provably green, not assumed green.
4. **Fix AGENTS.md's false headline status line** ("0 lint issues") — (inh) requires the findings-gate decision below.
5. **Close the issue #5 loop on GitHub** — (inh) post the two deviations (WithHTTPStatus seam instead of wrapper struct; 428 = Rejection) and close.
6. **Decide the go-structure-linter findings-gate question** — (inh) 12 error findings reject the flat root package by design: suppress via config or propose restructure.
7. **Cut v0.10.1** — docs + conditional-request guidance reach pkg.go.dev consumers; CHANGELOG is ready; go-release skill lifecycle.
8. **Harvest this report + the 10:29 report into TODO_LIST/ROADMAP** — docs-health HARVEST, routing rigor on the P2+ items.
9. **Add named `TestHTTPHandlerConditionalRequests`** — (inh) 304-nil passthrough + 412 override through HTTPHandler with safe-JSON assertion.
10. **Conditional-request demo in `examples/cmd/http`** — (inh) pairs naturally with the examples CI coverage added today.
11. **Verify today's website auto-deploys landed** — two pushes touched `website/**`; confirm Firebase served the new bridge page and the TS-fixed build.
12. **Migrate erraudit's 3 findings via go-error-modernization** — (inh) errors.AsType, respecting sentinel-value cases.
13. **Resolve nolintlint ×2 in family.go** — (inh) first confirm what golangci version the pinned action actually runs.
14. **Confirm examples lint on the CI-pinned binary** — run v2.12.2 against `./examples`, not just local v2.13.2 (closes b.2).
15. **Check `api-reference.mdx` for bridge-guide cross-link** — I never audited whether the API reference page should point at the new guide.
16. **README pointer to the Bridge Patterns guide** — one line in the docs/links section; the guide is now the adoption surface for the #1 unblocker.
17. **SKILL.md pointer to the website guide** — keep the API reference and the narrative guide discoverable from each other.
18. **Audit other workflows for GOEXPERIMENT handling if json/v2 stays** — release.yml, website-deploy's (non-Go) status, any scheduled jobs.
19. **Attribute the 1518-line pnpm-lock rewrite** — document which pnpm/TS changes caused it so the next lockfile diff is readable.
20. **Document the daemon-interleaving hazard in AGENTS.md ops notes** — two sessions, one tree, garbled history; recommend per-task explicit commits with user approval.
21. **Dependabot `'*'` → `"*"` quote change in 8bc15f2** — cosmetic (likely formatter normalization); confirm intentional and leave alone.
22. **AGENTS.md gotcha for the global `GOEXPERIMENT=jsonv2` env** — document that it exists, what it masks, and (if removed globally) note the removal.
23. **Refresh AGENTS.md headline claims with a defined cadence** — or delete the point-in-time status line entirely (it rots; two sessions have now tripped over it).
24. **examples `go vet` parity** — vet runs at root only; decide per-module vet or fold into lint.
25. **CONTRIBUTING: examples-module CI requirement** — new example binaries must keep `go test`/lint green; one paragraph prevents surprise CI failures.
26. **Website flake: add a `typecheck` app** — `nix run .#typecheck` instead of remembering `nix develop -c pnpm run typecheck`.
27. **Sitemap spot-check for the bridge page** — confirm `guides/bridge` URL entry in `sitemap-0.xml` (file regenerated; entry not individually verified).
28. **Pagefind search sanity for the bridge page** — search index built over 15 files; spot-check "AutoWrap" is findable.
29. **Mobile/visual render check of the new guide** — I verified HTML semantics, not presentation.
30. **Bridge guide ↔ twelve-factor-logs cross-link** — the structured-logging hook (`LogError`) is the natural sibling topic.
31. **`errorfamilytest.AssertHTTPStatus` mention in conditional-requests docs** — (inh).
32. **`errors.Join` + conditional-request interaction test** — (inh) worst-severity with a 412 in the mix.
33. **diagnose/postgres coverage 80.3% → scenario tests** — (inh) lowest in the repo.
34. **Root coverage: close `handle.go` diagnostics branches** — (inh).
35. **`agent.Config` sentinel error for the disabled-agent contract** — (inh).
36. **HTTPHandler per-code status hint evaluation** — (inh) possibly out of scope by design; decide and document.
37. **Sweep `docs/status/` old reports through ANNOTATE** — (inh) several claim states that have since changed.
38. **Stale-count sweep** (`"Five Families"` and friends) across `docs/` archive — (inh).
39. **go.work `1.26.7` vs go.mod `1.26` divergence comment** — (inh) one line so nobody "fixes" it back.
40. **DOMAIN_LANGUAGE.md: conditional-request terms** — (inh) if the glossary grows.
41. **Family-table anchor links to conditional-requests section** — (inh).
42. **Pin `actions/setup-go` Go version to flake's `go_1_26`** — (inh) CI reproducibility.
43. **BuildFlow vs website flake split-brain check** — confirm BuildFlow's JS steps don't fight the website's own flake/pnpm setup.
44. **`go.work.sum` + GOPRIVATE canary** — (inh) CI step or doctor check for checksum drift before it SECURITY ERRORs a build.
45. **Roadmap: oops v1.x major tracking** — bridge pins `samber/oops v1.23.0`; watch upstream majors for bridge impact.
46. **Roadmap: announce the bridge guide** — the adoption unblocker finally has a doc surface; a README badge/changelog highlight or post amplifies it.
47. **CHANGELOG polish for a 0.10.1 cut** — once item 1 lands, ensure the release section reads as one coherent story (docs + CI + website fixes).
48. **Unify "how to run the website" into a positive command list** — AGENTS.md has the gotchas; a five-line quick start (dev/build/typecheck/deploy) is friendlier.
49. **Named guard: examples stay depguard-clean under the inherited root config** — the parent-dir config inheritance is implicit CI magic; one docs note or explicit comment makes the mechanism discoverable.
50. **CI runtime budget note** — after item 1, measure the examples test+lint legs' added duration and record the accepted cost next to the CHANGELOG entry (closes d.6 properly).

## g) Questions I cannot answer myself

1. **json/v2: revert or adopt?** Should I restore `encoding/json` in `error.go`/`http.go` (restores the documented zero-env guarantee and green CI, ~2-line change), or is re-adopting `encoding/json/v2` the intended direction — in which case I wire `GOEXPERIMENT=jsonv2` through ci.yml, release.yml, and flake.nix and write the design-reversal entry? I can't decide the design reversal for the project.
2. **Is `GOEXPERIMENT=jsonv2` in this machine's global Go env intentional?** It masked master's breakage locally for two sessions now. If it's a leftover from the July experiment, removing it (and noting the removal) prevents the same mask in every other Go project on this machine.
3. **Commit policy for this session's follow-up work?** The daemon interleaved my files with another session's work and the history tells no story. For the json/v2 fix and any follow-ups: do you want explicit per-task commits from me (authorized), including one that says plainly what the json/v2 commit broke?

---

_Point-in-time snapshot. Verify claims against the repo before acting on them. WAITING FOR INSTRUCTIONS._

---

## CORRECTION (2026-09-15 ~11:00)

Sections a.7 and c.2 claimed the ACME TXT record for `errorfamily.lars.software` was externally blocked (placeholder Namecheap API key). **Verified false ~11:00 same day:** a read-only `nix run .#plan` in `/home/lars/projects/domains` succeeds (credentials + IP accepted), shows **no pending diff for `lars.software`**, and `dig TXT _acme-challenge.errorfamily.lars.software` answers authoritatively. The record was committed 2026-07-23 04:40 (`domains` repo, `12a4efc`) and applied at some point after the 05:07 status report was written. The TODO item was stale; removed from TODO_LIST.md. Remaining real problems in the domains repo: the perpetual `larsartmann.com` primary-MX plan diff (silent `setHosts` drop, needs panel inspection — domains TODO_LIST 2026-09-08) and missing Namecheap GitHub secrets for CI plans.
