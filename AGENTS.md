# go-error-family

Structured error protocol library. Library only — no `main`, no build system, no external deps. Full API reference: `SKILL.md`.

**Status:** All tests pass (root + bridge + submodules), 0 lint issues, 0 race conditions
**Workspace modules:** root (zero-dep), `agent`, `bridge` (oops integration), `diagnose`, `diagnose/git`, `diagnose/postgres`, `examples`, `website`

## Quick Start

```bash
go test ./... -count=1 -timeout 120s -race   # all tests
golangci-lint run ./...                        # lint (all modules)
go build ./...                                 # build check
```

## Architecture Decision: Libraries Classify, Applications Enrich

**go-error-family (classification) and samber/oops (enrichment) are complementary, not competing.** The `bridge/` package is the seam where they meet.

- **LIBRARY code** (clients, SDKs, domain packages) imports `go-error-family` only and returns classified errors. A library knows its own domain contract (404 = Rejection, timeout = Transient) but must NOT presume the application's observability stack — so it never imports oops.
- **APPLICATION code** imports oops for enrichment (stack traces, trace IDs, request context) and, if it also needs behavioral decisions, wraps library errors via the bridge.

The classification protocol is the **four interfaces** (`Coded`/`Classified`/`Contextual`/`Retryable`) — the sole public contract. `Error` is a reference implementation, not the contract; domain types implement only the interfaces they need.

## Surprising Behaviors

- **`Classify(nil)` returns `Rejection`**, not a zero value. Intentional: nil error = caller's fault.
- **`Classify` defaults unknown errors to `Transient`** (retryable). Fail-open design — unknown errors get retried. Same for `ParseFamily` with unrecognized strings.
- **`errors.Is` matches on `code + family` only**, ignoring message. Two `*Error`s with different messages but same code and family will match.
- **`Wrap(nil, ...)` returns `nil`** — nil-safe, but means you can't construct an error wrapping nil.
- **`WithContext`/`WithCause`/`WithTimestamp`/`WithExitCode` are copy-on-write** — they return a NEW `*Error`, not the same pointer. Safe to chain from shared/sentinel errors. Do NOT assume identity preservation.
- **Template placeholders use `{key}`, not `{{.key}}`** — the old syntax collided with Go's `text/template`. Migration: replace all `{{.key}}` with `{key}` in registered templates.
- **Consumer interfaces (`Coded`, `Classified`, `Contextual`, `Retryable`, `ExitCoder`) embed `error`** — required for Go 1.26's `errors.AsType[T]()`. Don't remove the embedding.
- **`HandleErrorWithContext` is the canonical entry point** — `HandleError` and `HandleErrorWithConfig` delegate to it. Always prefer the context-accepting variant when you have a `context.Context`.
- **Package-level `Classify`/`RegisterClassification`/`RegisterTemplate` delegate to `DefaultRegistry`** — backward compatible. For test isolation or scoped handling, construct a `NewRegistry()` and pass it via `HandleConfig.Registry`.
- **`CommandRunner` defaults to `DefaultCommandRunner{}`** — rules with a nil `Runner` field use the real system commands. Tests inject mocks.
- **`Error()`/`Summary()` use `safeCauseString` for panic recovery** — if a wrapped cause's `Error()` method panics (e.g. nil internal values in third-party error types), the panic is caught and the cause message is omitted rather than crashing the process.
- **`ExitCode(err)` checks `ExitCoder` before family** — an error implementing `ExitCoder` with a non-zero code overrides the family-based BSD exit code. `*Error` always implements `ExitCoder`, but returns 0 (meaning "use family default") unless `WithExitCode` was called.
- **`WrapOnce` is idempotent** — if the error chain already contains a `*Error`, it is returned unchanged. This prevents double-wrapping at API boundaries.

## API Surface (v0.8.0)

**Family adapters** (in `family.go` / `retry.go`, all single-source-of-truth via `familyData`):

