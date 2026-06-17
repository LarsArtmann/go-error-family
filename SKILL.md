# go-error-family — AI Agent Context

**Module:** `github.com/larsartmann/go-error-family`
**Go:** 1.26+ | **External deps:** zero | **Kind:** library (no `main`, no build system)

---

## What This Library Is

A structured error protocol for Go. Every error gets a behavioral **Family** (retry/no-retry), a machine-readable **code**, human-readable context, and optional diagnostics/agent analysis. Designed for the CLI/HTTP boundary — the place where errors leave your program and meet a human or a downstream system.

## Architecture at a Glance

```
errorfamily/          ← root package: types, constructors, classification, CLI boundary
  error.go              Error struct (reference implementation)
  family.go             Family enum + Audience/Tone metadata
  interfaces.go         Coded, Classified, Contextual, Retryable
  constructors.go       New/Wrap + family shortcuts
  classify.go           Classify(), IsRetryable, ExitCode, RegisterClassification(s)
  registry.go           Registry type (injectable sentinels + templates), DefaultRegistry, NewRegistry()
  handle.go             HandleError(), HandleErrorWithContext(), HandleErrorDetailed(), template system

diagnose/             ← concurrent diagnostic rules (zero-dep core)
  diagnose.go           Runner, DiagnosticRule, RuleSpec, CommandRunner, ContextKey, ErrorContext
  command.go           RunCommand, CommandExists, DefaultCommandRunner
  rules_filesystem.go   FilesystemRule
  rules_network.go      NetworkRule

diagnose/git/         ← submodule: GitRule (opt-in)
  rules_git.go          GitRule

diagnose/postgres/    ← submodule: PostgresRule (opt-in)
  rules_postgres.go     PostgresRule, IsPostgresRunning

agent/                ← analysis-only debug agent
  agent.go              DebugAgent interface, deterministic analyzer

bridge/               ← submodule: samber/oops integration (opt-in, depends on both libraries)
  bridge.go             ClassifiedError (satisfies Classified, Coded, Retryable, Contextual), Wrap
  classify.go           InferFamily, AutoWrap (tag/domain → Family mapping)
```

---

## The Five Families

| Family           | Retry?  | Exit | Whose fault | Audience | Tone          | When                                             |
| ---------------- | ------- | ---- | ----------- | -------- | ------------- | ------------------------------------------------ |
| `Rejection`      | No      | 1    | User        | User     | Instructional | Bad input, unauthorized, not found               |
| `Conflict`       | No      | 1    | User        | User     | Explanatory   | Version mismatch, duplicate, state clash         |
| `Transient`      | **Yes** | 75   | System      | All      | Reassuring    | Temporary infra failure (the only retryable one) |
| `Corruption`     | No      | 65   | System      | Ops      | Urgent        | Source of truth damaged, unparseable data        |
| `Infrastructure` | No      | 69   | System      | Ops      | Apologetic    | System cannot serve, nil deps, startup fail      |

Only `Transient` is retryable. Everything else is not. This is the core design decision.

### Family Methods

```go
family.IsRetryable() bool      // true only for Transient
family.ExitCode() int          // BSD sysexits.h code (see table above)
family.IsValid() bool          // true if within defined range
family.String() string         // "rejection", "transient", etc.
family.Tone() Tone             // presentation tone hint
family.Audience() Audience     // who to notify: User, Ops, or All
family.DefaultMessage() string // generic human-readable message
family.DefaultWhy() string     // generic "why" explanation
family.DefaultFix() string     // generic fix suggestion
```

### Audience & Tone Types

```go
type Audience int // AudienceUser, AudienceOps, AudienceAll
type Tone string  // "instructional", "explanatory", "reassuring", "urgent", "apologetic"
```

Audience mapping: Rejection/Conflict → User, Corruption/Infrastructure → Ops, Transient → All.

---

## Consumer Interfaces

All embed `error` (required for Go 1.26 `errors.AsType[T]()` — do not remove):

```go
type Coded interface {       // machine-readable identity (e.g. "db.timeout")
    error
    ErrorCode() string
}

type Classified interface {   // behavioral classification
    error
    ErrorFamily() Family
}

type Contextual interface {   // factual key-value details
    error
    ErrorContext() map[string]string
}

type Retryable interface {    // explicit retry hint (overrides Family)
    error
    IsRetryable() bool
}
```

