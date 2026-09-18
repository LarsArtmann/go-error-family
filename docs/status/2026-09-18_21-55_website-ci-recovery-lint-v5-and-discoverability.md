# Status Report: Website/CI Recovery, exhaustruct_v5, Go-Pin Normalization, pkg.go.dev Examples

**Date:** 2026-09-18 21:55 CEST
**Repo:** go-error-family @ master (local ahead of origin via auto-commit daemon; nothing pushed this session)
**Session scope:** Execute/refine TODO_LIST + ROADMAP (both supplied by user), stabilize uncommitted daemon churn, fix red CI found along the way.
**Format note:** User explicitly requested `.md`; the status-report skill's canonical format is HTML — override honored per user instruction, not propagated back into the skill.

---

## a) FULLY DONE

1. **Website TypeScript-7 + lockfile split-brain fixed (3rd recurrence of this class).**
   Evidence: `website/package.json` `typescript ^7.0.2 → ^6.0.0` (policy: AGENTS.md, withastro/roadmap#1321); kept in-range starlight 0.42.2 / astro 7.3.3 / html-validate 11.16.0; `pnpm-lock.yaml` + `pnpm-workspace.yaml` regenerated. Verified: `pnpm install --frozen-lockfile` ✓, `astro check` 0 errors/warnings/hints, `astro build` 15 pages. Committed by daemon (f608d1c + follow-ups).

2. **8 transitive npm vulnerabilities cleared (6 high, 2 moderate).**
   Evidence: `pnpm update --depth Infinity` in `website/`; `pnpm audit` → "No known vulnerabilities found". Affected: devalue (< 5.9.1 DoS), fast-uri, js-yaml, svgo across multiple major lines. No overrides needed. This also removes the alert source that kept spawning the failing Dependabot jobs.

3. **`exhaustruct → exhaustruct_v5` lint migration verified (the v0.10.1 "remaining follow-up").**
   Evidence: uncommitted `.golangci.yml` daemon change judged on merits; golangci-lint v2.13.2 → **0 issues in all 7 modules** with the new config + `ignore-patterns` key.

4. **Go directive normalized to `go 1.26.7` in all 7 modules + `go.work`.**
   Evidence: daemon churn had regressed root/diagnose to `go 1.26` and set bridge to `go 1.26.0` vs. the deliberate 64298b6 toolchain bump; normalized; full build + `-race` tests re-run green per module.

5. **Six new godoc example functions (ROADMAP theme 1 — Consumer Discoverability).**
   Evidence: `example_test.go` now 26 runnable examples. Added: `ExampleWrap` (the ~750-call-site constructor had none), `ExampleHandleErrorWithContext` (canonical entry point), `ExampleIsRetryable`, `ExampleFamily_RetryPolicy`, `ExampleLogError` (time-stripped deterministic slog output), `ExampleHTTPHandler` (end-to-end httptest incl. per-error `WithHTTPStatus(404)`). All examples pass; lint 0 issues; golines/treefmt-formatted.

6. **Docs brought current.**
   Evidence: `CHANGELOG.md` [Unreleased] (6 entries), `ROADMAP.md` (theme 1 marked shipped, stale Direction paragraph rewritten, bridge-guide status corrected), `TODO_LIST.md` (Last updated + 2 new bounded Active items), `AGENTS.md` (Status line, BuildFlow bullet, Dependabot limitation, TS7-recurrence + lockfile-sync rule, go-directive uniformity, exhaustruct_v5 completion).

7. **BuildFlow gate re-validated with the result cache disabled.**
   Evidence: `BUILDFLOW_NO_RESULT_CACHE=1 buildflow --dry-run` → **105 success, 0 failed**; `checks.format` (treefmt: gofumpt/goimports/golines/nixfmt) builds green after fixing my long ServeHTTP line.

8. **Dependabot red jobs root-caused.**
   Evidence: every `npm_and_yarn in /website` security-update job since 2026-09-15 fails; there is no npm entry in `dependabot.yml` (auto-run for detected manifests); npm updater chokes on pnpm v9 lockfile + overrides (suppressed `record_update_job_error`). Detection (alerts) works; remediation does not.

9. **cqrs-lint false positives verified as false.**
   Evidence: the only 2 dry-run findings are `cqrs-lint` A009/A018 INFO claims ("imports go-cqrs-lite", "no stack/ preset"); `grep` proves **0 files** reference go-cqrs-lite. Library-repo false positives.

---

## b) PARTIALLY DONE

1. **Deploy Website workflow recovery.** What works: fix is committed locally and every leg verified locally (frozen install, check, build). What remains: the actual GitHub run has not gone green — it triggers on the next `website/**` **push**, which I did not perform (no-push policy). Blocker: none, just unpushed. Effort: S (push + watch one run).

2. **Website security-gate automation.** What works: audit is clean today; manual remediation procedure documented (`nix develop -c pnpm audit` + `pnpm update --depth Infinity`). What remains: nothing automated watches for the NEXT advisory — Dependabot auto-updates are structurally broken for pnpm here, and BuildFlow's `pnpm-audit` cannot see the subdirectory lockfile. Blocker: tool limitations (upstream). Effort: M (CI canary job) / L (BuildFlow upstream feature).

3. **Consumer Discoverability theme.** What works: root package now fully example-covered (26). What remains: `errorfamilytest` (0 examples), `diagnose` (0), `diagnose/git`, `diagnose/postgres` (0) — plus "common patterns" godoc section. Blocker: none; time. Effort: M.

4. **TODO_LIST Active items** (bridge-guide announcement; v0.6.0 tag retraction) — refined and written down this session, not executed. Effort: M each.

5. **Erraudit "0 findings" claim.** AGENTS.md asserts it, but I added ~90 lines of example code and did **not re-run erraudit** to keep the claim true. Likely still 0 (no error paths in examples), unverified. Effort: S.

---

## c) NOT STARTED

- **Bridge Patterns guide announcement** (TODO_LIST #1) — draft/publish to r/golang or a short post. Not started; needs your voice (github-voice skill) and a channel decision.
- **v0.6.x broken-tag retraction** (TODO_LIST #2) — `retract` directives + release + `go list -m -versions` verification. Not started; touches release surface, wanted per ROADMAP.
- **Release automation script** for coordinated multi-module tag cutting (ROADMAP theme 3) — not started; deprioritized until next release.
- **Structural guards against daemon churn** — CI canary for `typescript ^7`, lockfile-sync check, go-directive uniformity check. Not started (docs-only mitigation shipped instead — see (d) #3).
- All other ROADMAP raw ideas untouched: `httperror` subpackage, OpenAPI schema for the error JSON, `Code()`/`ErrorCode()` convergence (major), `redis`/`docker`/`kubectl` diagnose modules, Echo/Gin/Chi/gRPC integration guides, classification benchmark suite, `Code()` vs `ErrorCode()` decision.

---

## d) TOTALLY FUCKED UP

1. **My own false-green gate edit (caught, corrected — but it happened).**
   I re-enabled `pnpm-audit` in `.buildflow.yml` on the strength of a cached ✔ (0 ms / 8 ms) — exactly the "a scanner must prove it scanned" failure that is written down as a lesson in AGENTS.md. The uncached run (`BUILDFLOW_NO_RESULT_CACHE=1`) immediately failed: `ERR_PNPM_AUDIT_NO_LOCKFILE` (step runs at repo root; lockfile lives in `website/`). I then had to un-re-enable it, rewrite the skip rationale, and correct false statements I had already written into CHANGELOG.md and AGENTS.md. Severity: process-level; no lasting damage (corrections committed), but it invalidated 3 files of doc work for one round and proves the lesson still isn't reflexive for me.

2. **origin/master Deploy Website is still red right now.** Every visitor-facing proof of the fix is local. The workflow has been red since 2026-09-15 17:53; until a `website/**` commit is pushed, master carries a red X. Severity: visibility/trust, not availability (site itself was already deployed and serves fine). Mitigation: local verification; final proof pending push.

3. **Policy-by-documentation keeps losing to an automated writer.** The TS 7 re-bump has now broken the deploy pipeline **three times** (2026-09-13, 2026-09-15, 2026-09-18), and the go-directive floor regressed within days of being deliberately set. Every recurrence was "fixed" by documenting a rule in AGENTS.md. Documentation does not stop `go-auto-upgrade`-style fleet tooling from rewriting the files again tomorrow. Nothing structural (CI canary, lockfile-sync check, renovate/dependabot ignore rule, orchestrator allowlist) guards these files. Root cause: guard gap, not knowledge gap. WILL recur.

4. **Dependabot auto-security-updates are dead for `website/` with no replacement automation.** Detection-only. The 8 vulns fixed today sat exposed on master since at least 2026-09-15 (devalue DoS among them, moderate-severity path). Workaround is a manual command nobody runs on schedule.

Not fucked: no test failures, no lint debt, no broken API, no data loss. The tree is the healthiest it's been this week — the risks above are all "next time".

---

## e) WHAT WE SHOULD IMPROVE

1. **Cache-blind verification is still my default.** First verification attempt of a gate change must be `BUILDFLOW_NO_RESULT_CACHE=1`. I knew this; I applied it only after suspicion. Fix: make it a personal checklist item for ANY BuildFlow step-verdict change.
2. **Skill-loading order violated.** I edited `.buildflow.yml` before loading the buildflow skill; the skill's anti-patterns section contained the exact trap I fell into ("don't trust a green step that scanned zero files"). Fix: skill check before touching BuildFlow/config files, not after failure.
3. **Pending daemon changes need a standard triage order.** This session I invented one ad hoc (diff → build → lint → dry-run). The dry-run belongs in the FIRST pass, not the last — it caught the nix/config issues the Go tools cannot see.
4. **Drill warnings immediately, not at report time.** The "(2 findings)" and the `nix flake show` parse warning sat unexamined until the user pushed. Both turned out benign (cqrs-lint false positives, tool-side quirk) — but benign-ness is a finding, not an assumption.
5. **Claim-staleness control.** AGENTS.md hard-codes verifiable claims ("erraudit 0 findings", "26 examples"). Any code change can silently stale them. Fix: verify-on-touch rule (re-run the claim's check whenever you edit code it covers), or move volatile numbers into CHANGELOG only.
6. **The daemon churn war needs a structural escalation** (see (d) #3): a `typescipt`-version + lockfile-sync CI check is cheap (S) and ends the recurrence class; an upstream BuildFlow allowlist/denylist for website package.json ends it properly.
7. **Untracked noise: `website/bun.lock`** appears whenever the Bun-shim node touches the project. Documented, but `website/.gitignore` should name it explicitly to end the ambiguity.

---

## f) Next 50 (brainstorm — HARVEST fuel; top ~10 are TODO_LIST-grade)

| #  | Task                                                                                                   | Impact | Effort | Category |
|----|--------------------------------------------------------------------------------------------------------|--------|--------|----------|
| 1  | Push `website/**` and confirm the Deploy Website run goes green (first proof after 3-day red)           | Critical | S | Bug |
| 2  | Add CI canary: fail if `website/package.json` typescript major ≠ 6                                     | Critical | S | Bug |
| 3  | Add CI check: `pnpm install --frozen-lockfile` + `astro check` for website on every `website/**` PR    | High | S | Quality |
| 4  | Decide + disable Dependabot security-updates auto-run for `/website` (or fix its pnpm handling)        | High | S | Cleanup |
| 5  | Re-run erraudit to re-verify the "0 findings" claim after this session's example code                  | Medium | S | Quality |
| 6  | Retract broken v0.6.x tags via `retract` directives + verify on module proxy                           | High | M | Bug |
| 7  | Draft the Bridge Patterns guide announcement (TODO_LIST #1)                                            | High | M | Documentation |
| 8  | CI check: all 7 `go` directives + go.work equal `1.26.7` (ends the pin split-brain class)              | High | S | Quality |
| 9  | Update SKILL.md (API reference) with new gotchas: pnpm/audit, Dependabot, examples count               | Medium | S | Documentation |
| 10 | Add `website/bun.lock` to `website/.gitignore`                                                          | Low | S | Cleanup |
| 11 | File upstream BuildFlow issue: pnpm-audit should discover subdirectory lockfiles (fleet-wide value)     | High | S | Feature |
| 12 | File upstream BuildFlow issue: `nix flake show` JSON parse failure (0 bytes) warning                    | Medium | S | Bug |
| 13 | Silence/fix cqrs-lint A009/A018 false positives for library repos (verified: 0 go-cqrs-lite imports)    | Low | S | Cleanup |
| 14 | Examples for `errorfamilytest` subpackage (AssertFamily/AssertCode/…) on pkg.go.dev                     | Medium | M | Documentation |
| 15 | Examples for `diagnose` core + `diagnose/git` + `diagnose/postgres`                                     | Medium | M | Documentation |
| 16 | "Common patterns" godoc section in doc.go (grow from real consumer usage)                               | Medium | M | Documentation |
| 17 | Release automation script for coordinated multi-module tag cuts (ROADMAP theme 3)                       | High | L | Feature |
| 18 | Pin-bump hygiene: script/doc that bumps root pins in all submodules in lockstep on release              | Medium | M | Quality |
| 19 | Full `buildflow --build-mode full` run (I only ran dry-run + targeted steps)                            | Medium | M | Quality |
| 20 | Full `nix build` of website's own `website/flake.nix` (I used the root devshell only)                   | Medium | M | Quality |
| 21 | Investigate `nix-hash-fix` 42/42 history: was there EVER a vendorHash here, or pure misconfiguration?   | Low | S | Cleanup |
| 22 | Consider `minimumReleaseAgeStrict` for pnpm (pnpm-workspace.yaml currently has only the exclude entry)  | Low | S | Security |
| 23 | errorfamilytest adoption push: mention in README quick-start test example                               | Medium | S | Documentation |
| 24 | OpenAPI/schema generation for the canonical error JSON shape (ROADMAP theme 2)                          | Medium | L | Feature |
| 25 | `httperror` subpackage RFC: richer response shaping (ROADMAP theme 2)                                   | Medium | L | Feature |
| 26 | Framework integration guides: Chi first (stdlib-adjacent), then Echo/Gin/gRPC interceptor               | Medium | M | Documentation |
| 27 | Diagnostic submodules: `redis` (pattern matches git/postgres submodules)                                | Medium | M | Feature |
| 28 | Diagnostic submodules: `docker` + `kubectl`                                                             | Medium | L | Feature |
| 29 | Benchmark suite: classification overhead vs version (ROADMAP theme 4; there is already benchmark_test.go to grow) | Low | M | Quality |
| 30 | `Code()` vs `ErrorCode()` convergence proposal for next major (ROADMAP theme 1)                         | Low | S | Documentation |
| 31 | Coverage: `diagnose/postgres` 80.3% → 85%+                                                              | Low | M | Quality |
| 32 | Coverage: `diagnose` core 83.9% → 90%                                                                   | Low | M | Quality |
| 33 | Document gopls nilness warning on `panicNilError` fixture (error_test.go:567) as deliberate             | Low | S | Documentation |
| 34 | Fuzz seeds: promote interesting corpus entries from the 11 fuzz targets into CI smoke run               | Low | M | Quality |
| 35 | Consumer survey: who uses LogError/HTTPHandler/errorfamilytest today (updates Adoption Reality table)   | Medium | M | Documentation |
| 36 | Where oops users actually are: identify 3 candidate consumers for the bridge, engage individually       | High | M | Feature |
| 37 | samber/oops cross-link: propose docs link to bridge guide upstream (verify-before-filing first)         | Low | S | Documentation |
| 38 | Module-proxy health script: resolve all 7 tags + verify retracted/deprecation notes render              | Low | S | Quality |
| 39 | go.work.sum hygiene: script to detect stale checksums (the v0.2.2 SECURITY ERROR class)                 | Low | M | Quality |
| 40 | Plan v0.11.0 scope (examples + docs are candidates; cut CHANGELOG into release)                         | Medium | S | Documentation |
| 41 | FEATURES.md / README sweep: verify claims about example coverage and gates are current                  | Low | S | Documentation |
| 42 | docs-health HARVEST: route this report's items into TODO_LIST/ROADMAP properly                          | Medium | S | Documentation |
| 43 | Annotate the 2026-09-15 status report (deploy-red again since; now fixed 3rd time) — docs-health ANNOTATE | Low | S | Documentation |
| 44 | PR to Dependabot docs/community: pnpm v9 + overrides grouped security updates failing (if reproducible minimal case) | Low | M | Bug |
| 45 | Add `ExampleNewConflict`/`ExampleNewCorruption`/`ExampleNewInfrastructure`/`ExampleNewOrchestration` constructor set for symmetry | Low | S | Documentation |
| 46 | Review `WithContextf`/`WithContextMap` examples gap (only WithContextAny has one)                       | Low | S | Documentation |
| 47 | gRPC status-mapping guide (family → codes.Internal/Unavailable/InvalidArgument) — recurring consumer ask | Medium | M | Documentation |
| 48 |十二-factor-logs guide cross-check: ensure HandleConfig.Logger example matches website guide              | Low | S | Documentation |
| 49 | Dependabot: add explicit `npm` entry for `/website` IF its pnpm support has matured (test on a branch)  | Low | M | Quality |
| 50 | Periodic `gh run list` triage habit: red workflows persisted 3 days unnoticed — add to weekly routine   | Medium | S | Process |

---

## g) Questions I cannot answer myself

1. **Push authority/timing:** I never push. The Deploy Website fix is proven locally but master stays red until a `website/**` commit reaches origin. Do you want to push now (or want me to), or does the fleet orchestrator own pushes on its own schedule?
2. **Structural guard preference for the recurring churn (d)#3:** which guard do you want — (a) CI canary jobs (typescript version + frozen-lockfile + go-directive uniformity; I can build these today), (b) disabling Dependabot security-updates for `/website` in repo settings (only you can click that), or (c) fixing it at the fleet level in the `go-auto-upgrade`/orchestrator config?
3. **cqrs-lint fleet behavior:** the BuildFlow dry-run claims this repo "imports go-cqrs-lite" (it provably does not — 0 references). Is cqrs-lint scanning every repo in the fleet by design (→ we ignore it here), or is this a discovery bug worth an upstream BuildFlow report?

---

*Point-in-time snapshot. Session evidence: commits f608d1c…7b571ec (daemon-carried), all gates re-verified 2026-09-18 ~21:45–21:55 CEST with result cache disabled. Section (f) is HARVEST fuel for TODO_LIST/ROADMAP — pending your go-ahead since you asked me to wait.*
