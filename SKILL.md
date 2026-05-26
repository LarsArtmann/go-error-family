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
  classify.go           Classify(), RegisterClassification()
  handle.go             HandleError(), HandleErrorDetailed(), template system

diagnose/             ← concurrent diagnostic rules
  diagnose.go           Runner, DiagnosticRule interface, rule matching helpers
  context.go            runCommand, commandExists (shared OS helpers)
  rules_postgres.go     PostgresRule
  rules_filesystem.go   FilesystemRule
  rules_network.go      NetworkRule
  rules_git.go          GitRule

agent/                ← analysis-only debug agent
  agent.go              DebugAgent interface, deterministic analyzer
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
```

**Classification precedence** (first match wins):

1. `Classified` interface → `ErrorFamily()`
2. `Retryable` interface → infer `Transient` (true) or `Rejection` (false)
3. Registered sentinels via `errors.Is` chain walk (lock-free snapshot)
4. Default → `Transient` (fail-open)

### CLI Boundary (main.go pattern)

```go
func main() {
    if err := run(); err != nil {
        os.Exit(errorfamily.HandleError(err))  // classify → format → stderr → exit code
    }
}
```

### Structured Result (HTTP/gRPC)

```go
result := errorfamily.HandleErrorDetailed(err)
// result.ExitCode, result.Message, result.SuggestedFix
```

### Configurable Handler

```go
exitCode := errorfamily.HandleErrorWithConfig(err, errorfamily.HandleConfig{
    Output: os.Stderr,
    Diagnose: true,
    DiagnosticFunc: func(ctx context.Context, err error) []errorfamily.DiagnosticFinding {
        return ... // adapt diagnose.Runner results
    },
    TemplateOverride: map[string]errorfamily.MessageTemplate{
        "db.timeout": {What: "DB timed out on {{.host}}", Fix: "Check {{.host}}"},
    },
    OnDiagnosed: func(err error, findings []errorfamily.DiagnosticFinding) { ... },
})
```

### Diagnostics

```go
// One-shot with all built-in rules
results := diagnose.RunAuto(ctx, err)

// Custom runner
runner := diagnose.NewRunner(&diagnose.PostgresRule{}, &myCustomRule{})
results := runner.Run(ctx, err)
// results sorted by confidence desc; nil if no rules applicable
```

**DiagnosticResult fields:** `RuleName`, `Status` (Healthy/Degraded/Failed/Unknown), `Summary`, `Details` (map[string]string), `SuggestedFix`, `Confidence` (0.0–1.0), `Duration`.

**Standalone helpers:**

```go
diagnose.IsPostgresRunning(ctx, host, port) bool  // pg_isready or TCP check
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
        // retry with backoff
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
| Template `{{.key}}` uses `strings.ReplaceAll`      | Not html/template — just simple substitution                     |
| `DiagnosticFunc` is a function type, not interface | Avoids circular import between root and diagnose packages        |

---

## Template Resolution Order

1. `HandleConfig.TemplateOverride[code]` — per-call override
2. `lookupTemplate(code)` — global registry via `RegisterTemplate()`
3. `defaultMessages[code]` — built-in exact-match (see handle.go)
4. `family.DefaultMessage()` — generic fallback

All lookups are exact code match (case-insensitive). No substring matching.

Built-in codes: `file.not_found`, `permission.denied`, `db.timeout`, `db.connection`, `db.error`, `config.invalid`, `config.not_found`, `conflict`, `validation`, `timeout`, `connection.refused`, `git.error`.

---

## Diagnostic Rule Pattern

### Adding a New Rule

1. Implement `diagnose.DiagnosticRule` (3 methods: `Name`, `Applicable`, `Run`)
2. Use `ruleSpec` for matching — define it as a package-level `var`:

```go
var mySpec = ruleSpec{
    ContextKeys:   []string{"my_key"},          // matches if error context has any of these keys
    CodeContains:  []string{"my."},              // matches if error code contains substring
    ContextSubstr: []string{"my_thing"},         // matches if any context value contains substring
    Extra:         func(err error) bool { ... }, // custom logic
}

func (r *MyRule) Applicable(err error) bool { return mySpec.matches(err) }
```

3. Use matching helpers from `diagnose/diagnose.go` (NOT context.go): `hasContextKey`, `contextValue`, `resolveContextKey`, `hasContextSubstring`, `familyIs`, `errorCodeContains`
4. Rules run concurrently via `Runner.Run`; results sorted by confidence descending
5. Register in `DefaultRunner()` (diagnose.go) if it's a built-in rule

### Built-in Rules

| Rule             | Matches On                                                                                                                                                                | Checks                        |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------- |
| `PostgresRule`   | Keys: `db_host`, `db_port`, `db_name`, `database_url`, `postgres_host`. Codes: `db.`, `database`. Substr: `postgres`, `postgresql`, `database`, `sql` + Transient family  | pg_isready, TCP, start cmd    |
| `FilesystemRule` | Keys: `path`, `file`, `dir`, `directory`, `config_path`, `output_path`. Codes: `file`, `dir`, `path`, `config`, `permission`                                              | Existence, permissions, write |
| `NetworkRule`    | Keys: `host`, `port`, `url`, `endpoint`, `address`, `remote`. Codes: `network`, `connect`, `dial`, `timeout`. Substr: `connection refused`, `no such host`, `i/o timeout` | DNS, TCP, port reachability   |
| `GitRule`        | Keys: `git`, `repository`, `repo`, `branch`, `git_dir`. Codes: `git`. Substr: `git`                                                                                       | Repo, tree, merge, remote     |

---

## Testing

```bash
go test ./...                                    # all tests
go test -cover ./...                             # with coverage
go test -coverprofile=cover.out ./... && go tool cover -func=cover.out  # detailed coverage
go test -run TestName ./...                      # specific test
```

Test files and scope:

- `errorfamily_test.go` — Family, ParseFamily, Error, constructors, Classify, RegisterClassification, errors.Is/As integration
- `handle_test.go` — HandleError, HandleErrorWithConfig, HandleErrorDetailed, template overrides, diagnostics wiring
- `diagnose/diagnose_test.go` — Runner, rule matching helpers, Applicable, Run for local paths
- `agent/agent_test.go` — Analyze (enabled/disabled/with diagnosis/empty/timeout), extractCommand

**Coverage:** root 97.1% | agent 100% | diagnose 60.6% (rules that shell out are integration-test territory)

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
- **Thread-safe registries** — `sync.RWMutex` for classification and template registries; snapshots for reads
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
    What   string  // "Could not connect to {{.host}}"
    Why    string  // "The database is not reachable."
    Fix    string  // "Check that {{.host}} is running."
    WayOut string  // "Run with --verbose for more details."
}
```

`{{.key}}` placeholders are replaced from error context values. Empty fields fall back to family defaults.

---

## Dependency Graph

```
agent → errorfamily (root)
agent → diagnose

diagnose → errorfamily (root)

errorfamily → (stdlib only)
```

The root package has no dependency on `diagnose` or `agent`. `DiagnosticFunc` in `handle.go` is a function type to avoid circular imports — the consumer wires `diagnose.Runner` to it.