`*Error` implements all four. Third-party error types can implement whichever subset makes sense.

---

## Quick API Reference

### Error Struct Methods

```go
// Accessors (beyond the interface methods)
err.ErrorCode() string                  // from Coded
err.ErrorFamily() Family                // from Classified
err.ErrorContext() map[string]string     // from Contextual (returns a copy)
err.IsRetryable() bool                  // from Retryable

// Direct accessors (no interface assertion needed)
err.Code() string                       // same as ErrorCode()
err.Family() Family                     // same as ErrorFamily()
err.Message() string                    // human-readable technical message
err.Cause() error                       // underlying error in the chain
err.Timestamp() time.Time               // when the error was created

// Mutators (chainable)
err.WithContext(key, value string) *Error
err.WithCause(cause error) *Error
err.WithTimestamp(ts time.Time) *Error   // deterministic timestamp for tests

// Helpers
err.HasContext(key string) bool
err.ContextValue(key string) string
err.Summary() string                    // "code: message" (no family prefix)

// Formatting (fmt.Formatter)
fmt.Sprintf("%v", err)    // [family:code] message[: cause]
fmt.Sprintf("%+v", err)   // verbose: context, timestamp, cause chain
fmt.Sprintf("%s", err)    // message only
```

### Creating Errors

```go
// Direct constructors
err := errorfamily.NewTransient("db.timeout", "query took too long")
err := errorfamily.WrapRejection(originalErr, "validation", "invalid email format")

// With context (chainable)
err := errorfamily.NewTransient("db.connection", "could not connect").
    WithContext("host", "localhost").
    WithContext("port", "5432")

// Generic constructors
errorfamily.New(family, code, message) *Error
errorfamily.Newf(family, code, format, args...) *Error
errorfamily.Wrap(err, family, code, message) *Error    // nil-safe: returns nil if err is nil
errorfamily.Wrapf(err, family, code, format, args...) *Error

// Family shortcuts (New + Wrap for each)
NewRejection / NewConflict / NewTransient / NewCorruption / NewInfrastructure
WrapRejection / WrapConflict / WrapTransient / WrapCorruption / WrapInfrastructure
```

### Classification

```go
family := errorfamily.Classify(err)     // always returns a Family (never panics)
retryable := errorfamily.IsRetryable(err)
exitCode := errorfamily.ExitCode(err)

family := errorfamily.ParseFamily("transient")  // parse from string (case-insensitive, defaults to Transient if unrecognized)

// Register third-party sentinels (call from init())
errorfamily.RegisterClassification(sql.ErrConnDone, errorfamily.Transient)
errorfamily.RegisterClassifications(map[error]errorfamily.Family{...})

// Combine errors for partial-success patterns — use stdlib errors.Join
combined := errors.Join(err1, err2)  // Classify picks the worst Family automatically
```

**Classification precedence** (first match wins):

1. **Multi-error** (`errors.Join`) → classify each sub-error, first non-Transient wins
2. `Classified` interface → `ErrorFamily()`
3. `Retryable` interface → infer `Transient` (true) or `Rejection` (false)
4. Registered sentinels via `errors.Is` chain walk (RLock-protected iteration)
5. Default → `Transient` (fail-open)

### Injectable Registry (test isolation, scoped handling)

Package-level functions (`Classify`, `RegisterClassification`, `RegisterTemplate`) delegate to `DefaultRegistry`. For test isolation or scoped error handling within a binary, construct a custom registry:

```go
reg := errorfamily.NewRegistry()
reg.RegisterClassification(sql.ErrConnDone, errorfamily.Transient)
reg.RegisterTemplate("custom.code", errorfamily.MessageTemplate{What: "Custom"})

// Pass via HandleConfig.Registry
code := errorfamily.HandleErrorWithConfig(err, errorfamily.HandleConfig{
    Registry: reg,
})

// Or classify directly
family := reg.Classify(err)
```

No `t.Cleanup(Unregister...)` needed — the registry is local, no global state mutated.

### CLI Boundary (main.go pattern)

