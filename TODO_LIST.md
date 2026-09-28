# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and
traceable to its source. When an item ships, remove it here and record it in
`CHANGELOG.md` under the version it shipped in.

**Last updated:** 2026-09-29

---

## Active

None. The 2026-09-27/28 batch shipped in full (v0.11.0, 2026-09-29):

1. ~~Announce the Bridge Patterns guide~~ — **DONE 2026-09-28:**
   [Discussion #12](https://github.com/LarsArtmann/go-error-family/discussions/12)
   (Announcements; repo Discussions enabled for it), cross-linked from README.
   r/golang adaptation deferred until adoption signal.
2. ~~Website guard canaries in CI~~ — **DONE 2026-09-28:** `website-check.yml`
   (TS-major==6, frozen-lockfile install, `astro check`+`build`); Dependabot
   decision recorded in AGENTS.md (security updates stay enabled; red jobs are
   the alert signal — no per-directory toggle exists).
3. ~~art-dupl baseline + threshold policy~~ — **DONE 2026-09-28:**
   `.art-dupl-baseline.json` (threshold 1, 3 accepted groups); policy recorded
   in AGENTS.md, ROADMAP OQ1 resolved (routine `-t 5`, deep sweeps `-t 1`).
4. ~~Lift diagnose-family coverage~~ — **DONE 2026-09-28:** diagnose 97.4%,
   diagnose/git 98.7%, diagnose/postgres 89.2% — all targets exceeded.
5. ~~Re-verify the standing claims battery~~ — **DONE 2026-09-29:** erraudit
   0 findings ×7 modules, `go-structure-linter` exit 0 (flat preset),
   `buildflow --build-mode full` exit 0 (115 success / 0 failed), website
   frozen install + `astro check` + `astro build` green (16 pages).
6. ~~File the two upstream BuildFlow issues~~ — **DONE 2026-09-28:**
   BuildFlow #23 filed (phantom `IsIgnored` unwired in `pkg/phantom`);
   evidence comment on #19 (pnpm-audit lockfile discovery).
7. ~~Website chores~~ — **DONE 2026-09-28** (`minimumReleaseAgeStrict: true`,
   install + check green; `bun.lock` gitignore verified 2026-09-27).
8. ~~Cut v0.11.0~~ — **DONE 2026-09-29:** see `CHANGELOG.md` [0.11.0]
   (gRPC guide landed, changelogs synced, tags per runbook, notes curated).
