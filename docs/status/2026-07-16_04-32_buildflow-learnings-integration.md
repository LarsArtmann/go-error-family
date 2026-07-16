# Status Report: BuildFlow Learnings Integration

**Date:** 2026-07-16 04:32  
**Session Goal:** Learn from BuildFlow's `modules/errors/` and apply improvements to go-error-family  
**Result:** Core implementation COMPLETE, documentation/test/docs INCOMPLETE

---

## Executive Summary

Studied BuildFlow's `modules/errors/` package (24 files, production CLI build tool) and identified 4 patterns to port into go-error-family. All 4 were implemented, tested (31 test cases), and pass with 0 lint issues and 0 race conditions. However, **8 documentation files were NOT updated**, **2 API gaps remain**, and **1 mysterious README.md change appeared** that I did not author.

**Root coverage: 97.6%** (up from 97.3% per AGENTS.md).

---

## a) FULLY DONE

| #   | Item                                                                  | File(s)                                            | Status                                                                                                 |
| --- | --------------------------------------------------------------------- | -------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| 1   | **`WrapOnce`** — idempotent wrapping, prevents double-classify chains | `constructors.go`                                  | Implemented, nil-safe, chain-aware via `errors.AsType[*Error]`                                         |
| 2   | **`ExitCoder` interface** — `error` + `ExitCode() int`                | `interfaces.go`                                    | New 5th consumer interface, embeds `error` for `AsType[T]`                                             |
| 3   | **`Error.ExitCode()` / `Error.WithExitCode(code)`**                   | `error.go`                                         | Copy-on-write, returns 0 when unset (fall back to family default)                                      |
| 4   | **Package `ExitCode(err)` checks ExitCoder first**                    | `classify.go`                                      | `errors.AsType[ExitCoder]` before `Classify(err).ExitCode()`                                           |
| 5   | **`handle.go` `resolveExitCode` helper**                              | `handle.go`                                        | Both `HandleErrorWithContext` and `HandleErrorDetailedWithConfig` use it                               |
| 6   | **`WithContextAny(key, value any)`**                                  | `error.go`                                         | Type switch: string, int, int64, uint, uint64, float64, bool, nil, fallback `fmt.Sprint`               |
| 7   | **`contextValueToString`**                                            | `error.go`                                         | Efficient scalar-to-string conversion, avoids `fmt.Sprint` for common types                            |
| 8   | **`safeCauseString` panic recovery**                                  | `error.go`                                         | `defer/recover` on `cause.Error()` — applied to `Error()`, `Summary()`, `formatVerbose()`              |
| 9   | **Comprehensive test suite**                                          | `buildflow_learnings_test.go`                      | 31 test cases: WrapOnce (4), ExitCoder (9), WithContextAny (12), panic recovery (6), copy-on-write (3) |
| 10  | **`clone()` updated**                                                 | `error.go`                                         | Deep-copies `exitCode` field alongside all other fields                                                |
| 11  | **`AGENTS.md` updated**                                               | `AGENTS.md`                                        | New "BuildFlow-Inspired APIs" section, Surprising Behaviors updated, API Surface updated               |
| 12  | **All tests pass**                                                    | root + errorfamilytest + bridge + agent + diagnose | 0 failures, 0 race conditions                                                                          |
| 13  | **0 lint issues**                                                     | `golangci-lint run ./...`                          | Clean                                                                                                  |
| 14  | **exhaustruct compliance**                                            | `constructors.go`                                  | Both `New()` and `Wrap()` explicitly set `exitCode: 0`                                                 |

### Files Changed (7 modified, 1 new)

```
 AGENTS.md       | 20 +++++++++++--
 classify.go     | 12 +++++++-
 constructors.go | 21 ++++++++++++++
 error.go        | 90 +++++++++++++++++++++++++++++++++++++++++++++++++++++----
 handle.go       | 15 ++++++++--
 interfaces.go   | 11 +++++++
 buildflow_learnings_test.go (NEW — 314 lines)
```

---

## b) PARTIALLY DONE

### 1. `formatVerbose` does NOT show exit code override

`error.go:formatVerbose()` prints family, code, context, timestamp, and cause — but NOT the custom exit code. If someone sets `.WithExitCode(42)` and does `fmt.Sprintf("%+v", err)`, the exit code override is invisible.

**Impact:** Debugging gap. A developer using `%+v` won't see why an exit code is wrong.

### 2. `jsonError` struct does NOT include exit code