```go
func main() {
    if err := run(); err != nil {
        os.Exit(errorfamily.HandleError(err))  // classify → format → stderr → exit code
    }
}
```

**Canonical entry point when you have a `context.Context`:**

```go
exitCode := errorfamily.HandleErrorWithContext(ctx, err, errorfamily.HandleConfig{})
```

`HandleError` and `HandleErrorWithConfig` both delegate to `HandleErrorWithContext`.

### Structured Result (HTTP/gRPC)

```go
result := errorfamily.HandleErrorDetailed(err)
// result.ExitCode, result.Message, result.SuggestedFix

// Template-aware structured result
result := errorfamily.HandleErrorDetailedWithConfig(err, cfg)
```

### Configurable Handler

```go
exitCode := errorfamily.HandleErrorWithConfig(err, errorfamily.HandleConfig{
    Output: os.Stderr,
    DiagnosticFunc: func(ctx context.Context, err error) []errorfamily.DiagnosticFinding {
        return ... // adapt diagnose.Runner results
    },
    TemplateOverride: map[string]errorfamily.MessageTemplate{
        "db.timeout": {What: "DB timed out on {host}", Fix: "Check {host}"},
    },
    OnDiagnosed: func(err error, findings []errorfamily.DiagnosticFinding) { ... },
})
```

### Diagnostics

```go
// One-shot with zero-dep built-in rules (Filesystem, Network)
results := diagnose.RunAuto(ctx, err)

// Custom runner with opt-in submodules
import (
    "github.com/larsartmann/go-error-family/diagnose/git"
    "github.com/larsartmann/go-error-family/diagnose/postgres"
)
runner := diagnose.NewRunner(&git.GitRule{}, &postgres.PostgresRule{}, &myCustomRule{})
results := runner.Run(ctx, err)
// results sorted by confidence desc; nil if no rules applicable
```

**DiagnosticResult fields:** `RuleName`, `Status` (Healthy/Degraded/Failed/Unknown), `Summary`, `Details` (map[string]string), `SuggestedFix`, `Confidence` (0.0–1.0), `Duration`, `Context` (the error context that triggered the rule, `map[string]string`).

**Standalone helpers (postgres submodule):**

```go
postgres.IsPostgresRunning(ctx, host, port) bool  // pg_isready or TCP check
```

### Partial Success (Recipe)

This library does not provide batch/multi-error types — partial success is a **consumption pattern**, not a classification concern. Use the existing primitives to compose what your domain needs:

```go
// 1. Process items, collect per-item results.
type outcome struct {
    value Item
    err   error
}
var results []outcome
for _, item := range items {
    v, err := process(item)
    results = append(results, outcome{value: v, err: err})
}

// 2. Separate successes from failures.
var successes []Item
var failures []outcome
for _, r := range results {
    if r.err == nil {
        successes = append(successes, r.value)
    } else {
        failures = append(failures, r)
    }
}

// 3. Use Classify to decide what to do with each failure.
for _, f := range failures {
    switch errorfamily.Classify(f.err) {
    case errorfamily.Transient:
        // retry (backoff, jitter, idempotency are the consumer's responsibility)
    case errorfamily.Rejection:
        // skip, log, or surface to user
    case errorfamily.Corruption:
        // escalate to ops
    }
}

// 4. If you need a single exit code, pick the one with the highest ExitCode().
// Exit codes map: Transient(75) > Infrastructure(69) > Corruption(65) > Conflict(1) = Rejection(1).
// Note: exit codes ≠ severity. Transient is retryable (not "worst"); Corruption is severe but low exit code.
worst := errorfamily.Transient
for _, f := range failures {
    if errorfamily.Classify(f.err).ExitCode() > worst.ExitCode() {
        worst = errorfamily.Classify(f.err)
    }
}
```

Why not a built-in `ErrorBatch` type? Because batch semantics vary by domain — some consumers want fail-fast, some want collect-all, some want per-item retry with circuit breakers. The library provides the **classification vocabulary**; you compose the **collection strategy**.

### Agent (Analysis-Only)

```go
ag := agent.New(agent.Config{Enabled: true})
result, err := ag.Analyze(ctx, err, diagnosis)
// result.RootCause, result.Confidence, result.Explanation, result.FixSteps
// FixSteps have Description, Command, Rationale — consumer decides whether to execute
```

