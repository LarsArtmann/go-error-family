# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and
traceable to its source. When an item ships, remove it here and record it in
`CHANGELOG.md` under the version it shipped in.

**Last updated:** 2026-09-27

---

## Active

1. **Announce the Bridge Patterns guide** (source: ROADMAP theme 4; carried
   from TODO_LIST 2026-09-18) — the classify→enrich→handle walkthrough shipped
   in v0.10.1 (`website/src/content/docs/guides/bridge.mdx`), but nothing has
   put it in front of consumers who use `samber/oops`. Draft a public
   announcement (r/golang post or a linkable short post) that leads with the
   three patterns and the decision guide. Bounded: one draft, review, publish.

2. **Retract the broken v0.6.0 tag family** (source: ROADMAP theme 3) — the
   phantom-`replace` v0.6.x tags still resolve from the module proxy and can
   be `go get`-ed by consumers. Add `retract` directives to `go.mod` (with
   reason comments pointing at the replacement version), release, and verify
   `go list -m -versions` shows them as retracted. Verified still open
   2026-09-27 (no `retract` in `go.mod`). Bounded: one go.mod edit + release +
   verification.

3. **Investigate why the Release workflow did not fire on the v0.10.2 tag
   push** (source: docs-health audit 2026-09-27) — `release.yml` triggers on
   `v[0-9]+.[0-9]+.[0-9]+*` tag pushes and ran for v0.10.1, but produced no
   run for v0.10.2 (tag exists, CI ran on the same commit, proxy indexed).
   The missing GitHub Release was created manually on 2026-09-27. Find the
   trigger cause (push mechanics? workflow file at tag ref?) so the next
   release cannot silently skip the gate. Bounded: one investigation + fix or
   documented acceptance.

4. **Add website guard canaries to CI** (source: docs/status/archived/2026-09-22 §f5–f8,
   §f13) — the TS-7 re-bump class has broken `website-deploy` three times and
   nothing structural guards it: (a) fail CI when `website/package.json`
   typescript major ≠ 6; (b) run `pnpm install --frozen-lockfile` + `astro
   check` for `website/**` changes; (c) decide the Dependabot security-update
   auto-run for `/website` (disable it or fix its pnpm handling) and keep the
   manual `nix develop -c pnpm audit` cadence until then. Bounded: one CI
   workflow edit + one settings decision.

5. **art-dupl: suppression/baseline + standing threshold policy** (source:
   docs/status/archived/2026-09-22 §f1–f2) — check whether art-dupl supports
   exclude/baseline patterns and wire the accepted `strTrue`/`strFalse` clone
   into it (AGENTS.md "do not fix again" is prose-only today); decide whether
   `-t 1` or `-t 5` is the routine gate and record the policy in AGENTS.md.
   Bounded: one capability check + one config/doc edit.

6. **Lift diagnose-family coverage** (source: docs/status/archived/2026-09-22 §f18–f19;
   baselines re-measured 2026-09-27) — `diagnose` core 84.2% → ≥90% (targeted
   tests on uncovered rule paths), `diagnose/postgres` 78.5% → ≥85%, and
   recover the `diagnose/git` regression 98.5% → 91.0% (new erraudit error
   branches from v0.10.1 are untested). Bounded: three test additions.

7. **Re-verify the standing claims battery** (source: docs/status/archived/2026-09-22
   §f15, §f20, §f38) — erraudit (0-findings claim), `go-structure-linter` CLI
   (exit 0 with the flat preset), a full `buildflow --build-mode full` run,
   and a `nix build` of `website/flake.nix`. These claims are asserted in
   AGENTS.md/FEATURES.md but decay silently. Bounded: four command runs.

8. **File the two upstream BuildFlow issues** (source: docs/status/archived/2026-09-22
   §f13–f14; verify-before-filing gate applies) — (a) `pnpm-audit` should
   discover subdirectory lockfiles (this repo's skip exists only until then);
   (b) branching-flow's phantom analyzer does not honor `IsIgnored` for
   `pkg/phantom`. Bounded: two reproductions + two filings.

9. **Website chores** (source: docs/status/archived/2026-09-22 §f30) — consider
   `minimumReleaseAgeStrict` for pnpm in `website/pnpm-workspace.yaml`
   (supply-chain freshness). (`bun.lock` gitignore already done — verified
   2026-09-27, `website/.gitignore:22`.) Bounded: one config decision.

10. **Plan v0.11.0 scope** (source: docs/status/archived/2026-09-18 §f40) — candidates
    from the backlog: example-coverage gaps (`errorfamilytest`, `diagnose`),
    coverage lifts, website canaries. Cut the CHANGELOG into a release plan.
    Bounded: one planning pass.
