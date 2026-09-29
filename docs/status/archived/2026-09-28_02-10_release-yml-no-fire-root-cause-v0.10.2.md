# Root Cause: release.yml Never Fired on v0.10.2

**Date:** 2026-09-28
**Status:** RESOLVED — root cause identified, mitigations landed (workflow_dispatch fallback + runbook).
**Resolves:** ~~TODO_LIST #3 (docs-health audit 2026-09-27), plan task M02.~~ done — closed 2026-09-28; fallback live-tested on v0.10.3 and again on v0.11.0 (2026-09-29); runbook in AGENTS.md.

## Symptom

Tag `v0.10.2` (commit `aefb86a`, pushed 2026-09-22T20:33Z) produced **no** `Release`
workflow run. The GitHub Release had to be created manually on 2026-09-27.
`v0.10.1` (commit `1d629ba`, 2026-09-15) fired normally.

## Evidence

| Check                                   | v0.10.1                                         | v0.10.2                                                     |
| --------------------------------------- | ----------------------------------------------- | ----------------------------------------------------------- |
| `release.yml` at tag                    | trigger `push: tags: v[0-9]+.[0-9]+.[0-9]+*`    | **byte-identical** (empty diff)                             |
| Tag glob match                          | yes                                             | yes                                                         |
| Tag exists on origin, SHA matches local | yes                                             | yes (annotated tag `fa896f4`)                               |
| Release run                             | `35001405325`, `head_branch=v0.10.1`, 17:27:42Z | **none** (not even startup_failure)                         |
| CI run on same commit                   | 17:27:44Z (master)                              | 20:33:37Z (master)                                          |
| PushEvent `refs/heads/master`           | yes                                             | yes (20:33:36Z, head `aefb86a`)                             |
| PushEvent `refs/tags/v0.10.2`           | (equivalent fired for v0.10.1)                  | **ABSENT** from the events API across the whole push window |

The `refs/tags/v0.10.2` push webhook was **never delivered**, so the `push`-triggered
release workflow was never evaluated. The commit-level CI ran fine because the
branch push event went through at the same moment.

## Root Cause

**Push-mechanics loss of the tag ref event, not the workflow.** The tag points at
the exact commit that was the master head in the same push window — the known
GitHub Actions behavior where a combined `git push origin master <tag>` (or
`--tags`) coalesces/drops the tag-ref `push` webhook when the tag refpoints at the
branch head (community-documented race; delivery is not guaranteed for both refs).
v0.10.1 shows both events CAN be delivered from a combined push (Release ran 2s
before CI) — i.e. the race is flaky, which matches "worked at v0.10.1, silently
dropped at v0.10.2". (The other classic cause — tag created via API/UI, which
never emits a push webhook — is not what happened here: the tag exists as a real
git annotated tag pushed from a clone.)

## Mitigations (landed)

1. **`workflow_dispatch` fallback** in `release.yml`: input `tag`, force-fetches
   and checks out the tag, and `softprops/action-gh-release` pins
   `tag_name` to the resolved tag. A swallowed tag event can now be re-run from
   the Actions UI without re-pointing anything.
2. **Release runbook (AGENTS.md):** verify the Release run starts within ~2 min of
   the tag push; **push tags in a separate `git push` from the branch**; if the
   run is missing, dispatch the workflow with the tag name.

## Verification

- `gh api .../actions/runs?head_sha=<v0.10.2 sha>` → CI + Dependabot graph runs only, no Release.
- `gh api .../events` (20:00–21:00Z window) → two `refs/heads/master` PushEvents, zero tag PushEvents.
- `git diff v0.10.1 v0.10.2 -- .github/workflows/release.yml` → empty.
- Fallback path exercised on the next release (v0.10.3 retraction release, M04):
  tags pushed separately from the branch; Release run confirmed within ~2 min —
  and repeated on v0.11.0 (2026-09-29).
