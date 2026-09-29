# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and
traceable to its source. When an item ships, remove it here and record it in
`CHANGELOG.md` under the version it shipped in.

**Last updated:** 2026-09-29

---

## Active

1. **Announce the Bridge Patterns guide** (source: ROADMAP theme 4) —
   REOPENED 2026-09-29: the GitHub Discussion venue was tried and withdrawn
   (a link-post in an otherwise empty Discussions tab serves no purpose;
   Discussions disabled again). The draft
   (`docs/planning/2026-09-28_bridge-patterns-announcement-draft.md`) is
   voice-checker-clean and self-reviewed; pick a channel with real audience
   (r/golang candidate) before publishing. Bounded: one channel decision +
   one publish.
2. **Heal the v0.11.0-tagged README on pkg.go.dev** (source: 2026-09-29
   docs-health pass) — the tag was cut before the announcement-withdrawal
   commit (`986a6a6`), so the README rendered at `@v0.11.0` still links the
   deleted Discussion #12. Master is already clean; any next release heals
   it. Bounded: decide standalone v0.11.1 vs ride-along on the next release.
3. **Review the post-v0.11.0 Dependabot wave** (source: 2026-09-29 report
   §f20) — submodule root-pin bumps (agent, bridge, diagnose, diagnose/git,
   diagnose/postgres, examples → v0.11.0) arrive on the weekly schedule;
   merge the green ones or hold for the next release. Bounded: one triage
   pass.
4. **Enforce the art-dupl baseline in CI** (source: 2026-09-28 Pareto report
   §f5) — `.art-dupl-baseline.json` is committed but nothing runs
   `art-dupl check -t 1 .` in CI; a baseline nobody executes is prose, not a
   gate. Bounded: one `ci.yml` job.