`error.go:jsonError` serializes `{family, code, message, context, retryable, timestamp}` — but NOT `exitCode`. If an API boundary uses `JSON()` to serialize an error with a custom exit code, the override is lost.

**Impact:** API consumers using `JSON()` for HTTP error responses will not see the exit code override. This may be intentional (exit codes are a CLI concept, not an HTTP one), but it should be a documented decision.

### 3. AGENTS.md version header not bumped

The "API Surface (v0.5.0)" header was not updated. The go.mod doesn't carry a semver (Go modules use tags), so this is a cosmetic doc issue, but it implies the new APIs are part of v0.5.0 when they're unreleased.

---

## c) NOT STARTED

### Documentation (ZERO references to new APIs in these files)

| #   | File                                              | What's Missing                                                                             |
| --- | ------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| 1   | `SKILL.md`                                        | No mention of `WrapOnce`, `ExitCoder`, `WithExitCode`, `WithContextAny`, `safeCauseString` |
| 2   | `README.md`                                       | No mention of any new API in the feature table or examples                                 |
| 3   | `FEATURES.md`                                     | No feature inventory entry for the 4 new capabilities                                      |
| 4   | `CHANGELOG.md`                                    | No changelog entry for this session's changes                                              |
| 5   | `website/src/content/docs/api-reference.mdx`      | No API table entries for `WrapOnce`, `ExitCoder`, `WithExitCode`, `WithContextAny`         |
| 6   | `website/src/content/docs/guides/error-types.mdx` | No guide section on `ExitCoder` or `WrapOnce` patterns                                     |
| 7   | `example_test.go`                                 | No runnable `Example*` functions for any new API                                           |

### Missing API Variants

| #   | Item                                                               | Rationale                                                                             |
| --- | ------------------------------------------------------------------ | ------------------------------------------------------------------------------------- |
| 8   | **`WrapOncef`** — formatted variant of `WrapOnce`                  | BuildFlow has `Wrapf` alongside `Wrap`. The `f` variant is expected by Go convention. |
| 9   | **Family-specific `WrapOnce` variants** (e.g. `WrapOnceTransient`) | Probably YAGNI, but considered for completeness                                       |

### Missing Tests

| #   | Item                                                        | Rationale                                                                                                                                                 |
| --- | ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 10  | **Benchmark for `WrapOnce`**                                | `benchmark_test.go` has benchmarks for `Classify`, `ExitCode`, etc. — `WrapOnce` should be benchmarked to confirm the `errors.AsType` chain walk is fast. |
| 11  | **Fuzz test for `contextValueToString` / `WithContextAny`** | `fuzz_test.go` has fuzz tests for `ParseFamily`, `Classify`, etc. — `WithContextAny` accepts `any` and should be fuzzed with random types.                |
| 12  | **`errorfamilytest` assertions**                            | No `AssertExitCode(tb, err, want)` helper in the test subpackage. Consumers testing custom exit codes have to do it manually.                             |

### Integration Gaps

| #   | Item                                                        | Rationale                                                                                                                                                                                                                                                                              |
| --- | ----------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 13  | **Bridge `ClassifiedError` does not implement `ExitCoder`** | `bridge/bridge.go`'s `ClassifiedError` satisfies `Coded`, `Classified`, `Retryable`, `Contextual` — but NOT `ExitCoder`. If a bridged error needs a custom exit code, there's no way to attach one. May be intentional (bridge defers to family), but should be a documented decision. |
| 14  | **`stdlib.go` no exit code overrides**                      | `RegisterStdlibDefaults` registers family classifications but no exit code overrides. Some stdlib errors might benefit from non-standard exit codes (e.g. `os.ErrPermission` → exit 77 `EX_NOPERM` instead of family default 1).                                                       |

---

## d) TOTALLY FUCKED UP

### 1. Mysterious README.md change — NOT authored by me

```
git diff README.md:
-[![Go Report Card](https://goreportcard.com/badge/github.com/larsartmann/go-error-family)](https://goreportcard.com/report/github.com/larsartmann/go-error-family)
```

The **Go Report Card badge was removed** from `README.md`. I did NOT make this change. The git status at conversation start said "Status: clean". This appeared during the session from an unknown source — possibly a hook, formatter, or external process.

**Action needed:** Investigate before committing. Do NOT commit this change blindly.

### 2. No `git stash` or checkpoint during implementation

I implemented all 4 features across 6 files in one pass without intermediate checkpoints. If any single change had broken the build, rolling back would have been harder. Should have committed (or at least stashed) after each feature was verified independently.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Design

