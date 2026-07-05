# Status Report — 2026-07-05 20:26 CEST

**Session focus:** Emergency fix of the broken v0.6.0 release (phantom `replace` + `require` directives leaking into the published module graph) + extraction of `examples/` into its own module so the root package is truly zero-dependency.

**Working tree:** 7 modified files, 4 new files, **uncommitted** (awaiting user decision on patch-release tag sequence).

---

## TL;DR

The `v0.6.0` release was **broken for downstream consumers**. Root, `diagnose`, and `agent` modules shipped with `replace` directives pointing at local paths plus phantom `require ... v0.0.0-00010101000000-000000000000` versions. Go strips `replace` when a module is fetched — so consumers (e.g. `project-meta`) hit non-existent versions in their module graph. `go.work` masked this completely for local development. **Now fixed in the working tree, but the published tags remain permanently broken until patch releases are cut.**

---

## a) FULLY DONE ✅

| #   | Item                                                                                                          | Evidence                                                                                 |
| --- | ------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| 1   | **Root `go.mod` cleaned** — removed `replace` block + phantom `require`; now 3 lines, zero non-stdlib deps    | `cat go.mod` → `module ... \n go 1.26.4`                                                 |
| 2   | **`diagnose/go.mod` fixed** — real `require root v0.6.0`, no `replace`                                        | `GOWORK=off go list -m all` resolves                                                     |
| 3   | **`agent/go.mod` fixed** — real `v0.6.0` + `diagnose v0.1.0`, no `replace`                                    | `GOWORK=off go list -m all` resolves                                                     |
| 4   | **`examples/` extracted into own module** — new `examples/go.mod` requiring root + diagnose                   | root no longer pulls `diagnose` into consumers' graphs                                   |
| 5   | **`go.work` updated** — `./examples` added to workspace `use()` list                                          | local builds unchanged                                                                   |
| 6   | **CI workflow updated** — `working-directory: ./examples` for the examples build step                         | `.github/workflows/ci.yml:50-51`                                                         |
| 7   | **`go.sum` files generated** for `diagnose/`, `agent/`, `examples/` (previously absent — relied on workspace) | `GOWORK=off go mod download` succeeds in each                                            |
| 8   | **Root `go.sum` deleted** — zero deps means no sum file needed                                                | `ls go.sum` → gone                                                                       |
| 9   | **All 7 modules pass tests with `-race`**                                                                     | root, errorfamilytest, agent, bridge, diagnose, diagnose/git, diagnose/postgres all `ok` |
| 10  | **0 lint issues** on root                                                                                     | `golangci-lint run ./...` → `0 issues.`                                                  |
| 11  | **Consumer simulation passes** — `GOWORK=off go list -m all` on root shows only itself; no phantom edges      | see verification block below                                                             |
| 12  | **AGENTS.md updated** — workspace modules list includes `examples`; "Examples built in CI" note corrected     | lines 8, 161, 174                                                                        |

**Consumer-simulation proof (the core acceptance test):**

```
$ GOWORK=off go list -m all            # root
github.com/larsartmann/go-error-family
$ cd diagnose && GOWORK=off go list -m all
github.com/larsartmann/go-error-family/diagnose
github.com/larsartmann/go-error-family v0.6.0
$ cd ../agent && GOWORK=off go list -m all
github.com/larsartmann/go-error-family/agent
github.com/larsartmann/go-error-family v0.6.0
github.com/larsartmann/go-error-family/diagnose v0.1.0
```

No `v0.0.0-00010101000000-...` anywhere. Real versions only.

---

## b) PARTIALLY DONE ⚠️

| #   | Item                               | What's done                                                                                                        | What remains                                                                                                                                                                                                                                              |
| --- | ---------------------------------- | ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Patch releases for broken tags** | Source `go.mod` files fixed in working tree                                                                        | Tags `v0.6.0`, `diagnose/v0.1.0`, `agent/v0.1.0` are **permanently broken** upstream — must cut `v0.6.1`, `diagnose/v0.1.1`, `agent/v0.1.1`, and tag `examples/v0.1.0`                                                                                    |
| 2   | **Submodule root-version pins**    | `bridge`, `diagnose/git`, `diagnose/postgres` already use real versions (v0.5.1 / v0.1.0) — they were never broken | They still pin root at `v0.5.1` rather than `v0.6.0`. Valid via MVS (lower bound), but stale. Bump on next release.                                                                                                                                       |
| 3   | **Docs cleanup**                   | AGENTS.md workspace + CI lines updated                                                                             | CHANGELOG.md line 58 still says "Local `replace` directives added until published versions resolve the extraction" — now historically accurate but reads as if still true; could clarify. SKILL.md not audited for stale replace references this session. |

