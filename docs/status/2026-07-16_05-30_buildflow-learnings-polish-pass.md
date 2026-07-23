# Status Report: BuildFlow Learnings Polish & Documentation Pass

**Date:** 2026-07-16 05:30
**Session:** Continuation of BuildFlow-inspired API integration (commit `fa60334` was the prior session's commit)
**Branch:** master (uncommitted changes)

---

## Executive Summary

> **Update 2026-07-23:** All work described here was committed as `814b493`
> ("Harden v0.8.0 APIs with type coverage, fuzz tests, and doc updates"). The
> v0.8.0 hardening session report (`2026-07-16_06-30_v080-hardening-session.md`)
> is the authoritative follow-up — it resolved the `[]byte`/`time.Time`/`error`
> cases in `contextValueToString`, added fuzz tests, and verified `nix flake
check`. v0.8.0 code is at HEAD but has **not been tagged** (latest tag is
> `v0.7.0`).

The prior session implemented 4 BuildFlow-inspired APIs (`ExitCoder`, `WrapOnce`, `WithContextAny`, `safeCauseString`) and committed them as `fa60334`. This session was the polish pass: filling API gaps, splitting the monolithic test file, adding examples/benchmarks/assertions, and updating all 6 documentation surfaces. All tests pass (97.6% root coverage), 0 lint issues, 0 race conditions.

---

## a) FULLY DONE

### Code Changes (this session)

| #   | Change                                                                          | File                                                                                    | Status         |
| --- | ------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | -------------- |
| 1   | `formatVerbose` shows `exit_code` when non-zero                                 | `error.go:137-139`                                                                      | DONE, tested   |
| 2   | `jsonError` documents why `exitCode` is excluded                                | `error.go:300-302`                                                                      | DONE           |
| 3   | `WrapOncef` — formatted variant of `WrapOnce`                                   | `constructors.go:154-158`                                                               | DONE, tested   |
| 4   | Split `buildflow_learnings_test.go` into 4 focused files                        | `wraponce_test.go`, `exitcode_test.go`, `context_any_test.go`, `panic_recovery_test.go` | DONE           |
| 5   | `AssertExitCode(tb, err, want)` helper                                          | `errorfamilytest/errorfamilytest.go`                                                    | DONE, tested   |
| 6   | 4 Example functions (WrapOnce, WithExitCode, WithContextAny, ExitCode)          | `example_test.go`                                                                       | DONE, verified |
| 7   | 4 Benchmarks (WrapOnceWrap, WrapOnceIdempotent, WithExitCode, ExitCodeOverride) | `benchmark_test.go`                                                                     | DONE           |

### Documentation Updates (this session)

| #   | File                                         | What changed                                                                           |
| --- | -------------------------------------------- | -------------------------------------------------------------------------------------- |
| 1   | `CHANGELOG.md`                               | v0.8.0 entry with all new APIs                                                         |
| 2   | `SKILL.md`                                   | ExitCoder interface, WithExitCode, WithContextAny, WrapOnce/WrapOncef in API reference |
| 3   | `README.md`                                  | Restored Go Report Card badge (was phantom-removed), added ExitCoder/WrapOnce/features |
| 4   | `FEATURES.md`                                | New FULLY_FUNCTIONAL entries, verified date bumped to 0.8.0                            |
| 5   | `website/src/content/docs/api-reference.mdx` | ExitCoder interface, WrapOnce/WrapOncef, ExitCode description                          |
| 6   | `AGENTS.md`                                  | WrapOncef mention, AssertExitCode added to errorfamilytest list                        |

### Prior Session (commit `fa60334` — already committed)

- `ExitCoder` interface (`interfaces.go`)
- `Error.WithExitCode(code)`, `Error.ExitCode()`, `Error.WithContextAny(key, value)` (`error.go`)
- `WrapOnce(err, family, code, msg)` (`constructors.go`)
- `safeCauseString` panic recovery (`error.go`)
- `ExitCode(err)` checks ExitCoder first (`classify.go`)
- `resolveExitCode` helper in `handle.go`
- `contextValueToString` type switch (`error.go`)
- `buildflow_learnings_test.go` (now split — see above)
- `docs/status/2026-07-16_04-32_buildflow-learnings-integration.md`

### Verification

- **Build:** OK (`GOEXPERIMENT=jsonv2 go build ./...`)
- **Tests:** 97.6% root (up from 97.3%), 95.8% errorfamilytest (up from 95.2%)
- **Lint:** 0 issues (`golangci-lint run ./...`)
- **Race:** Clean (`-race` flag)
- **Submodules:** bridge, agent, diagnose, diagnose/git, diagnose/postgres — all pass
- **Examples:** All 15 example tests produce correct output

---

## b) PARTIALLY DONE

### Nothing is committed

All 13 modified + 4 new files are uncommitted. The prior session committed `fa60334` with the core code, but this session's polish layer (formatVerbose exitCode, WrapOncef, test split, AssertExitCode, examples, benchmarks, all doc updates) has no commit. The working tree has both staged and unstaged changes mixed.

### Previous status report is stale

`docs/status/2026-07-16_04-32_buildflow-learnings-integration.md` was committed in `fa60334` but now describes an incomplete state (lists WrapOncef as "not started", formatVerbose gap as "missing", etc.). This report supersedes it.

### go.mod version not bumped

`go.mod` has no version tag mechanism (Go uses git tags), but the CHANGELOG references v0.8.0. No git tag exists yet. This is expected — tags happen at release time, not during development.

---

## c) NOT STARTED

### Fuzz tests for new APIs

No fuzz tests exist for `WrapOnce`, `WithExitCode`, `WithContextAny`, `contextValueToString`, or `safeCauseString`. The existing fuzz suite covers `Classify`, `ParseFamily`, and error formatting, but the new APIs are fuzz-blind.

### TODO_LIST.md update

Not checked or updated. May have stale entries or missing entries for the new APIs.

### ROADMAP.md update

Not checked or updated.

### docs/DOMAIN_LANGUAGE.md update

Not checked. `ExitCoder` may warrant a glossary entry if the file covers the consumer interfaces.

### Website rebuild/deploy

The `website/src/content/docs/api-reference.mdx` was updated but the Astro site hasn't been rebuilt or deployed. The live site at `errorfamily.lars.software` still shows the old API reference.

---

## d) TOTALLY FUCKED UP

### Nothing is fucked up

All code compiles, all tests pass, lint is clean, race detector is clean. No bugs introduced.

### However — things I should have caught earlier:

1. **README.md phantom change should have been caught immediately.** The prior session somehow deleted the Go Report Card badge line. I noticed it only because the handoff notes mentioned it. If I had run `git diff --cached` at the start, I would have caught it instantly instead of needing a dedicated investigation step.

2. **The prior session left `buildflow_learnings_test.go` with lint warnings** (`gci` formatting, `unparam` for unused `wrapExternal` parameter). These were visible in diagnostics throughout but only got resolved when I split the file. The prior session should have fixed these before committing.

3. **The prior session committed `buildflow_learnings_test.go` with a `wrapExternal` helper that was already known to be unused** (per the handoff notes: "unparam linter flagged wrapExternal helper in test file as having unused base parameter. Fixed by removing the helper and inlining fmt.Errorf"). But the committed version still had the issue. This means the fix was described but not actually applied before commit.

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Run `git diff --cached` before starting work** — would have caught the README badge deletion immediately.
2. **Verify lint passes before committing** — the prior session committed code with known lint warnings.
3. **Don't leave monolithic test files** — `buildflow_learnings_test.go` should have been split from the start. Naming tests after the inspiration source ("buildflow_learnings") instead of the feature ("wraponce", "exitcode") is an anti-pattern.
4. **Update docs in the same commit as the code** — the prior session committed code without updating CHANGELOG, SKILL.md, README.md, FEATURES.md, or the website. This creates a window where the code and docs disagree.

### Code improvements

5. **`contextValueToString` doesn't handle `[]byte`** — common in Go (raw JSON, file contents). Currently falls through to `fmt.Sprint` which produces `[65 66 67]` instead of "ABC". Should add a `[]byte` case that uses `string(val)`.
6. **`contextValueToString` doesn't handle `error` type** — an error value in context would render via `fmt.Sprint` which calls `Error()`. This is correct but could panic (the very thing `safeCauseString` guards against). Should wrap in recovery or document the risk.
7. **`contextValueToString` doesn't handle `time.Time`** — would render as `2006-01-02 15:04:05.999999999 -0700 MST` via `fmt.Sprint`. Should use RFC3339 for consistency with the rest of the library.
8. **`WrapOnce` uses `errors.AsType[*Error]` but doesn't walk the full chain** — `errors.AsType` does walk the chain (it's the generic version of `errors.As`), so this is actually correct. But the doc comment could be clearer about this.
9. **`safeCauseString` has no test for `Error()` method that panics with a non-string value** — the recover catches `any`, but we only tested string panics.
10. **`ExitCode(err)` package function doesn't have a benchmark for the `errors.AsType[ExitCoder]` path** — only the family-default path is benchmarked (`BenchmarkExitCode` exists but `BenchmarkExitCodeOverride` was added this session and tests the override path, so this is actually covered).

