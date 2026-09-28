# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and
traceable to its source. When an item ships, remove it here and record it in
`CHANGELOG.md` under the version it shipped in.

**Last updated:** 2026-09-28

---

## Active

1. **Announce the Bridge Patterns guide** (source: ROADMAP theme 4) —
   DRAFTED 2026-09-28 (`docs/planning/2026-09-28_bridge-patterns-announcement-draft.md`,
   self-reviewed against the guide). Remaining: user review, channel pick
   (GitHub Discussion vs r/golang), publish, cross-link from README/related-tools.

2. **Add website guard canaries to CI** (source: docs/status/archived/2026-09-22 §f5–f8,
   §f13) — the TS-7 re-bump class has broken `website-deploy` three times and
   nothing structural guards it: (a) fail CI when `website/package.json`
   typescript major ≠ 6; (b) run `pnpm install --frozen-lockfile` + `astro
   check` for `website/**` changes; (c) decide the Dependabot security-update
   auto-run for `/website` (disable it or fix its pnpm handling) and keep the
   manual `nix develop -c pnpm audit` cadence until then. Bounded: one CI
   workflow edit + one settings decision.

3. **art-dupl: suppression/baseline + standing threshold policy** (source:
   docs/status/archived/2026-09-22 §f1–f2) — check whether art-dupl supports
   exclude/baseline patterns and wire the accepted `strTrue`/`strFalse` clone
   into it (AGENTS.md "do not fix again" is prose-only today); decide whether
   `-t 1` or `-t 5` is the routine gate and record the policy in AGENTS.md.
   Bounded: one capability check + one config/doc edit.

4. **Lift diagnose-family coverage** (source: docs/status/archived/2026-09-22 §f18–f19;
   baselines re-measured 2026-09-27) — `diagnose` core 84.2% → ≥90% (targeted
   tests on uncovered rule paths), `diagnose/postgres` 78.5% → ≥85%, and
   recover the `diagnose/git` regression 98.5% → 91.0% (new erraudit error
   branches from v0.10.1 are untested). Bounded: three test additions.

5. **Re-verify the standing claims battery** (source: docs/status/archived/2026-09-22
   §f15, §f20, §f38) — erraudit (0-findings claim), `go-structure-linter` CLI
   (exit 0 with the flat preset), a full `buildflow --build-mode full` run,
   and a `nix build` of `website/flake.nix`. These claims are asserted in
   AGENTS.md/FEATURES.md but decay silently. Bounded: four command runs.

6. **File the two upstream BuildFlow issues** (source: docs/status/archived/2026-09-22
   §f13–f14; verify-before-filing gate applies) — (a) `pnpm-audit` should
   discover subdirectory lockfiles (this repo's skip exists only until then);
   (b) branching-flow's phantom analyzer does not honor `IsIgnored` for
   `pkg/phantom`. Bounded: two reproductions + two filings.

7. **Website chores** (source: docs/status/archived/2026-09-22 §f30) — consider
   `minimumReleaseAgeStrict` for pnpm in `website/pnpm-workspace.yaml`
   (supply-chain freshness). (`bun.lock` gitignore already done — verified
   2026-09-27, `website/.gitignore:22`.) Bounded: one config decision.
   DONE 2026-09-28 (`minimumReleaseAgeStrict: true`, install + check green).

8. **Cut v0.11.0** (scope frozen 2026-09-28, see CHANGELOG `[Unreleased]`) —
   discoverability release: constructor examples + errorfamilytest/diagnose
   examples + gRPC guide + release-engineering guards. Checklist: (a) land the
   gRPC guide (`website/src/content/docs/guides/grpc.mdx` + sidebar) and
   verify `astro check`/`astro build`; (b) sync the website changelog; (c)
   finalize the CHANGELOG section with the release date; (d) workspace tests
   - lint green (all 7 modules, `-race`); (e) follow the AGENTS release
     runbook (separate tag pushes, ~2 min Release-run check, dispatch fallback
     ready); (f) curate the GitHub Release notes (v0.10.2 style). Bounded:
     one guide + one coordinated release.
