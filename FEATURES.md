# Features

Honest inventory of what exists, what works, and what doesn't. Every claim is
verifiable against the code — citations point at the source.

**Last verified:** 2026-07-16 against v0.8.0

---

## Status Legend

| Status               | Meaning                                                  |
| -------------------- | -------------------------------------------------------- |
| FULLY_FUNCTIONAL     | Code present AND working (tests pass).                   |
| PARTIALLY_FUNCTIONAL | Ships but has known gaps, edge cases, or missing pieces. |
| BROKEN               | Code exists but does not work / is disabled / fails.     |
| PLANNED              | Designed or documented but **no code exists yet**.       |

---

## Root Package (`errorfamily`) — FULLY_FUNCTIONAL

The classification core. Zero third-party dependencies (stdlib only, including
experimental `encoding/json/v2` which requires `GOEXPERIMENT=jsonv2`).

### Classification

| Feature                                                                                              | Status           | Evidence                     |
| ---------------------------------------------------------------------------------------------------- | ---------------- | ---------------------------- |
| `Family` enum (5 values: Rejection, Conflict, Transient, Corruption, Infrastructure)                 | FULLY_FUNCTIONAL | `family.go`                  |
| `Family.Severity()` — total order for multi-error worst-case selection                               | FULLY_FUNCTIONAL | `family.go`                  |
| `Family.HTTPStatus()` — canonical family→HTTP mapping                                                | FULLY_FUNCTIONAL | `family.go`                  |
| `Family.RetryPolicy()` — advisory retry defaults                                                     | FULLY_FUNCTIONAL | `retry.go`                   |
| `Family.ExitCode()` — BSD sysexits.h codes                                                           | FULLY_FUNCTIONAL | `family.go`                  |
| `Family.Audience()` / `Family.Tone()` — presentation metadata                                        | FULLY_FUNCTIONAL | `family.go`                  |
| `Family` implements `encoding.TextMarshaler`/`TextUnmarshaler`                                       | FULLY_FUNCTIONAL | `family.go`                  |
| `Classify(err)` — 6-step pipeline (multi-error→Classified→Retryable→sentinels→classifiers→Transient) | FULLY_FUNCTIONAL | `classify.go`, `registry.go` |
| `Classify(nil)` → Rejection (intentional: nil = caller's fault)                                      | FULLY_FUNCTIONAL | `registry.go:Classify`       |
| Multi-error (`errors.Join`) → worst-severity wins, order-independent                                 | FULLY_FUNCTIONAL | `registry.go:Classify`       |
| `IsRetryable(err)` — binary retry signal                                                             | FULLY_FUNCTIONAL | `classify.go`                |
| `ExitCode(err)` — checks ExitCoder interface first, then family default                              | FULLY_FUNCTIONAL | `classify.go`                |
| `Code(err)` — one-liner code extraction via `Coded` interface                                        | FULLY_FUNCTIONAL | `classify.go`                |
| `Compose(errs...)` — thin wrapper around `errors.Join` (kept for backward compatibility)             | FULLY_FUNCTIONAL | `classify.go`                |
| `ParseFamily(string)` — case-insensitive, defaults to Transient                                      | FULLY_FUNCTIONAL | `family.go`                  |
| `ParseAudience(string)` — case-insensitive audience parsing                                          | FULLY_FUNCTIONAL | `family.go`                  |

### Error Construction

| Feature                                                                                                                  | Status           | Evidence          |
| ------------------------------------------------------------------------------------------------------------------------ | ---------------- | ----------------- |
| `New(family, code, msg)` / `Newf(family, code, fmt, args)`                                                               | FULLY_FUNCTIONAL | `constructors.go` |
| Family-specific `New*`: `NewRejection`, `NewConflict`, `NewTransient`, `NewCorruption`, `NewInfrastructure`              | FULLY_FUNCTIONAL | `constructors.go` |
| `Wrap(err, family, code, msg)` / `Wrapf(...)` — nil-safe (returns nil for nil err)                                       | FULLY_FUNCTIONAL | `constructors.go` |
| Family-specific `Wrap*`: `WrapRejection`, `WrapConflict`, `WrapTransient`, `WrapCorruption`, `WrapInfrastructure`        | FULLY_FUNCTIONAL | `constructors.go` |
| Formatted `Wrap{Family}f`: `WrapRejectionf`, `WrapConflictf`, `WrapTransientf`, `WrapCorruptionf`, `WrapInfrastructuref` | FULLY_FUNCTIONAL | `constructors.go` |
| `WrapOnce(err, family, code, msg)` / `WrapOncef(...)` — idempotent wrap (prevents double-wrapping)                       | FULLY_FUNCTIONAL | `constructors.go` |

### Error Struct (Reference Implementation)

| Feature                                                                                            | Status           | Evidence   |
| -------------------------------------------------------------------------------------------------- | ---------------- | ---------- |
| `Error` type implementing `Coded`, `Classified`, `Contextual`, `Retryable`, `fmt.Formatter`        | FULLY_FUNCTIONAL | `error.go` |
| `WithContext(key, value)` / `WithContextf(key, fmt, args)` / `WithContextMap(map)` — copy-on-write | FULLY_FUNCTIONAL | `error.go` |
| `WithContextAny(key, value any)` — type-safe context for non-string values                         | FULLY_FUNCTIONAL | `error.go` |
| `WithCause(err)` / `WithTimestamp(ts)` / `WithExitCode(code)` — copy-on-write                      | FULLY_FUNCTIONAL | `error.go` |
| `Error.Is(target)` — matches on code + family (not message)                                        | FULLY_FUNCTIONAL | `error.go` |
| `Error.JSON()` — canonical `{family,code,message,context,retryable,timestamp}`                     | FULLY_FUNCTIONAL | `error.go` |
| `Error.Summary()` — structured one-liner                                                           | FULLY_FUNCTIONAL | `error.go` |
| `Error.Format(%v`, `%+v`, `%s`) — verbose mode with context                                        | FULLY_FUNCTIONAL | `error.go` |