### Architecture improvements

11. **The bridge package's `ClassifiedError` should document why it doesn't implement `ExitCoder`** — the decision was made (YAGNI) but not documented in code. A comment on `ClassifiedError` would prevent future contributors from adding it without understanding the tradeoff.
12. **No integration test exists for the full ExitCoder flow** — unit tests verify each piece (interface, override, handler) but no test runs `NewTransient(...).WithExitCode(42)` through `HandleError` and checks that `os.Exit` would receive 42. The `TestHandleErrorDetailedRespectsExitCoder` test is close but only checks `HandleResult.ExitCode`, not the actual `HandleError` return value.

---

## f) Up to 50 Things We Should Get Done Next

### Immediate (blocking release)

1. **Commit all changes** from this session (13 modified + 4 new files)
2. **Delete or update the stale status report** (`docs/status/2026-07-16_04-32_buildflow-learnings-integration.md`) — it describes an incomplete state
3. **Add `[]byte` case to `contextValueToString`** — common Go type, currently renders badly
4. **Add `time.Time` case to `contextValueToString`** — should use RFC3339
5. **Add `error` case to `contextValueToString`** with safeCauseString — defense in depth

### Testing

6. **Add fuzz test for `WrapOnce`** — fuzz the error input, verify idempotency holds
7. **Add fuzz test for `contextValueToString`** — fuzz with random `any` values, verify no panic
8. **Add fuzz test for `WithExitCode` chain** — fuzz exit codes, verify copy-on-write isolation
9. **Add integration test: `HandleError` return value respects `WithExitCode`** — end-to-end CLI path
10. **Add test: `safeCauseString` with non-string panic value** — e.g., `panic(42)` or `panic(nil)`
11. **Add test: `contextValueToString` with `[]byte`** (after adding the case)
12. **Add test: `contextValueToString` with `time.Time`** (after adding the case)
13. **Add test: `contextValueToString` with negative numbers** — verify `-42` renders correctly
14. **Add test: `WrapOncef` with existing `*Error` in wrapped chain** — `fmt.Errorf("wrap: %w", classifiedErr)` then `WrapOncef`
15. **Add benchmark: `contextValueToString` for each type** — type switch vs `fmt.Sprint` comparison