---

## Surprising Behaviors (Gotchas)

| Behavior                                           | Why                                                              |
| -------------------------------------------------- | ---------------------------------------------------------------- |
| `Classify(nil)` returns `Rejection`                | nil error = caller's fault                                       |
| `Classify` defaults unknown → `Transient`          | Fail-open: unknown errors get retried                            |
| `ParseFamily("unknown")` → `Transient`             | Same fail-open design                                            |
| `errors.Is` matches on **code + family** only      | Two `*Error`s with different messages but same code+family match |
| `Wrap(nil, ...)` returns `nil`                     | Nil-safe, but can't construct error wrapping nil                 |
| `Error.ErrorContext()` returns a **copy**          | Mutations won't affect the original                              |
| Template `{key}` uses `strings.ReplaceAll`         | Not html/template — just simple substitution                     |
| `DiagnosticFunc` is a function type, not interface | Avoids circular import between root and diagnose packages        |

---

## Template Resolution Order

1. `HandleConfig.TemplateOverride[code]` — per-call override
2. `Registry.lookupTemplate(code)` — registry templates via `RegisterTemplate()` (uses `DefaultRegistry` unless `HandleConfig.Registry` is set)
3. `defaultMessages[code]` — built-in exact-match (see handle.go)
4. `family.DefaultMessage()` — generic fallback

`SuggestedFix` resolution follows the same chain via `resolveSuggestedFix()`.

All lookups are exact code match (case-insensitive). No substring matching.

Built-in codes: `file.not_found`, `permission.denied`, `db.timeout`, `db.connection`, `db.error`, `config.invalid`, `config.not_found`, `conflict`, `validation`, `timeout`, `connection.refused`, `git.error`.

---

## Diagnostic Rule Pattern

### Adding a New Rule

1. Implement `diagnose.DiagnosticRule` (3 methods: `Name`, `Applicable`, `Run`)
2. Use `diagnose.RuleSpec` for matching — define it as a package-level `var`:

```go
var mySpec = diagnose.RuleSpec{
    ContextKeys:   []diagnose.ContextKey{diagnose.KeyHost}, // typed string constants
    CodeContains:  []string{"my."},                          // matches if error code contains substring
    ContextSubstr: []string{"my_thing"},                     // matches if any context value contains substring
    Extra:         func(err error) bool { ... },             // custom logic
}

func (r *MyRule) Applicable(err error) bool { return mySpec.Matches(err) }
```

3. Use matching helpers from `diagnose` package: `HasContextKey`, `ContextValue`, `ResolveContextKey`, `HasContextSubstring`, `FamilyIs`, `ErrorCodeContains`
4. Use execution helpers: `diagnose.RunCommand`, `diagnose.CommandExists`
5. Extract context from any error: `diagnose.ErrorContext(err)` → `map[string]string`
6. Rules run concurrently via `Runner.Run`; results sorted by confidence descending

### Testability: CommandRunner

Rules that shell out should accept a `CommandRunner` for mock injection:

```go
type MyRule struct {
    Runner diagnose.CommandRunner // nil → DefaultCommandRunner{}
}
```

### Built-in Rules

| Rule             | Module                                                     | Matches On                                                                                                                                                                | Checks                        |
| ---------------- | ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------- |
| `FilesystemRule` | `github.com/larsartmann/go-error-family/diagnose`          | Keys: `path`, `file`, `dir`, `directory`, `config_path`, `output_path`. Codes: `file`, `dir`, `path`, `config`, `permission`                                              | Existence, permissions, write |
| `NetworkRule`    | `github.com/larsartmann/go-error-family/diagnose`          | Keys: `host`, `port`, `url`, `endpoint`, `address`, `remote`. Codes: `network`, `connect`, `dial`, `timeout`. Substr: `connection refused`, `no such host`, `i/o timeout` | DNS, TCP, port reachability   |
| `GitRule`        | `github.com/larsartmann/go-error-family/diagnose/git`      | Keys: `git`, `repository`, `repo`, `branch`, `git_dir`. Codes: `git`. Substr: `git`                                                                                       | Repo, tree, merge, remote     |
| `PostgresRule`   | `github.com/larsartmann/go-error-family/diagnose/postgres` | Keys: `db_host`, `db_port`, `db_name`, `database_url`, `postgres_host`. Codes: `db.`, `database`. Substr: `postgres`, `postgresql`, `database`, `sql` + Transient family  | pg_isready, TCP, start cmd    |