### Registry

| Feature                                                                                         | Status           | Evidence                     |
| ----------------------------------------------------------------------------------------------- | ---------------- | ---------------------------- |
| `Registry` type — injectable sentinels + classifiers + templates                                | FULLY_FUNCTIONAL | `registry.go`                |
| `NewRegistry()` / `DefaultRegistry`                                                             | FULLY_FUNCTIONAL | `registry.go`                |
| `Registry.Clone()` — deep-copy for inherit-and-extend                                           | FULLY_FUNCTIONAL | `registry.go`                |
| `RegisterClassification(sentinel, family)` / `RegisterClassifications(map)` — batch             | FULLY_FUNCTIONAL | `classify.go`, `registry.go` |
| `RegisterClassifier(func(error) (Family, bool))` / `RegisterClassifiers(...)` — predicate-based | FULLY_FUNCTIONAL | `classify.go`, `registry.go` |
| `UnregisterClassification(sentinel)` — for test cleanup                                         | FULLY_FUNCTIONAL | `classify.go`, `registry.go` |
| `RegisterTemplate(code, tmpl)` / `RegisterTemplates(map)` / `UnregisterTemplate(code)`          | FULLY_FUNCTIONAL | `handle.go`, `registry.go`   |
| `TemplateForCode(code)` — registry→builtin lookup without CLI pipeline                          | FULLY_FUNCTIONAL | `handle.go`, `registry.go`   |
| Lock-free reads via `atomic.Pointer` (copy-on-write for sentinels + classifiers)                | FULLY_FUNCTIONAL | `registry.go`                |
| `RegisterStdlibDefaults(reg)` — context/sql/os error taxonomy                                   | FULLY_FUNCTIONAL | `stdlib.go`                  |

### CLI Boundary

| Feature                                                                                             | Status           | Evidence    |
| --------------------------------------------------------------------------------------------------- | ---------------- | ----------- |
| `HandleError(err)` — classify, format, stderr write, return exit code                               | FULLY_FUNCTIONAL | `handle.go` |
| `HandleErrorWithContext(ctx, err, cfg)` — context-propagating handler                               | FULLY_FUNCTIONAL | `handle.go` |
| `HandleErrorWithConfig(err, cfg)` — delegates to Context variant                                    | FULLY_FUNCTIONAL | `handle.go` |
| `HandleErrorDetailed(err)` / `HandleErrorDetailedWithConfig(err, cfg)` — structured `*HandleResult` | FULLY_FUNCTIONAL | `handle.go` |
| `HandleConfig`: `Output`, `Registry`, `TemplateOverride`, `DiagnosticFunc`, `OnDiagnosed`           | FULLY_FUNCTIONAL | `handle.go` |
| Message templates: What / Why / Fix / WayOut with `{key}` substitution                              | FULLY_FUNCTIONAL | `handle.go` |

### HTTP Boundary

| Feature                                                                           | Status           | Evidence  |
| --------------------------------------------------------------------------------- | ---------------- | --------- |
| `HTTPStatus(err)` — classify→status-code                                          | FULLY_FUNCTIONAL | `http.go` |
| `HTTPHandler(fn)` — net/http middleware writing safe JSON (no `err.Error()` leak) | FULLY_FUNCTIONAL | `http.go` |

### Structured Logging

