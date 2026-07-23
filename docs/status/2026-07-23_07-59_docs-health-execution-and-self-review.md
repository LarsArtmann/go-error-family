# Status Report: Docs Health Execution + Self-Review

**Date:** 2026-07-23 07:59 CEST
**Session goal:** Execute the plan from the prior session's TODO_LIST.md and status report (2026-07-23_06-49), completing all remaining verification and code-quality tasks.
**Working tree:** Clean (auto-commit hook fired — see section d.1)
**Branch:** `master`
**HEAD:** `e9c7219 ci: harden release pipeline with module-graph gate, vet, and consumer simulation`

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                             | Evidence                                                                                 |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| 1   | **Fixed 2 stale `SuggestedFix` refs in website `diagnostics.mdx`** — replaced with `Fix.Summary` / `Fix.Command`                                                                                                                 | `website/src/content/docs/guides/diagnostics.mdx` lines 17, 87                           |
| 2   | **Fixed website `error-types.mdx`** — added ExitCoder to interface table (4→5), added WithContextAny/WithExitCode to mutator section, added full mutator table                                                                   | `website/src/content/docs/guides/error-types.mdx`                                        |
| 3   | **Fixed website `api-reference.mdx`** — added mutators section (7 methods with signatures), added errorfamilytest subpackage section                                                                                             | `website/src/content/docs/api-reference.mdx`                                             |
| 4   | **Added `[Unreleased]` section to website `changelog.mdx`** — documents ExitCoder, WithExitCode, WithContextAny, WrapOnce, safeCauseString, Compose re-add                                                                       | `website/src/content/docs/changelog.mdx`                                                 |
| 5   | **Resolved Compose split brain** — CHANGELOG v0.5.0 "Removed" vs FEATURES "FULLY_FUNCTIONAL". Added re-add note to CHANGELOG `[Unreleased]` with commit ref                                                                      | `CHANGELOG.md` — verified `classify.go` has `func Compose`, commit `8cb240a` re-added it |
| 6   | **Fixed CONTRIBUTING.md** — removed dead CODE_OF_CONDUCT.md link (file doesn't exist), added ExitCoder to architecture tree                                                                                                      | `CONTRIBUTING.md` lines 17, 130                                                          |
| 7   | **Fixed SKILL.md WithContextAny** — replaced vague "etc." with full type list: string, int, int64, uint, uint64, float64, bool, []byte, time.Time, error, nil                                                                    | `SKILL.md` line 159                                                                      |
| 8   | **Added errkit consumer pattern to SKILL.md** — domain error helper example showing typed factory functions                                                                                                                      | `SKILL.md` after line 207                                                                |
| 9   | **Updated DOMAIN_LANGUAGE.md** — added Registry, WrapOnce, HTTPHandler to glossary                                                                                                                                               | `docs/DOMAIN_LANGUAGE.md`                                                                |
| 10  | **Annotated DiscordSync scorecard** — added "Ratings reflect codebase at time of feedback" disclaimer noting HTTPHandler/Registry/Classifier improvements                                                                        | `docs/feedback/2026-07-05_DiscordSync.md`                                                |
| 11  | **Refactored `contextValueToString`** — split into `contextValueToString` + `scalarToString`, eliminated `//nolint:cyclop`, added `time.Duration` case                                                                           | `error.go` — tests pass with `-race`                                                     |
| 12  | **Documented negative exit codes** — WithExitCode godoc now explains POSIX wrapping (0-255 range, negative values become 255)                                                                                                    | `error.go` WithExitCode comment                                                          |
| 13  | **Added `writeHTTPError` error-branch test** — failingResponseWriter that returns error on Write, covers the json-encode error path                                                                                              | `http_test.go` TestWriteHTTPErrorMarshalFailure                                          |
| 14  | **Added `safeCauseString` non-string panic tests** — int panic, nil panic, struct panic all recovered                                                                                                                            | `error_test.go` TestSafeCauseStringNonStringPanics                                       |
| 15  | **Added CI improvements** — `GOWORK=off go list -m all` gate, `go vet ./...`, consumer-simulation job (throwaway module import)                                                                                                  | `.github/workflows/ci.yml`                                                               |
| 16  | **Cleaned TODO_LIST.md** — removed 12 completed items (New* vs Wrap*, website audit, mutators section, errkit, writeHTTPError test, negative exit codes, contextValueToString refactor, time.Duration, CI gate, CI consumer-sim) | `TODO_LIST.md` — from 12 active items to 3 (1 High, 2 Low)                               |
| 17  | **Updated CHANGELOG `[Unreleased]`** — recorded Compose re-add, time.Duration, new tests, CI improvements, contextValueToString refactor, WithExitCode docs                                                                      | `CHANGELOG.md`                                                                           |
| 18  | **Quality gate: all tests pass** — root, errorfamilytest, diagnose, agent, bridge all pass with `-race`                                                                                                                          | All 5 test suites green                                                                  |
| 19  | **Quality gate: treefmt passes** — auto-formatted http_test.go and verified clean                                                                                                                                                | `nix fmt` + `nix flake check` treefmt stage                                              |

**Stats:** 17 files changed, 336 insertions, 150 deletions. Auto-committed as `e9c7219` (see section d.1).

---

## b) PARTIALLY DONE

| #   | Item                           | What's done                                                                                                                           | What remains                                                                                                                                                                                                                       |
| --- | ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Website docs audit**         | Fixed 4 website files (diagnostics.mdx, error-types.mdx, api-reference.mdx, changelog.mdx). All 11 .mdx files were read by sub-agent. | Did NOT fix `changelog.mdx` to match root `CHANGELOG.md` line-for-line (website version is a summary, not a mirror — may be intentional but wasn't verified). Website `contributing.mdx` not checked for CODE_OF_CONDUCT.md ghost. |
| 2   | **Living doc consistency**     | CHANGELOG, TODO_LIST, SKILL.md, DOMAIN_LANGUAGE, CONTRIBUTING all updated                                                             | FEATURES.md NOT updated with this session's changes (time.Duration, contextValueToString refactor, new tests). AGENTS.md NOT updated — still references `//nolint:cyclop` which I removed.                                         |
| 3   | **Quality gate**               | Tests pass, treefmt passes, go vet passes, go build passes                                                                            | `nix flake check` lint stage FAILS with 96 pre-existing issues (err113: 50, varnamelen: 12, testpackage: 10, mnd: 14, etc.). These are from prior v0.8.0 work, NOT from this session's changes.                                    |
| 4   | **CI consumer-simulation job** | Job written and committed                                                                                                             | NOT tested locally — the YAML heredoc was originally wrong (caught and fixed), but the actual `go mod init` + replace + build sequence was never executed locally to verify it works.                                              |

---

## c) NOT STARTED

| #   | Item                                                                                                                                               | Why                                                                                                                                                                                                                   |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Update FEATURES.md** with this session's changes (time.Duration, refactored contextValueToString, new tests, CI improvements)                    | Forgot. FEATURES is the feature inventory and should reflect current state.                                                                                                                                           |
| 2   | **Update AGENTS.md** — remove `//nolint:cyclop` reference, update contextValueToString description to mention scalarToString split + time.Duration | Forgot. AGENTS.md is the AI context file and now has stale info.                                                                                                                                                      |
| 3   | **Verify website `contributing.mdx`** for CODE_OF_CONDUCT.md ghost reference                                                                       | Only checked root CONTRIBUTING.md. Website copy may have the same dead link.                                                                                                                                          |
| 4   | **Add explicit `time.Duration` test case** in error_test.go                                                                                        | Added the Duration case to the type switch but no dedicated test asserts `5s` renders correctly. FuzzContextValueToString would catch it eventually but no explicit assertion exists.                                 |
| 5   | **Verify CONTRIBUTING.md architecture tree renders correctly**                                                                                     | Changed `  interfaces.go` to `│   interfaces.go` — the box-drawing character may not align with the rest of the ASCII tree which uses spaces.                                                                         |
| 6   | **Cross-check website changelog.mdx vs root CHANGELOG.md**                                                                                         | Both have `[Unreleased]` now but content differs. Website is a summary; root is detailed. Didn't verify they tell the same story.                                                                                     |
| 7   | **Update prior status report** (`2026-07-23_06-49`) questions                                                                                      | That report asked Q1 (commit now?), Q2 (tag v0.8.0?), Q3 (Compose exists?). This session answered Q3 (yes, it exists, CHANGELOG updated) and partially Q1 (auto-committed). The report wasn't annotated with answers. |
| 8   | **Rebuild and deploy website**                                                                                                                     | Website docs are fixed in source but the live site (`errorfamily.lars.software`) is still stale. Needs `nix run .#deploy` from `website/`.                                                                            |

---

## d) TOTALLY FUCKED UP

### 1. Auto-commit hook bundled everything into one misleading commit

The project has an auto-commit hook that fired when I finished. It committed ALL 17 files as `e9c7219` with the message "ci: harden release pipeline with module-graph gate, vet, and consumer simulation."

This is terrible because:

- The commit message only describes the CI changes (3 of 17 files)
- The other 14 files include website doc fixes, code refactoring, new tests, living doc updates, feedback annotations — none mentioned in the message
- Anyone reading `git log` will think this commit is CI-only and miss the docs/code changes
- The commit mixes 5+ logically distinct concerns (website fixes, code refactor, test additions, CI, docs-health, feedback annotations)

**Root cause:** I didn't know the auto-commit hook existed. I should have checked `crush_info` or the hooks config at session start. I also should have committed incrementally with proper messages instead of letting the hook bundle everything.

### 2. I mixed code changes into a docs session

The user's task was about TODO_LIST.md and the status report, extending into docs-health execution. I expanded scope to include:

- `error.go` refactoring (contextValueToString → contextValueToString + scalarToString)
- New test code (3 test functions across 2 files)
- CI workflow changes (.github/workflows/ci.yml)

While the docs-health skill says "fix issues on sight" and the global AGENTS.md says "smart auto-fixes," I should have been more disciplined about scope. The code changes should have been a separate session or at minimum a separate commit.

### 3. The CHANGELOG [Unreleased] now has internal implementation details

I added entries like "contextValueToString refactored" and "WithExitCode godoc documents POSIX" to CHANGELOG `[Unreleased]`. These are internal improvements, not user-facing API changes. CHANGELOG should focus on what consumers see. A refactor that doesn't change behavior is not a changelog item — it's an internal commit message concern.

### 4. I didn't run `nix fmt` before `nix flake check`

The first `nix flake check` run failed on treefmt because my `http_test.go` had a long line. I should have formatted before checking. This wasted a full nix evaluation cycle (~30 seconds).

### 5. I didn't notice the working tree was clean at the end

When writing this report, I ran `git status` expecting to see my uncommitted changes. Instead the tree was clean — the auto-commit had already fired. I almost reported "14 files uncommitted" based on stale mental state. The docs-health skill explicitly warns: "Never describe working-tree state without a fresh `git status` in the same message."

---

## e) WHAT WE SHOULD IMPROVE

1. **Check for auto-commit hooks at session start.** Run `crush_info` or look at the hooks config. If a hook exists, commit incrementally BEFORE it fires, so each commit has a proper message. This session's mega-commit is a direct consequence of not knowing the hook existed.

2. **Separate docs work from code work.** A docs-health session should fix docs. Code refactoring, test additions, and CI changes are separate concerns that deserve separate commits with focused messages. When the docs-health skill says "fix issues on sight," that means fix doc issues — not refactor production code.

3. **CHANGELOG hygiene: user-facing changes only.** Internal refactors, test additions, and doc improvements don't belong in CHANGELOG unless they change the API or behavior. The CHANGELOG is for consumers, not developers.

4. **Update AGENTS.md when changing code it describes.** AGENTS.md mentions `//nolint:cyclop` on contextValueToString. I removed that nolint. AGENTS.md is now stale. This is the exact "docs drift" pattern the docs-health skill exists to prevent — and I caused it while running the skill.

5. **Run `nix fmt` before `nix flake check`.** Always. The check will fail on formatting, wasting time. Format first, then check.

6. **Test CI YAML locally before committing.** The consumer-simulation job was written blind. I should have at minimum run the shell commands manually to verify they work.

7. **Cross-file consistency after changes, not just before.** I read all files at the start and found drift. Then I made changes that created NEW drift (AGENTS.md stale, FEATURES.md stale). The consistency check should run AFTER changes, not just before.

---

## f) Up to 50 Things We Should Get Done Next

### Immediate (gaps from this session)

| #   | Task                                                                                                            | Impact |
| --- | --------------------------------------------------------------------------------------------------------------- | ------ |
| 1   | **Update AGENTS.md** — remove `//nolint:cyclop` reference, update contextValueToString description              | Medium |
| 2   | **Update FEATURES.md** — add time.Duration case, contextValueToString split, new tests                          | Medium |
| 3   | **Clean CHANGELOG [Unreleased]** — remove internal refactor/test entries, keep only user-facing changes         | Medium |
| 4   | **Verify CONTRIBUTING.md architecture tree** renders correctly (box-drawing alignment)                          | Low    |
| 5   | **Check website `contributing.mdx`** for CODE_OF_CONDUCT.md ghost                                               | Low    |
| 6   | **Add explicit time.Duration test** in error_test.go                                                            | Low    |
| 7   | **Test consumer-simulation CI job locally** — run the exact shell commands                                      | Medium |
| 8   | **Annotate prior status report** (2026-07-23_06-49) with answers to Q1-Q3                                       | Low    |
| 9   | **Fix the auto-commit** — consider amending `e9c7219` message to cover all changes, or leave as-is and document | Design |

### From TODO_LIST.md (genuinely open work)

| #   | Task                                                      | Impact |
| --- | --------------------------------------------------------- | ------ |
| 10  | Rebuild and deploy website (docs fixed, site still stale) | High   |
| 11  | Set up CI/CD for website deploys                          | Low    |
| 12  | Apply ACME TXT DNS record (needs Namecheap API key)       | Low    |

### Design decisions (need user input)

| #   | Task                                                      | Impact   |
| --- | --------------------------------------------------------- | -------- |
| 13  | **v0.8.0 release** — tag or wait?                         | Critical |
| 14  | Per-error HTTP status override (`WithHTTPStatus`)         | Design   |
| 15  | `Classify(nil)` semantics (keep Rejection vs change)      | Design   |
| 16  | Constructor context ergonomics (builder/variadic/options) | Design   |
| 17  | "Frozen" registry flag                                    | Design   |
| 18  | `RegisterClassificationType[T error]` generic             | Design   |
| 19  | json/v2 migration strategy                                | Design   |

### Pre-existing lint debt (96 issues from v0.8.0 code, not this session)

| #   | Task                                                                                                     | Impact |
| --- | -------------------------------------------------------------------------------------------------------- | ------ |
| 20  | Fix 50 `err113` issues — "do not define dynamic errors, use wrapped static errors"                       | Medium |
| 21  | Fix 14 `mnd` issues — magic numbers in test code                                                         | Low    |
| 22  | Fix 12 `varnamelen` issues — short variable names (`f`, `r`, `c`, `w`, `h`, `tp`)                        | Low    |
| 23  | Fix 10 `testpackage` issues — tests should be in `errorfamily_test` not `errorfamily`                    | Medium |
| 24  | Fix `containedctx`, `depguard`, `fatcontext`, `funlen`, `makezero`, `nonamedreturns`, `testableexamples` | Low    |

### Documentation polish

| #   | Task                                                                        | Impact |
| --- | --------------------------------------------------------------------------- | ------ |
| 25  | Cross-verify website changelog.mdx vs root CHANGELOG.md tell the same story | Low    |
| 26  | Add "last verified" date to README benchmark table                          | Low    |
| 27  | Verify CHANGELOG `{{.key}}` → `{key}` note in v0.1.0 entry is accurate      | Low    |

### Testing

| #   | Task                                                                                | Impact |
| --- | ----------------------------------------------------------------------------------- | ------ |
| 28  | Run extended fuzz sessions (`-fuzztime=30s`) for all 14 fuzz functions              | Low    |
| 29  | Add benchmark: `contextValueToString` per type (now that scalarToString exists)     | Low    |
| 30  | Add `fmt.Stringer` case to `contextValueToString` with panic recovery               | Low    |
| 31  | Add integration test: `HandleError` return value respects `WithExitCode` end-to-end | Low    |

### CI / Release

| #   | Task                                                                      | Impact |
| --- | ------------------------------------------------------------------------- | ------ |
| 32  | Add pre-commit check for `replace` directives in tagged go.mod files      | Medium |
| 33  | Create release automation script for coordinated multi-module tag cutting | Low    |
| 34  | Add benchmark regression check to CI                                      | Low    |
| 35  | Deprecation notes for broken v0.6.0 family tags                           | Low    |

### Website / Public Presence

| #   | Task                                                            | Impact |
| --- | --------------------------------------------------------------- | ------ |
| 36  | Add CSP to `astro.config.mjs` + `fix-csp.mjs` post-build script | High   |
| 37  | Add OG images via `astro-og-canvas`                             | Medium |
| 38  | Design a proper logo for go-error-family                        | Medium |
| 39  | Add Bridge package guide page (oops integration)                | Medium |
| 40  | Add `errorfamilytest` guide page                                | Low    |
| 41  | Add uptime monitor for `errorfamily.lars.software`              | Medium |
| 42  | Fix corrupted `flake.lock` in the domains repo                  | Low    |
| 43  | Verify all docs pages return HTTP 200 on the custom domain      | Low    |

### Process

| #   | Task                                                                                                     | Impact |
| --- | -------------------------------------------------------------------------------------------------------- | ------ |
| 44  | Document the auto-commit hook in AGENTS.md so future sessions know about it                              | High   |
| 45  | Consider splitting the auto-commit into logical units (docs, code, tests, CI) instead of one mega-commit | Design |
| 46  | Add a pre-commit message template that prompts for scope-appropriate messages                            | Low    |

---

## g) Top 3 Questions I Cannot Answer Myself

### Q1: Should I amend commit `e9c7219` to fix the misleading message, or leave it and move on?

The commit message says "ci: harden release pipeline" but includes 17 files spanning docs, code, tests, and CI. Amending would fix the message but rewrite history (the global AGENTS.md says "NEVER `git reset`"). Leaving it means the git log is misleading. The auto-commit hook made this decision for me — I'm asking whether to undo its damage.

### Q2: Should the CHANGELOG `[Unreleased]` section include internal refactors (contextValueToString split, nolint removal) or only user-facing API changes?

I added internal entries like "contextValueToString refactored" and "WithExitCode godoc documents POSIX" to CHANGELOG. Keep a Changelog says "notable changes" — but is an internal refactor "notable" to a consumer? I lean toward removing them, but the user may want full transparency.

### Q3: Is there a reason v0.8.0 has been untagged for 7+ days?

This is the same question from the prior session's report (Q2). The code is committed, CHANGELOG is ready, all tests pass. The only blocker I can think of is the website not being deployed yet, or a desire to batch more changes before tagging. But I don't know the actual reason — only the user does.

---

_Generated 2026-07-23 07:59 CEST. Auto-commit hook fired before I could commit manually._

---

## Resolution (2026-07-23)

| Item flagged here | Status |
| ----------------- | ------ |
| b.2 / c.2 — AGENTS.md stale (`//nolint:cyclop` ref) | **Fixed.** The 08-35 lint session corrected AGENTS.md — no `cyclop` reference remains. `contextValueToString` → `scalarToString` split and `time.Duration` case are documented. |
| c.1 — FEATURES.md not updated | **Fixed.** FEATURES.md reflects current API (verified 2026-07-23). |
| b.3 — 96 lint issues | **Fixed.** The 08-35 session eliminated all 154 golangci-lint issues across 7 modules (0 remaining). |
| Q1 (amend `e9c7219`?) | Left as-is. The misleading mega-commit message stands; 11 more auto-commits followed in the 15-52 session with equally generic messages. |
| Q2 (CHANGELOG internal entries?) | The `[Unreleased]` CHANGELOG still contains internal entries (contextValueToString refactor, WithExitCode godoc). Kept for transparency. |
| Q3 (v0.8.0 untagged?) | **Still open.** Latest tag is `v0.7.0`. Tracked in TODO_LIST "Design Decisions Needed". |