---

## Bridge Submodule (samber/oops integration)

`bridge/` is a separate Go module that connects go-error-family with samber/oops. It has its own `go.mod` with both libraries as dependencies. The root package remains zero-dependency.

### ClassifiedError

```go
// Wraps any error with a behavioral Family + oops context
type ClassifiedError struct {
    oops.OopsError           // preserves all oops methods (Stacktrace, Sources, etc.)
    // satisfies: Classified, Coded, Retryable, Contextual, fmt.Formatter
}

// Manual family assignment
classified := bridge.Wrap(err, errorfamily.Transient)

// Automatic inference from oops metadata
classified := bridge.AutoWrap(err)          // tags -> domain -> Transient (fail-open)
family := bridge.InferFamily(err)           // just the Family, no wrapping
```

### InferFamily cascade

1. **Tags** (developer-intentional) — `retryable`, `transient`, `conflict`, `corruption`/`corrupted`, `rejection`/`rejected`, `infrastructure`/`infra`
2. **Domain** (structural) — `validation`/`auth` -> Rejection, `database`/`network`/`cache`/`queue` -> Transient, `storage`/`infra`/`startup` -> Infrastructure, `data`/`schema`/`migration` -> Corruption
3. **Default** — `Transient` (fail-open, consistent with root Classify)

### What ClassifiedError bridges

| oops method  | error-family interface                          | Notes                                      |
| ------------ | ----------------------------------------------- | ------------------------------------------ |
| `.Code()`    | `ErrorCode() string` (Coded)                    | Converts `any` to string via fmt.Sprint    |
| `.Context()` | `ErrorContext() map[string]string` (Contextual) | Non-strings converted via fmt.Sprint       |
| `.Domain()`  | Included in `ErrorContext()["domain"]`          |                                            |
| `.Tags()`    | Included in `ErrorContext()["tags"]`            |                                            |
| —            | `ErrorFamily() Family` (Classified)             | From the attached Family                   |
| —            | `IsRetryable() bool` (Retryable)                | Derived from Family                        |
| `.Is()`      | `Is(target error) bool`                         | Delegates to OopsError.Is + original error |
| `Format()`   | `fmt.Formatter`                                 | `%+v` shows oops stacktrace when present   |

### Original error preservation

`Wrap(err, family)` always preserves the original error in the `Unwrap()` chain, even when `err` is not an OopsError. `errors.Is(classified, originalErr)` always works.

### Import

```go
import "github.com/larsartmann/go-error-family/bridge"
```

---

## Testing

```bash
go test ./...                                    # all tests
go test -cover ./...                             # with coverage
go test -coverprofile=cover.out ./... && go tool cover -func=cover.out  # detailed coverage
go test -run TestName ./...                      # specific test
go test -bench=. -run=^$ ./...                    # benchmarks only
```

Test files and scope:

- `errorfamily_test.go` — Family, ParseFamily, Error, constructors, Classify, RegisterClassification, errors.Is/As integration
- `benchmark_test.go` — Performance baselines for Classify, HandleError, ExitCode, ParseFamily, etc.
- `handle_test.go` — HandleError, HandleErrorWithConfig, diagnostics wiring, template overrides
- `handle_context_test.go` — HandleErrorWithContext, HandleErrorDetailed, context propagation, template overrides
- `template_test.go` — MessageTemplate rendering, RegisterTemplate, case-insensitive matching
- `fuzz_test.go` — FuzzParseFamily, FuzzClassify, FuzzErrorFormatting
- `example_test.go` — ExampleNewTransient, ExampleClassify, ExampleWrapRejection, ExampleParseFamily
- `diagnose/helpers_test.go` — HasContextKey, ContextValue, HasContextSubstring, FamilyIs, ErrorCodeContains, RuleSpec
- `diagnose/rules_test.go` — FilesystemRule and NetworkRule Applicable/Run
- `diagnose/runner_test.go` — Runner, context cancellation, confidence sorting, error handling
- `diagnose/benchmark_test.go` — Benchmarks for Runner.Run, RuleSpec.Matches, DefaultRunner
- `diagnose/git/scenario_test.go` — GitRule with MockCommandRunner (repo, working tree, remote)
- `diagnose/git/mock_test.go` — MockCommandRunner integration for git scenarios
- `diagnose/git/integration_test.go` — GitRule against real git repos
- `diagnose/postgres/rules_postgres_test.go` — PostgresRule Applicable, Run, resolveHost, IsPostgresRunning
- `agent/agent_test.go` — Analyze (enabled/disabled/with diagnosis/empty/timeout), extractCommand
- `bridge/wrap_test.go` — Wrap, family classification, Coded interface, errors.Is, fmt.Formatter, ErrorContext
- `bridge/autowrap_test.go` — AutoWrap, tag overrides, domain defaults, integration, benchmarks, examples
- `bridge/infer_test.go` — InferFamily, all tag overrides, all domain defaults, edge cases
- `bridge/fuzz_test.go` — FuzzInferFamily, FuzzAutoWrap, FuzzWrapRoundTrip, FuzzWrapOopsRoundTrip, FuzzFormat

**Coverage:** root 95.9% | agent 100% | diagnose core 67.1% | git 98.5% | postgres 81.0%
(rules that shell out to system commands are tested via `CommandRunner` mocks in git/postgres; diagnose core coverage reflects shell-out rules tested via integration)

### Test Style

Standard `testing.T` table-driven tests. No external test frameworks. Same-package tests (no `_test` suffix on package name — tests access internals).

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name string
        // ...
        want string
    }{
        {name: "basic", want: "expected"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := thing(); got != tt.want {
                t.Errorf("thing() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

---

## Code Conventions

- **No external dependencies** — stdlib only
- **Interfaces embed `error`** — for `errors.AsType[T]()` compatibility
- **Data-driven patterns** — `familyData` array, `defaultMessages` map, `ruleSpec` structs
- **Thread-safe registries** — `sync.RWMutex` for classification and template registries; RLock iteration for reads
- **Nil-safe** — `Wrap(nil, ...)` returns nil; `Classify(nil)` returns `Rejection`
- **`maps.Clone`** for defensive copies in `ErrorContext()`
- **Constructors set `timestamp: time.Now().UTC()`**
- **Context values are always `string`** — not `any`
- **Error codes use dot-notation** — e.g. `db.timeout`, `file.not_found`, `config.invalid`
- **No `main` package, no build system** — this is a library consumers import

---

## Key Files for Common Tasks

| Task                           | File(s)                                                                       |
| ------------------------------ | ----------------------------------------------------------------------------- |
| Add a new Family               | `family.go` (const + familyData entry)                                        |
| Add a new constructor shortcut | `constructors.go`                                                             |
| Change classification logic    | `classify.go`                                                                 |
| Add/modify message templates   | `handle.go` (defaultMessages) or use `RegisterTemplate()`                     |
| Add a diagnostic rule          | New file in `diagnose/`, implement `DiagnosticRule`, add to `DefaultRunner()` |
| Change CLI boundary behavior   | `handle.go`                                                                   |
| Modify agent analysis          | `agent/agent.go`                                                              |
| Understand the Error struct    | `error.go`                                                                    |
| Understand consumer interfaces | `interfaces.go`                                                               |

---

## MessageTemplate (Wix-style)

```go
type MessageTemplate struct {
    What   string  // "Could not connect to {host}"
    Why    string  // "The database is not reachable."
    Fix    string  // "Check that {host} is running."
    WayOut string  // "Run with --verbose for more details."
}
```

`{key}` placeholders are replaced from error context values. Empty fields fall back to family defaults.

---

## Dependency Graph

```
agent → errorfamily (root)
agent → diagnose

diagnose → errorfamily (root)

errorfamily → (stdlib only)
```

The root package has no dependency on `diagnose` or `agent`. `DiagnosticFunc` in `handle.go` is a function type to avoid circular imports — the consumer wires `diagnose.Runner` to it.