| Feature                                                                                                       | Status           | Evidence |
| ------------------------------------------------------------------------------------------------------------- | ---------------- | -------- |
| `LogError(err, logger)` / `LogErrorContext(ctx, err, logger)` — slog with family/code/retryable/context attrs | FULLY_FUNCTIONAL | `log.go` |

### Consumer Interfaces

| Feature                                                       | Status           | Evidence        |
| ------------------------------------------------------------- | ---------------- | --------------- |
| `Coded` (`ErrorCode() string`)                                | FULLY_FUNCTIONAL | `interfaces.go` |
| `Classified` (`ErrorFamily() Family`)                         | FULLY_FUNCTIONAL | `interfaces.go` |
| `Contextual` (`ErrorContext() map[string]string`)             | FULLY_FUNCTIONAL | `interfaces.go` |
| `Retryable` (`IsRetryable() bool`)                            | FULLY_FUNCTIONAL | `interfaces.go` |
| `ExitCoder` (`ExitCode() int`) — per-error exit code override | FULLY_FUNCTIONAL | `interfaces.go` |
| All five embed `error` (required for `errors.AsType[T]()`)    | FULLY_FUNCTIONAL | `interfaces.go` |

---

## `errorfamilytest` Subpackage — FULLY_FUNCTIONAL

Test assertion helpers mirroring `net/http/httptest`. Keeps `testing` out of production code.

| Feature                              | Status           | Evidence                             |
| ------------------------------------ | ---------------- | ------------------------------------ |
| `AssertFamily(tb, err, want)`        | FULLY_FUNCTIONAL | `errorfamilytest/errorfamilytest.go` |
| `AssertCode(tb, err, want)`          | FULLY_FUNCTIONAL | `errorfamilytest/errorfamilytest.go` |
| `AssertRetryable(tb, err, want)`     | FULLY_FUNCTIONAL | `errorfamilytest/errorfamilytest.go` |
| `AssertContext(tb, err, key, want)`  | FULLY_FUNCTIONAL | `errorfamilytest/errorfamilytest.go` |
| `AssertContextMissing(tb, err, key)` | FULLY_FUNCTIONAL | `errorfamilytest/errorfamilytest.go` |
| `AssertExitCode(tb, err, want)`      | FULLY_FUNCTIONAL | `errorfamilytest/errorfamilytest.go` |

---

## `diagnose` Module — FULLY_FUNCTIONAL

Concurrent diagnostic rules (zero-dep core). Separate Go module.

| Feature                                                                                                                        | Status           | Evidence                                      |
| ------------------------------------------------------------------------------------------------------------------------------ | ---------------- | --------------------------------------------- |
| `Runner` — concurrent rule execution, confidence-sorted results                                                                | FULLY_FUNCTIONAL | `diagnose/diagnose.go`                        |
| `NewRunner(rules...)` / `DefaultRunner()` / `RunAuto(ctx, err)`                                                                | FULLY_FUNCTIONAL | `diagnose/diagnose.go`                        |
| `DiagnosticRule` interface (`Name`, `Applicable`, `Run`)                                                                       | FULLY_FUNCTIONAL | `diagnose/diagnose.go`                        |
| `DiagnosticResult` with `Fix struct{Summary, Command}` (structured)                                                            | FULLY_FUNCTIONAL | `diagnose/diagnose.go`                        |
| `CommandRunner` interface + `DefaultCommandRunner` (injectable command execution)                                              | FULLY_FUNCTIONAL | `diagnose/diagnose.go`, `diagnose/command.go` |
| `RunCommand(...)` / `CommandExists(name)` — exported for rule authors                                                          | FULLY_FUNCTIONAL | `diagnose/command.go`                         |
| `RuleSpec` — data-driven rule matching (ContextKeys, CodeContains, ContextSubstr, Extra)                                       | FULLY_FUNCTIONAL | `diagnose/helpers.go`                         |
| Matching helpers: `HasContextKey`, `ContextValue`, `ResolveContextKey`, `HasContextSubstring`, `FamilyIs`, `ErrorCodeContains` | FULLY_FUNCTIONAL | `diagnose/helpers.go`                         |
| `FilesystemRule` — path existence, permissions, writability                                                                    | FULLY_FUNCTIONAL | `diagnose/rules_filesystem.go`                |
| `NetworkRule` — DNS resolution, TCP connectivity                                                                               | FULLY_FUNCTIONAL | `diagnose/rules_network.go`                   |
| `MockCommandRunner` — shared deterministic mock                                                                                | FULLY_FUNCTIONAL | `diagnose/mock.go`                            |

---

## `diagnose/git` Submodule — FULLY_FUNCTIONAL

