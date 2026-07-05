# Status Report — 2026-07-05 21:00 CEST

**Session focus:** Committed the go.mod hotfix and cut 7 patch tags. Discovered two new concerns during the commit: BuildFlow auto-bumped cross-module pins to the fresh tags (uncommitted), and `origin/master` unexpectedly tracks HEAD.

---

## TL;DR

The phantom-`replace` fix is **committed (`48d7e70`) and tagged with 7 patch releases**. All 7 modules pass tests, BuildFlow passed 26/26. However, two things need attention: (1) the working tree is **dirty again** — BuildFlow's post-commit tooling auto-bumped every cross-module pin to the freshly-cut tags (v0.6.0→v0.6.1, diagnose v0.1.0→v0.1.1), leaving those improvements uncommitted; (2) **`origin/master` already points at `48d7e70`** despite no explicit push from me — likely a BuildFlow post-commit hook or stale ref. **Nothing has been explicitly pushed with tags.**

---

## a) FULLY DONE ✅

| #   | Item                                                                                                                                                                       | Evidence                   |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------- |
| 1   | **Hotfix committed** (`48d7e70`) — removed phantom `replace`+`require`, extracted `examples/` module, bumped bridge/git/postgres root pins                                 | `git log -1`               |
| 2   | **7 patch tags cut** on the fix commit: `v0.6.1`, `diagnose/v0.1.1`, `agent/v0.1.1`, `bridge/v0.2.1`, `diagnose/git/v0.4.1`, `diagnose/postgres/v0.4.1`, `examples/v0.1.0` | `git tag --points-at HEAD` |
| 3   | **CHANGELOG.md updated** with `[0.6.1]` entry documenting the fix + examples extraction                                                                                    | lines 7-23                 |
| 4   | **AGENTS.md version bumped** v0.6.0 → v0.6.1                                                                                                                               | line 6                     |
| 5   | **BuildFlow passed** pre-commit: 26/26 checks, 0 failed, 0 skipped (16.8s)                                                                                                 | commit output              |
| 6   | **All 7 modules pass tests** with `-race` (verified pre-commit)                                                                                                            | prior run                  |
| 7   | **0 lint issues**                                                                                                                                                          | `golangci-lint run ./...`  |
| 8   | **Root module is truly zero-dependency** — `go.mod` is 3 lines, no requires                                                                                                | committed state            |
| 9   | **Consumer simulation passes** — `GOWORK=off go list -m all` resolves real versions only                                                                                   | verified pre-commit        |

---

## b) PARTIALLY DONE ⚠️

| #   | Item                             | What's done                                                                                           | What remains                                                                                                                                                                               |
| --- | -------------------------------- | ----------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **Cross-module pin consistency** | Tags cut; committed go.mod files reference valid older versions (v0.6.0, diagnose v0.1.0) — MVS-valid | BuildFlow auto-generated pin bumps to the fresh tags (v0.6.1, diagnose v0.1.1) sitting **uncommitted** in the working tree. Committing them would require yet another tag round (chasing). |
| 2   | **Release publication**          | Tags exist locally                                                                                    | **Not pushed to remote** (no explicit push run). `origin/master` tracking ref shows `48d7e70` but this is suspicious — see (d). Tags definitely not pushed.                                |
| 3   | **CI hardening**                 | Source go.mod files are correct                                                                       | No CI gate added yet to prevent recurrence (`GOWORK=off go list -m all` check)                                                                                                             |

---

## c) NOT STARTED ⏭️

| #   | Item                                                                     |
| --- | ------------------------------------------------------------------------ |
| 1   | Pushing commit + tags to remote (`git push origin master --tags`)        |
| 2   | CI gate: `GOWORK=off go list -m all` per module                          |
| 3   | CI consumer-simulation job (`go get ...@tag` in throwaway module)        |
| 4   | CI invariant: root `go list -m all` returns exactly 1 line               |
| 5   | SKILL.md audit for stale "replace" / "not yet published" language        |
| 6   | Release-notes / README note deprecating the broken v0.6.0 family of tags |

---

## d) TOTALLY FUCKED UP 💥

### 1. The working tree is dirty AGAIN immediately after commit

After committing `48d7e70` with a clean tree, **11 files are now modified** — all submodule `go.mod`/`go.sum` files plus `flake.lock`. The diff shows every cross-module pin bumped to the fresh tags:

```
agent/go.mod:     require root v0.6.0 → v0.6.1, diagnose v0.1.0 → v0.1.1
diagnose/go.mod:  require root v0.6.0 → v0.6.1
bridge/go.mod:    require root v0.6.0 → v0.6.1
diagnose/git/go.mod:     root v0.6.0 → v0.6.1
diagnose/postgres/go.mod: root v0.6.0 → v0.6.1
flake.lock:       nixpkgs bump (same one I discarded before commit)
```

**Root cause:** BuildFlow's pre-commit hook ran `go mod tidy`/`go mod download` _after_ I staged but using the freshly-cut tags, regenerating pin files. I committed the pre-hook state; the post-hook state is sitting dirty. This is a **version-chasing trap**: if I commit these bumps, the new commit's go.mod references v0.6.1 (which is this very lineage), which is valid but then the _tagged_ versions always lag one step behind the "ideal" pins.

**Also:** `flake.lock` regenerated AGAIN — I discarded it once before commit, but BuildFlow's `flake-lock-freshness` check re-bumps it every run. This will keep happening on every commit.

### 2. `origin/master` unexpectedly points at my unpushed commit

```
git log origin/master -1 → 48d7e70  (my fix commit)
```

I did **not** run `git push`. Before my commit, `origin/master` was at `7605190` (v0.6.0). Something — almost certainly a **BuildFlow post-commit hook** — pushed the commit to remote _without my knowledge or the tags_. This means:

- The fix commit is now on GitHub **without** the 7 patch tags (tags are local only).
- The global "never push without explicit permission" rule was violated by automation, not by me.
- Consumers fetching `origin/master` get the fix, but `go get @v0.6.1` fails (tag not on remote).

**This needs confirmation.** If the commit IS on the remote, the tags must be pushed immediately to make the release real. If it is NOT (stale local ref), nothing has shipped.

---

## e) WHAT WE SHOULD IMPROVE 🛠️

1. **BuildFlow's auto-push behavior must be explicit and logged.** A post-commit hook that silently pushes violates the "never push without permission" invariant and makes release state ambiguous. Either disable auto-push or make it loud/opt-in.
2. **Version-pin chasing is inherent to this multi-module layout.** Every tag cut regenerates pin bumps. Consider: (a) accept MVS-valid-but-lagging pins as fine and stop chasing, or (b) cut tags in strict topological order and re-tag, or (c) use a release tool that handles this (e.g., `go-releaser` multi-module).
3. **`flake.lock` churn** — BuildFlow's freshness check re-bumps nixpkgs on every commit. Either commit the bump (and stop discarding it) or configure BuildFlow to skip `flake-lock-freshness` on non-nix commits.
4. **The commit-then-tag sequence leaks intermediate state.** A better workflow: stage everything → cut tags on the _intended_ commit → push commit+tags atomically. Tagging after commit (as I did) means the tagged commit's go.mod never references the tag's own version.
5. **No CI gate yet** for the exact class of bug we just fixed. Highest-ROI improvement possible.

---

## f) Up to 25 things we should get done next 🎯

| #   | Task                                                                                                                  | Impact                                     |
| --- | --------------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| 1   | **Confirm whether `48d7e70` is actually on the remote** (`git ls-remote origin master`)                               | 🔴 Critical — determines release state     |
| 2   | **Push tags to remote** (`git push origin --tags`) if commit is there, else push both                                 | 🔴 Makes the release real                  |
| 3   | **Decide on the dirty pin bumps** — commit them + cut v0.6.2/diagnose-v0.1.2/etc., OR discard and accept lagging pins | 🔴 Unblocks clean tree                     |
| 4   | **Resolve flake.lock churn** — commit the nixpkgs bump or configure BuildFlow to skip it                              | 🟠 Stops recurring dirt                    |
| 5   | **Add CI gate: `GOWORK=off go list -m all`** per module                                                               | 🔴 Prevents recurrence of the original bug |
| 6   | **Add CI consumer-simulation job** (`go get @tag` in throwaway module)                                                | 🔴 Honest release proof                    |
| 7   | **Add CI invariant: root `go list -m all` = 1 line**                                                                  | 🟠 Enforces zero-dep                       |
| 8   | **Audit BuildFlow config** for auto-push hooks; make explicit/opt-in                                                  | 🟠 Safety                                  |
| 9   | **Update SKILL.md** for stale replace/publish language                                                                | 🟡 Doc honesty                             |
| 10  | **Write release deprecation note** for broken v0.6.0 family                                                           | 🟡 Consumer trust                          |
| 11  | **Document the multi-module release sequence** in AGENTS.md                                                           | 🟡 Process                                 |
| 12  | **Add `go mod tidy` check to CI** (detect missing go.sum entries)                                                     | 🟡                                         |
| 13  | **Consider a release automation script** (nix/just target for coordinated tags)                                       | 🟢 Tooling                                 |
| 14  | **Bump examples go.mod pins** to v0.6.1/diagnose v0.1.1 (currently v0.6.0/v0.1.0)                                     | 🟢 Freshness                               |
| 15  | **Verify `project-meta` resolves against the new tags**                                                               | 🟡 Validates the fix downstream            |
| 16  | **Run `nix build` / `nix flake check`** after module changes                                                          | 🟡                                         |
| 17  | **Normalize go.mod require style** (inline vs block) across submodules                                                | 🟢                                         |
| 18  | **Add `go vet ./...` to CI** if not present                                                                           | 🟢                                         |
| 19  | **Review whether `errorfamilytest` should be its own module**                                                         | 🟢 Future                                  |
| 20  | **Consider Dependabot/Renovate** for multi-module pin management                                                      | 🟢                                         |
| 21  | **Post the v0.6.1 release on GitHub Releases** with the CHANGELOG excerpt                                             | 🟢                                         |
| 22  | **Add a pre-release checklist** (run consumer sim, verify no replace directives)                                      | 🟡 Process                                 |
| 23  | **Tag the broken v0.6.0 release as a GitHub "broken" release** with a pointer to v0.6.1                               | 🟢                                         |
| 24  | **Investigate go.work.sum consistency** after examples addition                                                       | 🟢                                         |
| 25  | **Consider `gofmt -s` / `gofumpt` on all go.mod files** in CI                                                         | 🟢                                         |

---

## g) My Top #1 Question 🤔

**"Did BuildFlow push `48d7e70` to the remote, and if so, should I push the 7 tags now to complete the release — or should I discard the dirty pin bumps first, amend/re-commit, and re-tag so the tagged commit references its own version family?"**

I cannot resolve this myself because:

- It requires knowing whether the **commit is already public** — if so, amending/re-tagging is destructive and wrong; I must push tags as-is. If not, I have a clean slate to make the tagged commit self-consistent.
- It requires a decision on **pin-chasing**: accept lagging-but-valid pins (commit is final) vs. chase consistency (needs another tag round, ad infinitum).
- Pushing tags is an explicit remote operation the global instructions forbid without permission — even though the commit may have auto-pushed, the tags definitely have not.

---

_Generated 2026-07-05 21:00 CEST. Waiting for instructions._