### Documentation

16. **Update `TODO_LIST.md`** — add entries for fuzz tests, contextValueToString edge cases
17. **Check `ROADMAP.md`** — may need updating with the new API direction
18. **Check `docs/DOMAIN_LANGUAGE.md`** — add `ExitCoder` to the glossary if interfaces are documented there
19. **Add comment on bridge `ClassifiedError`** documenting why it doesn't implement `ExitCoder`
20. **Rebuild and deploy website** — `api-reference.mdx` was updated but the live site is stale
21. **Verify the website `api-reference.mdx` renders correctly** in Astro/Starlight
22. **Add a "What's New in v0.8.0" section to the website** if the pattern exists

### Code quality

23. **Run `nix flake check`** — the AGENTS.md says to check flake.nix first; we only ran go commands
24. **Run `nix run .#lint`** — verify the nix-based lint passes (may catch issues golangci-lint CLI misses)
25. **Run `nix run .#test`** — verify the nix-based test runner passes
26. **Verify `GOEXPERIMENT=jsonv2` is set in all CI paths** — the new code doesn't use json/v2 directly but the module does
27. **Check if `examples/` module needs updating** — it has its own go.mod; new APIs may warrant example additions
28. **Add `WrapOnce` usage to `examples/cmd/`** if a suitable example exists

### API completeness

29. **Consider `WrapOnce` family-specific variants** (`WrapOnceRejection`, etc.) — currently only the generic `WrapOnce`/`WrapOncef` exist; the `Wrap*` family has 5 variants each for `New` and `Wrap`
30. **Consider `WithContextAnyMap(map[string]any)`** — bulk typed-context attachment
31. **Consider `ExitCode` validation** — should negative exit codes be allowed? Currently any `int` is accepted
32. **Consider `WithExitCodef`** — probably unnecessary (exit codes are ints, not strings), but document why
33. **Review whether `ExitCoder` should be in the bridge** — the decision was "no" but could be revisited if consumers request it

### Architecture review