- `Family.Severity() int` — total order for multi-error classification (Transient<Rejection<Conflict<Infrastructure<Corruption).
- `Family.HTTPStatus() int` — canonical family→HTTP status (Rejection→400, Conflict→409, Transient→503, Corruption→500, Infrastructure→503).
- `Family.RetryPolicy() RetryPolicy` — advisory defaults (Transient: 3 attempts, 100ms-5s; others: single attempt). Library does not run the loop.

**Error methods** (`error.go`): `WithContextMap(map)`, `WithContextf(key, fmt, args)`, `WithContextAny(key, any)` (type-switched to string), `WithExitCode(int)` (overrides family-based exit code), `WithHTTPStatus(int)` (overrides family-based HTTP status), `ExitCode() int` (satisfies `ExitCoder`), `HTTPStatus() int` (satisfies `HTTPStatuser`), `JSON() ([]byte, error)` (canonical `{family,code,message,context,retryable,timestamp}` for API boundaries). All `With*` methods are copy-on-write.

**Constructors** (`constructors.go`): `WrapOnce(err, family, code, msg)` / `WrapOncef(...)` — idempotent wrapping; returns the existing `*Error` unchanged if the error chain already contains one. Prevents double-wrapping at API boundaries.

**Registry** (`registry.go`): `Clone()` (deep-copy, inherit-and-extend), `RegisterTemplates(map)` (batch, parity with `RegisterClassifications`).

**Stdlib taxonomy** (`stdlib.go`): `RegisterStdlibDefaults(reg)` — maps context/sql/os errors with documented rationale for ambiguous cases (DeadlineExceeded→Transient, Canceled→Rejection, etc.).

## Consumer-Feedback APIs (added 2026-07-05)

Driven by SEC and browser-history integration feedback. All use stdlib only (`net/http`, `log/slog`, `testing`); the root stays zero-dep.