| Feature                                                      | Status           | Evidence                    |
| ------------------------------------------------------------ | ---------------- | --------------------------- |
| `GitRule` — repo state, merge conflicts, remote reachability | FULLY_FUNCTIONAL | `diagnose/git/rules_git.go` |

---

## `diagnose/postgres` Submodule — FULLY_FUNCTIONAL

| Feature                                                                   | Status           | Evidence                              |
| ------------------------------------------------------------------------- | ---------------- | ------------------------------------- |
| `PostgresRule` — `pg_isready`, TCP connectivity, start command suggestion | FULLY_FUNCTIONAL | `diagnose/postgres/rules_postgres.go` |

---

## `agent` Module — FULLY_FUNCTIONAL

Analysis-only debug agent. Separate Go module (depends on root + diagnose).

| Feature                                                                               | Status           | Evidence         |
| ------------------------------------------------------------------------------------- | ---------------- | ---------------- |
| `DebugAgent` interface (`Analyze(ctx, err, diagnosis)`)                               | FULLY_FUNCTIONAL | `agent/agent.go` |
| `New(Config)` — constructor                                                           | FULLY_FUNCTIONAL | `agent/agent.go` |
| `AgentResult`: `RootCause`, `Confidence`, `Explanation`, `FixSteps`                   | FULLY_FUNCTIONAL | `agent/agent.go` |
| `FixStep`: `Description`, `Command`, `Rationale` — consumer executes, not the library | FULLY_FUNCTIONAL | `agent/agent.go` |
| `Config.Enabled` returns `(nil, error)` when disabled                                 | FULLY_FUNCTIONAL | `agent/agent.go` |

---

## `bridge` Module — FULLY_FUNCTIONAL

Connects go-error-family with `samber/oops`. Separate Go module (depends on both).

| Feature                                                                                         | Status           | Evidence             |
| ----------------------------------------------------------------------------------------------- | ---------------- | -------------------- |
| `Wrap(err, family)` — attach Family preserving OopsError context                                | FULLY_FUNCTIONAL | `bridge/bridge.go`   |
| `AutoWrap(err)` — infer Family from oops tags→domain→Transient                                  | FULLY_FUNCTIONAL | `bridge/classify.go` |
| `InferFamily(err)` — tags→domain→Transient (fail-open)                                          | FULLY_FUNCTIONAL | `bridge/classify.go` |
| `ClassifiedError` — satisfies `Classified`, `Coded`, `Retryable`, `Contextual`, `fmt.Formatter` | FULLY_FUNCTIONAL | `bridge/bridge.go`   |

---

## `examples` Module — FULLY_FUNCTIONAL

Separate Go module so root stays zero-dependency.

| Feature                                               | Status           | Evidence                    |
| ----------------------------------------------------- | ---------------- | --------------------------- |
| `cmd/cli` — CLI boundary handler example              | FULLY_FUNCTIONAL | `examples/cmd/cli/`         |
| `cmd/http` — HTTP middleware with status code mapping | FULLY_FUNCTIONAL | `examples/cmd/http/`        |
| `cmd/custom_rule` — writing your own DiagnosticRule   | FULLY_FUNCTIONAL | `examples/cmd/custom_rule/` |

---

## Test Coverage (verified 2026-07-13)

| Package              | Coverage |
| -------------------- | -------- |
| root (`errorfamily`) | 97.3%    |
| `errorfamilytest`    | 95.2%    |
| `agent`              | 100.0%   |
| `bridge`             | 94.1%    |
| `diagnose` (core)    | 83.9%    |
| `diagnose/git`       | 98.5%    |
| `diagnose/postgres`  | 80.3%    |

All packages at 80%+. Fuzz tests: `FuzzParseFamily`, `FuzzParseFamilyRoundTrip`, `FuzzClassify`, `FuzzClassifyPlainError`, `FuzzErrorFormatting` (root); `FuzzFormat` (bridge).

---

## Known Gaps

- **No per-error HTTP status override** — `Family.HTTPStatus()` is family-level only. Consumers needing 404 vs 400 within Rejection must handle it in their HTTP layer. (Design decision pending — see consumer feedback.)
- **`Classify(nil)` returns Rejection** — intentional but debated. Some consumers argue it should be Transient (fail-open) or Infrastructure (programming error). This is a design decision, not a bug.
- **Constructor context ergonomics** — `.WithContext().WithContext()` chains are verbose. No builder pattern or variadic context yet. Consumers build `errkit`-style helpers.
- **`encoding/json/v2` requirement** — the root module uses Go's experimental json/v2, requiring `GOEXPERIMENT=jsonv2`. This will drop when json/v2 becomes stable in a future Go release.