34. **Audit all `With*` methods for consistent copy-on-write** — `WithExitCode` was added; verify the pattern is identical to `WithContext`, `WithCause`, `WithTimestamp`
35. **Verify `clone()` copies ALL fields** — `exitCode` was added; if another field is added later, `clone()` must be updated. Consider a table-driven clone test.
36. **Review `resolveExitCode` naming** — it's in `handle.go` but could arguably live in `classify.go` next to `ExitCode()`. Consider consolidation.
37. **Check if `ExitCoder` should participate in `Classify`** — currently it doesn't (only `Classified` and `Retryable` influence classification). Should an ExitCoder with code 0 classify differently? Probably not, but document why.
38. **Consider whether `JSON()` should have an option to include exit code** — some consumers run CLI tools behind HTTP APIs and might want it. Currently documented as excluded.

### Security and robustness

39. **Audit `contextValueToString` for all types that could panic** — `fmt.Sprint` on a nil pointer would panic. Add a `recover` guard or test for this.
40. **Verify `safeCauseString` doesn't swallow stack traces** — the `recover()` catches the panic but doesn't log it. In production, a silently swallowed panic could hide a real bug. Consider a `log.Debug` or at minimum document the tradeoff.
41. **Check for integer overflow in `ExitCode`** — exit codes are OS-level and typically 0-255. Currently any `int` is accepted. Should we validate?

### CI/CD

42. **Verify CI pipeline handles the new test files** — the 4 new test files should be automatically picked up by `go test ./...`
43. **Check if golangci-lint cache needs clearing** — new files may not be linted if cache is stale
44. **Verify the examples build step in CI** — `examples/` is a separate module; new APIs don't affect it but verify

### Cleanup

45. **Remove the `docs/status/2026-07-16_04-32_buildflow-learnings-integration.md` if it's superseded** — or mark it as superseded by this report
46. **Review all `//nolint` directives** — none were added this session, but verify the existing ones still apply
47. **Check for any `TODO` or `FIXME` comments introduced** — none were added, but verify
48. **Verify `git mv` was used for the test file split** — actually, the old file was `trash`ed and new files were `write`n. This means git sees it as delete+create, not a rename. The commit will lose rename detection. Not critical but suboptimal.
49. **Stage all changes properly** — currently there's a mix of staged (deletion of old test file) and unstaged changes. Need a clean `git add -A` before committing.
50. **Write a proper commit message** following the project's commit conventions for the polish layer

---

## g) Top 2 Questions

### Q1: Should `contextValueToString` handle `[]byte`, `time.Time`, and `error` types?

These are extremely common Go types that currently fall through to `fmt.Sprint`:

- `[]byte` renders as `[65 66 67]` instead of `"ABC"` — broken UX
- `time.Time` renders as `2006-01-02 15:04:05.999999999 -0700 MST` instead of RFC3339 — inconsistent with the rest of the library
- `error` calls `.Error()` which could panic — the very thing `safeCauseString` exists to guard against

I can add these cases myself, but I want to confirm the desired behavior for `[]byte` (string conversion vs base64 encoding vs hex encoding). For API-boundary context, `string(val)` seems right, but if these values end up in logs, a long `[]byte` could be noisy.

### Q2: Should this session's work be committed as a single commit or split?

The changes fall into distinct logical groups:

1. **Code gap fixes** (formatVerbose exitCode, WrapOncef, jsonError documentation)
2. **Test infrastructure** (test file split, AssertExitCode, examples, benchmarks)
3. **Documentation updates** (CHANGELOG, SKILL, README, FEATURES, website, AGENTS)

A single commit is simpler and the changes are tightly coupled (all relate to the BuildFlow learnings). But splitting into 2-3 commits would give cleaner history. What's your preference?

---

## Session Metrics

| Metric                   | Value                                                                                       |
| ------------------------ | ------------------------------------------------------------------------------------------- |
| Files modified           | 13                                                                                          |
| Files created            | 4 (`wraponce_test.go`, `exitcode_test.go`, `context_any_test.go`, `panic_recovery_test.go`) |
| Files deleted            | 1 (`buildflow_learnings_test.go`)                                                           |
| Net lines changed        | -174 (315 deleted, 201 added — test split reduced duplication)                              |
| Test cases               | ~40 (across 4 new test files + errorfamilytest)                                             |
| New examples             | 4                                                                                           |
| New benchmarks           | 4                                                                                           |
| Root coverage            | 97.6% (was 97.3%)                                                                           |
| errorfamilytest coverage | 95.8% (was 95.2%)                                                                           |
| Lint issues              | 0                                                                                           |
| Race conditions          | 0                                                                                           |
| Commits made             | 0 (all uncommitted)                                                                         |