- **`Code(err) string`** (`classify.go`) — public code extraction (wraps `errors.AsType[Coded]`). `handle.go`'s `extractCode` delegates to it.
- **`RegisterClassifier` / `RegisterClassifiers`** (`classify.go` + `Registry.RegisterClassifier` in `registry.go`) — predicate-based classification for dynamic errors (`*sqlite.Error`). Stored in `Registry.classifiers atomic.Pointer[[]Classifier]`, copy-on-write, lock-free reads. `Registry.Clone()` copies them. No `UnregisterClassifier` exists (Go funcs aren't comparable) — use a custom `Registry` for isolation.
- **`RegisterClassificationType[T error]` / `RegisterClassificationTypeFor[T error]`** (`classify.go`) — generic type-based classifier sugar over `RegisterClassifier`. Two top-level functions because Go doesn't allow type parameters on methods: `RegisterClassificationType[T](family)` delegates to `DefaultRegistry`, `RegisterClassificationTypeFor[T](r, family)` targets a custom `Registry`.
- **`TemplateForCode(code) (MessageTemplate, bool)`** (`registry.go` + package-level in `handle.go`) — registry-then-builtin template lookup, for HTTP/gRPC consumers.
- **`Wrap{Family}f`** (`constructors.go`) — `WrapRejectionf`, `WrapConflictf`, `WrapTransientf`, `WrapCorruptionf`, `WrapInfrastructuref`. Nil-safe.
- **`HTTPStatus(err)` / `HTTPHandler(fn)`** (`http.go`) — net/http middleware. Checks `HTTPStatuser` interface first (per-error override via `WithHTTPStatus`), then falls back to `Classify(err).HTTPStatus()`. **`HTTPHandler` NEVER leaks `err.Error()`** — the response message comes only from a registered `MessageTemplate`; otherwise just family+code. This is deliberate (consumers value no internal leakage).
- **`LogError(err, *slog.Logger)` / `LogErrorContext`** (`log.go`) — Transient→Warn, others→Error; nil error is no-op; nil logger→`slog.Default`. Logs `family`, `code`, `retryable`, and `context.<key>` attrs.
- **`errorfamilytest`** subpackage — `AssertFamily`/`AssertCode`/`AssertRetryable`/`AssertContext`/`AssertContextMissing`/`AssertExitCode`. Mirrors `httptest`: keeps `testing` out of the production package.

## BuildFlow-Inspired APIs (added 2026-07-16)

Learned from BuildFlow's `modules/errors/` package — patterns proven in a production CLI build tool:

- **`ExitCoder` interface** (`interfaces.go`) — `error` + `ExitCode() int`. When `ExitCode(err)` or `HandleError*` encounters an error implementing this with a non-zero code, that code overrides the family-based BSD exit code. Allows per-error exit code overrides without changing the family classification. `*Error` satisfies this via `WithExitCode(code)`.
- **`HTTPStatuser` interface** (`interfaces.go`) — `error` + `HTTPStatus() int`. When `HTTPStatus(err)` encounters an error implementing this with a non-zero status, that status overrides the family-based HTTP default. Allows per-error HTTP status overrides (e.g. 404 for a not-found Rejection instead of the family default 400). `*Error` satisfies this via `WithHTTPStatus(status)`. Mirrors the `ExitCoder`/`WithExitCode` pattern exactly.
- **`WrapOnce(err, family, code, msg)`** (`constructors.go`) — idempotent wrapping. Uses `errors.AsType[*Error]` to detect existing classified errors in the chain. Prevents the `[transient:db.timeout] outer: [transient:db.timeout] inner: cause` double-wrap anti-pattern at API boundaries. Nil-safe.
- **`WithContextAny(key, value any)`** (`error.go`) — typed context values. Accepts `any` and converts via a type switch (string, int, int64, uint, uint64, float64, bool, []byte, time.Time, error, nil → empty, fallback `fmt.Sprint`). Ergonomic alternative to `fmt.Sprintf` for scalar values.
- **`safeCauseString`** (`error.go`) — panic recovery in `Error()`, `Summary()`, and `formatVerbose()`. Uses `defer/recover` around `cause.Error()` to guard against misbehaving third-party error types that panic on nil internal values. The error message renders without the cause instead of crashing.

## Classification Precedence

`Classify(err)` checks in order — first match wins:

1. **Multi-error** (`errors.Join`) → classify each sub-error, pick the **worst by severity** (see below)
2. `Classified` interface → `ErrorFamily()`
3. `Retryable` interface → infer `Transient` (true) or `Rejection` (false)
4. Registered sentinels via `errors.Is` chain walk (atomic.Pointer to immutable map — lock-free, allocation-free iteration)
5. Registered classifiers (`RegisterClassifier`) — predicate funcs for dynamic errors (e.g. `*sqlite.Error`); stored lock-free behind `atomic.Pointer[[]Classifier]`, copy-on-write; run in registration order, first `ok=true` wins
6. Default → `Transient`

This means a type implementing both `Classified` and `Retryable` will use `Classified` and ignore `Retryable`. Registering a sentinel for an error that already implements `Classified` has no effect. Classifiers only run when all earlier steps miss, so the hot path is unaffected.

**Multi-error behavior:** For `errors.Join(err1, err2, ...)`, each sub-error is classified recursively and the result is the **highest-severity** sub-error (`Family.Severity()` total order: Transient(1) < Rejection(2) < Conflict(3) < Infrastructure(4) < Corruption(5)). This is deterministic regardless of join argument order and remains fail-closed: if any sub-error is non-Transient (severity > 1), the joined result is non-Transient.

## Registry Pattern

The library uses an injectable `Registry` type (`registry.go`) that holds both classification sentinels and message templates. The zero value is not usable — use `NewRegistry()`.

- **`DefaultRegistry`** is a package-level `*Registry` used by all convenience functions (`Classify`, `RegisterClassification`, `RegisterTemplate`, etc.) and by `HandleError` when `HandleConfig.Registry` is nil.
- **Custom registries** enable test isolation (no `t.Cleanup(Unregister...)` needed) and scoped error handling within a single binary. Pass via `HandleConfig.Registry`.
- **Thread-safety:** `Registry.sentinels` is an `atomic.Pointer[sentinelMap]` to an immutable snapshot: reads (the `Classify` hot path) load the pointer once and iterate lock-free and allocation-free; rare writers serialize under the write lock and publish a new snapshot via copy-on-write. At 50 registered sentinels this is ~285 ns/0 allocs (was ~1330 ns/3 allocs/1.8KB under the old RLock+copy approach).
- **`resolveSuggestedFix`** and **`renderCLI`** share one `resolveTemplate(code, cfg, reg)` helper (override → registry → built-in default). Templates are cohesive units — What/Why/Fix belong together, never mixed across sources.

## Agent Is Analysis-Only

The `DebugAgent` interface has a single method: `Analyze`. It produces root cause analysis and `FixStep` suggestions. The library does NOT execute fixes — the consumer decides what to do with `FixStep.Command`. The `Involvement` and `RiskLevel` concepts belong to the consumer, not the library.

## Diagnostic Rule Pattern

When adding a new `DiagnosticRule`, use the matching helpers from the `diagnose` package: `HasContextKey`, `ContextValue`, `ResolveContextKey`, `HasContextSubstring`, `FamilyIs`, `ErrorCodeContains`. Use execution helpers `RunCommand` and `CommandExists` for system checks. Rules run concurrently via `Runner.Run` and results sort by confidence descending.

**Structured Fix:** `DiagnosticResult` carries a `Fix struct{Summary, Command string}` (not freeform prose). Rules populate both fields at construction time so the agent reads `Fix.Command` directly — there is no prose-parsing heuristic. When adding a rule, set `result.Fix = diagnose.Fix{Summary: "...", Command: "exact shell command"}`.

**Submodules:** `GitRule` lives in `github.com/larsartmann/go-error-family/diagnose/git`, `PostgresRule` in `github.com/larsartmann/go-error-family/diagnose/postgres`. `DefaultRunner()` only includes zero-dep rules (`FilesystemRule`, `NetworkRule`).

## Partial Success

Not a library type — partial success is a consumption pattern, not a classification concern. See SKILL.md for the recipe (collect outcomes, `Classify` each failure, pick worst family for exit code). The library provides the classification vocabulary; consumers compose the collection strategy.

## Test Coverage

| Package              | Coverage |
| -------------------- | -------- |
| root (`errorfamily`) | 97.0%    |
| `errorfamilytest`    | 96.3%    |
| `agent`              | 100.0%   |
| `bridge`             | 95.6%    |
| `diagnose` (core)    | 83.9%    |
| `diagnose/git`       | 98.5%    |
| `diagnose/postgres`  | 80.3%    |

All packages at 80%+; root and `diagnose/git` near-complete. (`errorfamilytest` is intentionally thin — assertion helpers delegating to the main package.)

## Fuzz Tests

`fuzz_test.go` (root) contains: `FuzzParseFamily`, `FuzzParseFamilyRoundTrip`, `FuzzClassify`, `FuzzClassifyPlainError`, `FuzzErrorFormatting`, `FuzzApplyContext`, `FuzzWrapOnce`, `FuzzContextValueToString`, `FuzzWithExitCode`, `FuzzWithHTTPStatus`, `FuzzRegisterClassificationType`. `bridge/fuzz_test.go` contains: `FuzzInferFamily`, `FuzzAutoWrap`, `FuzzWrapRoundTrip`, `FuzzWrapOopsRoundTrip`, `FuzzFormat`.

## Adoption Reality (audited 2026-07-23)

The library is **heavily adopted** — 50+ projects across the ecosystem directly import the root package. Core APIs are deeply embedded:

| API Surface                         | External Call Sites |
| ----------------------------------- | ------------------- |
| `New`/`Wrap` constructors           | ~750                |
| `Classify()`                        | ~130                |
| `IsRetryable()`                     | ~40                 |
| `HandleError*`                      | ~35                 |
| `RegisterClassification`/`Template` | ~27                 |
| `ExitCode()`                        | ~24                 |

**But consumer-facing enrichment APIs have minimal adoption:** `LogError` (~3 consumers), `HTTPHandler`/`HTTPStatus` (~5), `errorfamilytest` (~3), `diagnose` (~3). The classification core is the bread and butter; the higher-level boundary handlers haven't spread proportionally.

## Bridge Submodule (`bridge/`) — Zero Consumers

Connects go-error-family with `samber/oops`. Separate module with its own `go.mod` (depends on both libraries). The root package remains zero-dependency.

**Adoption status: ZERO external consumers.** Root causes:

1. **Near-zero oops adoption in the ecosystem** — only ~1 project uses `samber/oops` at all. The bridge connects two libraries, but the second one barely exists.
2. **The enrichment layer is skipped in practice** — consumers call `HandleError(err)` at the top with whatever error bubbled up, without adding stack traces, trace IDs, or domain context. The typical flow is classify→handle, not classify→enrich→handle.
3. **No reference implementation** — the "libraries classify, applications enrich" architecture is architecturally sound but no project demonstrates the full classify→enrich→handle flow end-to-end. The one project using both libraries combines them at separate layers that never need the bridge.

The bridge is correct, tested (95.6%), and fuzzed. The gap is **demand and demonstration, not quality**. It needs a reference implementation showing the oops + bridge + error-family stack in a real application to prove the pattern.

| API                        | Purpose                                                                               |
| -------------------------- | ------------------------------------------------------------------------------------- |
| `bridge.Wrap(err, family)` | Attach a Family to any error, preserving OopsError context                            |
| `bridge.AutoWrap(err)`     | Infer Family from oops metadata (tags + domain), then wrap                            |
| `bridge.InferFamily(err)`  | Derive Family from oops tags (explicit) → domain (structural) → Transient (fail-open) |
| `ClassifiedError`          | Embeds `oops.OopsError`; satisfies `Classified`, `Coded`, `Retryable`, `Contextual`   |

**Tag overrides** (checked first): `retryable`, `transient`, `conflict`, `corruption`/`corrupted`, `rejection`/`rejected`, `infrastructure`/`infra`.
**Domain defaults** (checked second): `validation`/`auth` → Rejection, `database`/`network`/`cache`/`queue` → Transient, `storage`/`infra`/`startup` → Infrastructure, `data`/`schema`/`migration` → Corruption.

**Surprising:** `Wrap(nil, family)` returns a ClassifiedError with zero OopsError — `Error()` returns `[family]`, `Unwrap()` returns nil. This is intentional: nil is still classifiable.

## Lint Configuration

- `bridge` package-level lookup tables (`domainDefaults`, `tagOverrides`) suppress `gochecknoglobals` via inline `//nolint` — same pattern as root's immutable lookup tables.

- G304 (gosec file inclusion) is excluded for `diagnose/rules_filesystem.go` via `.golangci.yml` path-based exclusion — `os.Open(path)` and `os.Create(testFile)` are intentional in diagnostic rules.
- Do NOT use `//nolint:gosec` directives for G304 in the diagnose package — the `.golangci.yml` exclusion handles it. Inline nolint directives break when `golines` wraps lines.
- `ContextKey` type replaces raw strings in rule specs. `CodeContains` fields still use raw strings (different semantic — substring matching on error codes, not context keys).
- `CommandRunner` interface allows mock injection; `DefaultCommandRunner` wraps real system calls.
- `gochecknoglobals` is enabled but suppressed via `//nolint:gochecknoglobals` on each legitimate package-level var (mutex-protected registries, immutable lookup tables, rule specs) — the BuildFlow pre-commit auto-configure hook re-enables it if disabled in `.golangci.yml`.
- `exhaustruct` is enabled but most project types are excluded via `.golangci.yml` because they have intentional optional fields (HandleConfig, MessageTemplate, DiagnosticResult, etc.). Test files also exclude exhaustruct.
- `flake.nix` uses `pkgs.go_1_26` as `goPkg` — do NOT use `let goPkg = goPkg;` (infinite recursion).
- The root module uses stdlib `encoding/json` (not json/v2). The json/v2 experiment was reverted — the library requires no special environment variables.
- `lookupRegistered` is now `Registry.lookupSentinel` — still snapshots the map before iterating, `errors.Is` runs lock-free. No deadlock possible.
- `HandleConfig.Registry` field added — when nil, falls back to `DefaultRegistry`. `resolveSuggestedFix` checks registry templates alongside built-in defaults.
- `Registry` is excluded from `exhaustruct` via `.golangci.yml` — the `mu` field (sync.RWMutex) has a correct zero value set by `NewRegistry()`.
- `HandleConfig.Diagnose` bool was removed — diagnostics run whenever `DiagnosticFunc` is set. No separate enable flag.
- `diagnose.Status` has `IsValid()` matching `Family.IsValid()` pattern.
- `diagnose.sortByConfidence` uses `slices.SortFunc` (Go 1.26 stdlib).
- CI now has explicit `bridge/` test and lint steps, plus an examples build step (`working-directory: ./examples`).
- `familyInfo` includes `Audience` field — adding a new Family truly requires only one entry in `familyData`.
- `NetworkRule.Run` returns `StatusUnknown` when no host found in error context (prevents undefined DNS behavior).
- `Audience.IsValid()` mirrors `Family.IsValid()` and `Status.IsValid()` — all three enum types have consistent validation.
- `ParseAudience` and `ParseStatus` mirror `ParseFamily` — case-insensitive string parsing for all enums.
- `Family` and `Audience` implement `encoding.TextMarshaler`/`TextUnmarshaler` for YAML/JSON config.
- `agent.Config.Enabled` now returns `(nil, error)` instead of synthetic result — calling `Analyze` on a disabled agent is a programming error.
- **depguard** allows `$gostd`, `$module`, `github.com/larsartmann/go-error-family` (all workspace modules), and `github.com/samber/oops` (bridge dependency). The root module's zero-dep guarantee is enforced by `go.mod` + CI's `GOWORK=off go build`, not depguard alone — depguard's `files` patterns are working-directory-relative and can't distinguish modules in a workspace.
- **Test files** (`_test.go`) exclude: `err113`, `testpackage`, `fatcontext`, `funlen`, `containedctx` — internal tests access unexported identifiers and legitimately create dynamic errors, capture contexts, and exceed function-length thresholds.
- **mnd** ignores `family.go` — the `familyData` table contains intentional HTTP status codes, exit codes, and severity values with inline comments. Extracting 15+ named constants would reduce readability.
- **varnamelen** ignore-names includes Go-idiomatic short names: `tc` (test case), `f` (fmt.State — Go stdlib convention), `w` (http.ResponseWriter), `ag` (agent).

## Known Limitations

- **`applyContext` uses `{key}` syntax (handle.go):** Template values are substituted via `strings.ReplaceAll` without HTML escaping. This is intentional for CLI output (stderr) but would be unsafe for HTML rendering. Consumers building HTTP responses should escape values before embedding in HTML.
- **`agent.Config.Enabled` is now honest:** A disabled agent returns `(nil, error)` instead of a synthetic `AgentResult`. Calling `Analyze` on a disabled agent is a programming error, not a silent result.
- **`ClassifiedError` value-embeds `oops.OopsError`:** The zero value has nil internals. Methods like `Error()` and `Is()` guard against this, but future methods added to `ClassifiedError` must handle the zero-OopsError case.
- **Examples are a separate module:** `examples/` has its own `go.mod` (requires root + diagnose). This keeps the root module truly zero-dependency — no `replace` directives, no phantom requires. CI builds it via `working-directory: ./examples`.
- **json/v2 reverted (2026-07-23):** The root module previously used `encoding/json/v2` (v0.7.0) but has been reverted to stdlib `encoding/json`. No `GOEXPERIMENT` required. The library is now a pure stdlib consumer with zero adoption friction.
- **Website (`website/`):** Astro 7 + Starlight + Tailwind v4 documentation site. Firebase Hosting target `errorfamily` in the `lars-software` project. Domain: `errorfamily.lars.software`. Deploy with `nix run .#deploy` from the `website/` directory (runs `npm run build && firebase deploy --only hosting`). The website is NOT part of the Go workspace or CI — it's a separate Node.js project with its own `flake.nix`.