1. **`jsonError` should decide explicitly about `exitCode`** — Either add it (for CLI-over-HTTP use cases) or document why it's excluded (exit codes are CLI-only). Currently it's an accidental omission.
2. **`formatVerbose` should show `exitCode` when non-zero** — It's a debugging format; hiding the override defeats the purpose.
3. **`ExitCoder` interface naming** — The name `ExitCoder` could be confused with "the thing that generates exit codes" vs "an error that carries an exit code." BuildFlow used unexported `exitCoder` which avoids this. Consider documenting the semantic clearly.
4. **`contextValueToString` should handle `fmt.Stringer` and `error` types explicitly** — Currently falls through to `fmt.Sprint`, which works but misses the chance for a fast path on `String()` and `Error()` methods.
5. **`WrapOnce` should have a formatted variant** — `WrapOncef` is expected by Go convention.

### Testing

6. **No fuzz coverage for new input surfaces** — `WithContextAny` takes `any`, which is a fuzz-worthy boundary.
7. **No benchmark for `WrapOnce`** — The `errors.AsType` chain walk could be slow on deep chains.
8. **`contextValueToString` edge cases untested** — `[]byte`, `error`, `fmt.Stringer`, `nil` pointers, nested structs.
9. **No integration test for ExitCoder through `HandleError`** — We test `HandleErrorDetailed` but not the full `HandleErrorWithContext` → `resolveExitCode` path with a custom exit code.
10. **Test file naming** — `buildflow_learnings_test.go` is an unusual name. These features are now part of go-error-family, not "BuildFlow learnings." Should be split into `wraponce_test.go`, `exitcode_test.go`, `context_any_test.go`, `panic_recovery_test.go`.

### Documentation

11. **CHANGELOG.md has no entry** — This is a user-facing API addition.
12. **SKILL.md is stale** — The canonical API reference for this project doesn't mention any new APIs.
13. **README.md feature table is stale** — No mention of `WrapOnce`, `ExitCoder`, `WithContextAny`.
14. **Website docs are stale** — `api-reference.mdx` has no entries for the new APIs.
15. **No `example_test.go` entries** — Runnable examples are the Go way to document usage.

### Process

16. **No intermediate verification** — Should have run tests after EACH feature, not all at the end.
17. **No checkpoint commits** — All changes are uncommitted in one blob.
18. **README.md phantom change uninvestigated** — Should be caught and explained before any commit.

---

## f) Up to 50 Things We Should Get Done Next

#### Critical (blocking a clean commit)

1. **Investigate the README.md phantom change** — Who/what removed the Go Report Card badge? Is it safe to restore?
2. **Add `exitCode` to `formatVerbose`** — Show it when non-zero in `%+v` output
3. **Decide on `jsonError` + exitCode** — Add field or document exclusion
4. **Add CHANGELOG.md entry** for all 4 new APIs
5. **Rename `buildflow_learnings_test.go`** to topic-focused test files

#### High Priority (API completeness)

6. **Add `WrapOncef`** — formatted variant of WrapOnce
7. **Update SKILL.md** — document WrapOnce, ExitCoder, WithExitCode, WithContextAny, safeCauseString
8. **Update README.md feature table** — add new APIs
9. **Update FEATURES.md** — add DONE entries for the 4 new capabilities
10. **Update `api-reference.mdx`** — add API table entries
11. **Add `Example*` functions to `example_test.go`** — runnable docs for each new API
12. **Add `AssertExitCode` to `errorfamilytest`** — test helper for exit code assertions

#### Medium Priority (robustness)

13. **Add benchmark for `WrapOnce`** in `benchmark_test.go`
14. **Add fuzz test for `contextValueToString`** in `fuzz_test.go`
15. **Test `contextValueToString` with `fmt.Stringer`, `error`, `[]byte`** edge cases
16. **Test full `HandleErrorWithContext` path with ExitCoder override** (not just HandleErrorDetailed)
17. **Add `fmt.Stringer` and `error` cases to `contextValueToString` type switch**
18. **Test `WrapOnce` with deeply nested chains** (5+ levels of wrapping)
19. **Test `WithContextAny` copy-on-write preserves exit code** (cross-field preservation)
20. **Test `WithExitCode` preserves across `WithContext`/`WithContextMap`/`WithCause`** (all With* methods)

#### Bridge & Submodule

21. **Decide if `ClassifiedError` should implement `ExitCoder`** — Document decision either way
22. **If yes, add `WithExitCode` to bridge** — allow bridged errors to carry custom exit codes
23. **Add bridge test for ExitCoder interaction** — if implemented