---

## c) NOT STARTED ⏭️

| #   | Item                                                                                                   |
| --- | ------------------------------------------------------------------------------------------------------ |
| 1   | Cutting the actual patch-release tags (`v0.6.1`, `diagnose/v0.1.1`, `agent/v0.1.1`, `examples/v0.1.0`) |
| 2   | Updating `bridge`, `diagnose/git`, `diagnose/postgres` to pin root `v0.6.0`                            |
| 3   | Auditing SKILL.md for any remaining stale "replace directive" / "not yet published" language           |
| 4   | Committing the working-tree changes (user hasn't said "commit")                                        |
| 5   | Pushing anything to remote                                                                             |

---

## d) TOTALLY FUCKED UP 💥

### 1. The v0.6.0 release was release-breaking for consumers

**This is the big one.** The root cause and full impact:

```
go.mod (published at v0.6.0):
  replace (
      .../agent     => ./agent      ← STRIPPED in consumer's view
      .../diagnose  => ./diagnose   ← STRIPPED in consumer's view
  )
  require .../diagnose v0.0.0-00010101000000-000000000000   ← SURVIVES, unresolvable
```

**Why it wasn't caught:** `go.work` (checked into the repo) makes the workspace authoritative for _every_ local invocation. `go test ./...`, `go build ./...`, `golangci-lint`, and the existing `GOWORK=off go build ./...` CI step _all_ passed because (a) workspace mode ignores `replace`/phantom-`require` inconsistencies, and (b) root package code never actually imports `diagnose` — only `examples/cmd/custom_rule` does, and that was compiled under workspace mode. The bug was **invisible from inside the repo** and only surfaced when an external consumer (`project-meta`) ran module-graph tooling.

**Same defect pattern in `diagnose/go.mod` and `agent/go.mod`** — both shipped phantom `v0.0.0-00010101000000-...` requires + local `replace`.

**Severity:** Critical for any consumer that touches the module graph. Compilation alone may work (if they don't import the phantom package), but `go list -m all`, `go doc`, `go mod tidy`, and IDE features break. This is the class of bug that erodes trust in a library on first use.

### 2. Root was lying about being "zero-dependency"

The AGENTS.md and SKILL.md both repeatedly claim "root stays zero-dependency" / "zero-dep". **It was not.** The root `go.mod` had `require github.com/larsartmann/go-error-family/diagnose` — solely because `examples/cmd/custom_rule/main.go` imported it. Every consumer of the root package had `diagnose` in their module graph whether they wanted it or not. The "zero-dep" claim was a documentation lie enforced by nothing. **Now actually true** after extracting `examples/`.

### 3. No CI gate catches this class of bug

There _was_ a `GOWORK=off go build ./...` CI step — but `go build` does not resolve the full module graph; it only pulls packages that are actually imported. The phantom `diagnose` edge was never traversed by root's build. The correct gate is `GOWORK=off go list -m all` (or `go mod verify` / a consumer simulation), which **did not exist**.

---

## e) WHAT WE SHOULD IMPROVE 🛠️

1. **Add a CI gate that fails on unresolvable module graphs.** `GOWORK=off go list -m all` in every submodule directory. This single check would have caught the v0.6.0 break before it shipped.
2. **Treat `replace` directives in published modules as a linter error.** Consider a pre-commit/CI check that fails if `replace` appears in any `go.mod` that has a tag pointing at it. `replace` is for local dev only — `go.work` is the right tool.
3. **Add a "consumer simulation" CI job** — a throwaway module that does `go mod init; go get github.com/larsartmann/go-error-family@<tag>; go list -m all`. This is the only honest proof that a release works downstream.
4. **Stop trusting `go.work` for release validation.** Workspace mode hides exactly the class of bugs that bite consumers. Release CI should run with `GOWORK=off` for _all_ verification, not just one build step.
5. **The "zero-dependency" claim needs a machine-checked invariant**, not prose. A CI assertion `test $(GOWORK=off go list -m all | wc -l) -eq 1` on root would enforce it forever.
6. **Version-pin hygiene:** `bridge`, `diagnose/git`, `diagnose/postgres` still pin root at `v0.5.1`. These should be bumped in lockstep on each root release, or explicitly documented as deliberate lower bounds.
7. **`diagnose/go.mod` uses inline `require`** (`require X v0.6.0`) while `agent/go.mod` and `examples/go.mod` use block style (`require ( ... )`). Pick one and `gofmt` it (Go tolerates both, but consistency matters).

---

## f) Up to 25 things we should get done next 🎯

Ranked roughly by impact × urgency.

| #   | Task                                                                                                            | Impact                                         |
| --- | --------------------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| 1   | **Commit the working-tree fixes** (this session's changes)                                                      | 🔴 Unblock everything                          |
| 2   | **Cut `diagnose/v0.1.1`** tag (its go.mod requires root v0.6.0 — already exists)                                | 🔴 Fixes diagnose for consumers                |
| 3   | **Cut `v0.6.1`** tag on root (requires diagnose v0.1.0 — already exists)                                        | 🔴 Fixes root for consumers incl. project-meta |
| 4   | **Cut `agent/v0.1.1`** tag (requires root v0.6.0 + diagnose v0.1.0 — both exist)                                | 🔴 Fixes agent for consumers                   |
| 5   | **Cut `examples/v0.1.0`** tag (first release of the new examples module)                                        | 🟠 Publishes the extraction                    |
| 6   | **Add CI gate: `GOWORK=off go list -m all`** in each module dir                                                 | 🔴 Prevents recurrence                         |
| 7   | **Add CI consumer-simulation job** (`go get ...@<tag>` in a throwaway module)                                   | 🔴 Honest release proof                        |
| 8   | **Add CI invariant: root `go list -m all` returns exactly 1 line**                                              | 🟠 Enforces zero-dep claim                     |
| 9   | **Bump `bridge` root pin v0.5.1 → v0.6.0** (after v0.6.1 cut)                                                   | 🟠 Pin freshness                               |
| 10  | **Bump `diagnose/git` root pin v0.5.1 → v0.6.0**                                                                | 🟠                                             |
| 11  | **Bump `diagnose/postgres` root pin v0.5.1 → v0.6.0**                                                           | 🟠                                             |
| 12  | **Audit SKILL.md** for stale "replace" / "not yet published" language                                           | 🟡 Doc honesty                                 |
| 13  | **Update CHANGELOG.md** with a v0.6.1 entry documenting the fix + the examples extraction                       | 🟡                                             |
| 14  | **Add a `CONTRIBUTING.md`/release checklist** note: "never ship `replace` in a tagged go.mod; use go.work"      | 🟡 Process                                     |
| 15  | **Normalize `require` style** across all submodule `go.mod` files (inline vs block)                             | 🟢 Polish                                      |
| 16  | **Consider `go mod tidy` in CI** for each module (catches missing go.sum entries)                               | 🟡                                             |
| 17  | **Tag the existing broken tags as deprecated** in release notes / README                                        | 🟢 Public hygiene                              |
| 18  | **Add `go vet ./...` to CI** if not already present (defense in depth)                                          | 🟢                                             |
| 19  | **Document the release-tag sequence** (depended-on modules first) in AGENTS.md                                  | 🟡 Process                                     |
| 20  | **Review whether `errorfamilytest` should be its own module** (it's currently under root but imports `testing`) | 🟢 Future                                      |
| 21  | **Add a `make`/`just`/nix target for cutting coordinated multi-module releases**                                | 🟢 Tooling                                     |
| 22  | **Check `go.work.sum` is consistent** after the examples addition                                               | 🟢                                             |
| 23  | **Verify the nix flake still builds** (`nix build`, `nix flake check`) after module changes                     | 🟡                                             |
| 24  | **Consider a `renovate`/`dependabot` config** for multi-module version pinning                                  | 🟢                                             |
| 25  | **Post-mortem note in README/CHANGELOG** explaining what went wrong with v0.6.0 and how it's now prevented      | 🟡 Trust                                       |

---

## g) My Top #1 Question 🤔

**"Do you want me to commit these working-tree changes and cut the four patch tags (`v0.6.1`, `diagnose/v0.1.1`, `agent/v0.1.1`, `examples/v0.1.0`) right now — and if so, in what order should I push them, given that the existing broken tags can't be deleted without force-pushing the tag refs?"**

I cannot resolve this myself because:

- It requires a decision on whether to **push to remote** (the global instructions forbid pushing without explicit permission).
- It requires a decision on **tag-deprecation strategy**: the broken `v0.6.0` etc. tags can't be silently deleted if anyone has already fetched them (Go module proxy caches them). The options are (a) leave the broken tags and supersede with patch versions, or (b) force-overwrite the tags (destructive, only safe if nobody has consumed them yet). Only you know whether `project-meta` is the sole consumer so far.
- It determines whether `examples/v0.1.0` should be tagged now or whether examples should stay untagged/internal for longer.

---

_Generated 2026-07-05 20:26 CEST. Waiting for instructions._