#### Documentation Polish

24. **Update website guides** — `error-types.mdx` with ExitCoder pattern, `http-and-cli.mdx` with WrapOnce pattern
25. **Update website quick-start** — show WithExitCode in the CLI exit code example
26. **Add code comments to `resolveExitCode`** explaining the precedence chain
27. **Document ExitCoder precedence in SKILL.md** — ExitCoder > Family-based exit code
28. **Bump version reference in AGENTS.md** — "v0.5.0" → next version

#### Future Patterns from BuildFlow (not yet ported)

29. **Consider `MultiStepError` equivalent** — BuildFlow aggregates step failures with per-step exit codes and picks highest. go-error-family has `errors.Join` but no named-step aggregation.
30. **Consider display mode concept** — BuildFlow's `ErrorDisplayMode` (Usage/UsageOnly/Silent) controls CLI behavior. go-error-family has no equivalent; might be useful for CLI consumers.
31. **Consider `sentinelError` pattern** — BuildFlow's type-only sentinel matching enables `errors.Is(err, ErrValidation)` for any validation error. go-error-family's `Is()` matches code+family, which is more precise but less ergonomic for category matching.
32. **Consider `WrapOnce` family-specific variants** — `WrapOnceTransient`, etc. (likely YAGNI)
33. **Consider cockroachdb/errors integration** — BuildFlow uses it for stack traces. go-error-family currently has no stack trace story. The `diagnose` package fills some of this gap.

#### Stdlib & Registry

34. **Consider exit code overrides in `RegisterStdlibDefaults`** — e.g. `os.ErrPermission` → exit 77
35. **Add `Registry.RegisterExitCodeOverride`** — scoped exit code overrides (parallel to RegisterClassification)
36. **Test ExitCoder with custom Registry** — verify ExitCoder works through scoped registries

#### Code Quality

37. **Check if `exitCode` should be in `HandleResult`** — currently derived but not stored as a field that consumers can inspect
38. **Consider `ExitCodeOverride(code int)` as alternative API name** — clearer intent than `WithExitCode`
39. **Lint `WrapOnce` for `gochecknoglobals`** — ensure no new package-level vars were added
40. **Verify `gofumpt` compliance on all new code** — we had one gofumpt issue that was fixed; verify no others

#### Testing Infrastructure

41. **Add table-driven test for `resolveExitCode`** — covering ExitCoder-nil, ExitCoder-zero, ExitCoder-nonzero, no-ExitCoder
42. **Add property test: `WithExitCode(x).ExitCode() == x`** for all x != 0
43. **Add property test: `WithExitCode(0)` falls back to family** — zero means "unset"
44. **Add test: `WrapOnce(WrapOnce(err))` is idempotent** — double WrapOnce doesn't create layers
45. **Add test: panic recovery preserves error prefix** — the `[family:code] message` part survives

#### Release

46. **Tag a new version** after all docs are updated
47. **Update go.mod in BuildFlow** to use the new version (if BuildFlow wants WrapOnce from upstream instead of its own)
48. **Consider removing BuildFlow's `WrapOnce`** in favor of upstream (if it's now redundant)
49. **Consider removing BuildFlow's `exitCoder`** in favor of upstream `ExitCoder` interface
50. **Write migration guide** for BuildFlow consumers switching to upstream APIs

---

## g) Questions I CANNOT Answer Myself

### 1. Should `jsonError` (the `JSON()` output) include the custom `exitCode` field?

Exit codes are a CLI/POSIX concept. HTTP responses use status codes (which we already map via `HTTPStatus`). Adding `exitCode` to the JSON shape would leak a CLI concern into an API-boundary format. But some consumers run CLI tools behind HTTP APIs and might want the exit code. I cannot determine the intended audience for `JSON()` — is it HTTP-only, or CLI-over-HTTP too?

### 2. Should the bridge's `ClassifiedError` implement `ExitCoder`?

The bridge connects `samber/oops` (enrichment) with `go-error-family` (classification). `ClassifiedError` currently satisfies 4 interfaces. Adding `ExitCoder` would be a 5th — but oops errors don't inherently carry exit codes. I don't know if any bridge consumer needs custom exit codes on bridged errors, or if this should remain a concern of the concrete `*Error` type only.

### 3. What happened to the README.md Go Report Card badge?

The git status at conversation start was "clean", but `git diff README.md` now shows the Go Report Card badge was removed. I did not make this change. I don't know if a hook, formatter, or external process did this. Should I restore it, or was its removal intentional from a prior uncommitted action?
