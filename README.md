# go-error-family

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-error-family.svg)](https://pkg.go.dev/github.com/larsartmann/go-error-family)
[![Go Report Card](https://goreportcard.com/badge/github.com/larsartmann/go-error-family)](https://goreportcard.com/report/github.com/larsartmann/go-error-family)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Structured error protocol for Go — behavioral classification, exit codes, and diagnostic rules.

**Share the protocol, not the implementation.**

---

## The Problem

Every Go program has `if err != nil`. But what do you _do_ with that error?

| Question                                | Before                                        | After                                       |
| --------------------------------------- | --------------------------------------------- | ------------------------------------------- |
| Should the caller retry?                | `if strings.Contains(err.Error(), "timeout")` | `errorfamily.IsRetryable(err)`              |
| What exit code?                         | `os.Exit(1)` for everything                   | `errorfamily.ExitCode(err)` — context-aware |
| What message for the user?              | `fmt.Println(err)` — often internal jargon    | Structured What/Why/Fix/WayOut              |
| Is it the user's fault or the system's? | Guess from the string                         | `errorfamily.Classify(err).Audience()`      |

go-error-family answers all of these with a single concept: **Family**.

## Installation

```bash
go get github.com/larsartmann/go-error-family
```

Requires Go 1.26+ (uses `errors.AsType`).

This is a Go workspace. The root module provides core types and classification. Diagnostic submodules (`diagnose/git`, `diagnose/postgres`) require separate imports.

## Quick Start

```go
package main

import (
    "errors"
    "os"

    "github.com/larsartmann/go-error-family"
)

func main() {
    err := errors.New("connection refused")

    family := errorfamily.Classify(err)
    // → Transient (default: unknown errors are retryable)

    if errorfamily.IsRetryable(err) {
        // yes — schedule a retry with backoff
    }

    os.Exit(errorfamily.ExitCode(err))
    // → 75 (EX_TEMPFAIL: temporary failure)
}
```

For your own errors, attach a Family at creation time:

```go
err := errorfamily.NewRejection("file.not_found", "config file missing").
    WithContext("path", "/etc/app/config.yaml")

os.Exit(errorfamily.HandleError(err))
// stderr: "A required resource was not found."
// stderr: "Check that the path and resource name are correct."
// exit: 1
```

See [examples/](examples/) for runnable CLI, HTTP, and custom diagnostic rule demos.

## What It Gives You

- **`Family`** — behavioral classification (Rejection, Conflict, Transient, Corruption, Infrastructure) that maps to retry decisions, exit codes, and user-facing tone
- **Small interfaces** — `Coded`, `Classified`, `Contextual`, `Retryable` — each error type implements what it needs
- **`Classify(err)`** — universal classification for any error (multi-error → interface → registered sentinels → default)
- **`ExitCode(err)`** — BSD sysexits.h exit codes derived from Family
- **`HandleError(err)`** — CLI boundary handler with structured messages (What / Why / Fix / WayOut)
- **`Compose(errs...)`** — combine errors via `errors.Join` for partial-success patterns
- **Diagnostic rules** — deterministic checks (PostgreSQL, filesystem, network, git) that auto-discover why an error occurred
- **AI debug agent** — root cause analysis and `FixStep` suggestions from diagnostic context

## The Five Families

| Family             | Retryable | Exit Code | Whose Fault             | Tone          |
| ------------------ | --------- | --------- | ----------------------- | ------------- |
| **Rejection**      | no        | 1         | User                    | Instructional |
| **Conflict**       | no        | 1         | User (needs to resolve) | Explanatory   |
| **Transient**      | **yes**   | 75        | System                  | Reassuring    |
| **Corruption**     | no        | 65        | Data damage             | Urgent        |
| **Infrastructure** | no        | 69        | System                  | Apologetic    |

Each family exposes `Audience()` (User / Ops / All) and `Tone()` for presentation-layer decisions.

## Constructors

```go
// Simple
err := errorfamily.NewRejection("config.invalid", "bad config")

// With cause
err := errorfamily.WrapTransient(dbErr, "db.timeout", "query timed out")

// With context (chainable)
err := errorfamily.NewRejection("file.not_found", "config missing").
    WithContext("path", "/etc/app/config.yaml").
    WithContext("format", "yaml")

// Formatted
err := errorfamily.Newf(errorfamily.Rejection, "file.not_found", "missing: %s", path)

// Multi-error (partial success)
err := errorfamily.Compose(err1, err2, err3)
```

Family-specific constructors: `NewRejection`, `NewConflict`, `NewTransient`, `NewCorruption`, `NewInfrastructure`. Wrap variants: `WrapRejection`, `WrapConflict`, `WrapTransient`, `WrapCorruption`, `WrapInfrastructure`.

## Classification

```go
// Classify any error → Family
family := errorfamily.Classify(err)

// Retry decision
if errorfamily.IsRetryable(err) {
    retry(err)
}

// Exit code for CLI
os.Exit(errorfamily.ExitCode(err))

// Parse from string (e.g. config, HTTP headers)
family := errorfamily.ParseFamily("transient") // defaults to Transient for unknowns
```

Classification precedence — first match wins:

1. **Multi-error** (`errors.Join`) → classify each sub-error, first non-Transient wins
2. `Classified` interface → `ErrorFamily()`
3. `Retryable` interface → infer Transient (true) or Rejection (false)
4. Registered sentinels via `errors.Is` chain walk
5. Default → Transient (fail-open for retry)

## Registering Third-Party Errors

For errors you don't own (stdlib, libraries):

```go
func init() {
    errorfamily.RegisterClassification(sql.ErrConnDone, errorfamily.Transient)
    errorfamily.RegisterClassification(os.ErrPermission, errorfamily.Rejection)

    // Batch registration
    errorfamily.RegisterClassifications(map[error]errorfamily.Family{
        sql.ErrConnDone: errorfamily.Transient,
        sql.ErrTxDone:   errorfamily.Transient,
    })
}
```

## Implementing Your Own Error Type

You don't have to use the built-in `Error` struct. Implement the interfaces you need:

```go
package mypkg

import (
    "fmt"

    "github.com/larsartmann/go-error-family"
)

type FindingError struct {
    Category string
    Message  string
    File     string
    Line     int
    Cause    error
}

func (e *FindingError) Error() string          { return e.Message }
func (e *FindingError) Unwrap() error          { return e.Cause }
func (e *FindingError) ErrorCode() string      { return e.Category }
func (e *FindingError) ErrorFamily() errorfamily.Family {
    switch e.Category {
    case "validation":
        return errorfamily.Rejection
    case "io":
        return errorfamily.Transient
    default:
        return errorfamily.Infrastructure
    }
}
func (e *FindingError) ErrorContext() map[string]string {
    return map[string]string{"file": e.File, "line": fmt.Sprintf("%d", e.Line)}
}
```

Now `Classify()`, `IsRetryable()`, `ExitCode()`, and `HandleError()` all work with your type.

## CLI Boundary: HandleError

`HandleError` is the top-of-main handler that classifies the error, picks an exit code, formats a user-facing message, and writes to stderr:

```go
func main() {
    if err := run(); err != nil {
        os.Exit(errorfamily.HandleError(err))
    }
}
```

For more control:

```go
// Structured result (no stderr write) — for HTTP handlers, gRPC interceptors
result := errorfamily.HandleErrorDetailed(err)
fmt.Printf("exit=%d msg=%q fix=%q\n", result.ExitCode, result.Message, result.SuggestedFix)

// Template-aware structured result
result := errorfamily.HandleErrorDetailedWithConfig(err, cfg)

// Context-propagating handler — preferred when you have a context.Context
exitCode := errorfamily.HandleErrorWithContext(ctx, err, errorfamily.HandleConfig{
    Diagnose:    true,
    Output:      myWriter,
    DiagnosticFunc: myDiagnoseFunc,
    OnDiagnosed: func(err error, findings []errorfamily.DiagnosticFinding) {
        logFindings(findings)
    },
    TemplateOverride: map[string]errorfamily.MessageTemplate{
        "file.not_found": {
            What:   "Could not find {{.path}}",
            Why:    "The file doesn't exist at the expected location.",
            Fix:    "Check that {{.path}} exists and is readable.",
            WayOut: "Run with --verbose for more details.",
        },
    },
})
```

### Template Resolution

Messages are resolved in this order (first match wins):

1. `HandleConfig.TemplateOverride[code]` — per-call consumer override
2. `RegisterTemplate(code, tmpl)` — global registry
3. Built-in `defaultMessages[code]` — exact-match defaults
4. `Family.DefaultMessage()` — generic family-based fallback

All lookups are exact code matches (case-insensitive). No substring matching.

Register templates globally for reusable error codes:

```go
errorfamily.RegisterTemplate("db.timeout", errorfamily.MessageTemplate{
    What: "Database connection timed out.",
    Why:  "The database server did not respond within the deadline.",
    Fix:  "Check database health and network connectivity.",
})
```

## Diagnostic Rules

Auto-discover why an error occurred:

```go
runner := diagnose.DefaultRunner()
results := runner.Run(ctx, err)

for _, r := range results {
    if r.Status == diagnose.StatusFailed {
        fmt.Println(r.Summary)
        fmt.Println("  Fix:", r.SuggestedFix)
    }
}
```

Built-in rules (zero-dependency, included in `DefaultRunner`):

- **FilesystemRule** — checks path existence, permissions, writability
- **NetworkRule** — checks DNS resolution, TCP connectivity

Opt-in submodules (import explicitly):

```go
import (
    "github.com/larsartmann/go-error-family/diagnose/git"
    "github.com/larsartmann/go-error-family/diagnose/postgres"
)

runner := diagnose.NewRunner(&git.GitRule{}, &postgres.PostgresRule{}, &diagnose.FilesystemRule{})
```

- **GitRule** (`diagnose/git`) — checks repo state, merge conflicts, remote reachability
- **PostgresRule** (`diagnose/postgres`) — checks `pg_isready`, TCP connectivity

Results include `Confidence` (0.0–1.0) and are sorted by confidence descending.

### Writing Custom Rules

Use `RuleSpec` for data-driven matching — the simplest and most common pattern:

```go
type RateLimitRule struct{}

const keyRetryAfter diagnose.ContextKey = "retry_after"

var rateLimitSpec = diagnose.RuleSpec{
    ContextKeys:  []diagnose.ContextKey{keyRetryAfter},
    CodeContains: []string{"rate.limit", "too_many_requests"},
}

func (r *RateLimitRule) Name() string              { return "rate_limit" }
func (r *RateLimitRule) Applicable(err error) bool { return rateLimitSpec.Matches(err) }

func (r *RateLimitRule) Run(ctx context.Context, err error) (*diagnose.DiagnosticResult, error) {
    retryAfter := diagnose.ResolveContextKey(err, []string{"retry_after"}, "unknown")
    // ... check system state ...
    return &diagnose.DiagnosticResult{
        Status:       diagnose.StatusDegraded,
        Confidence:   diagnose.ConfidenceHigh,
        Summary:      "Rate limited — retry after " + retryAfter,
        SuggestedFix: "Wait for the duration specified in the Retry-After header",
    }, nil
}
```

For testable rules that shell out, accept a `CommandRunner`:

```go
type MyRule struct {
    Runner diagnose.CommandRunner // inject mock in tests; defaults to DefaultCommandRunner{}
}
```

See the [custom rule example](examples/README.md) for a complete working example.

## AI Debug Agent

Root cause analysis and fix suggestions from diagnostic context:

```go
ag := agent.New(agent.Config{Enabled: true})
result, _ := ag.Analyze(ctx, err, diagnosis)

fmt.Println("Root cause:", result.RootCause)
fmt.Println("Confidence:", result.Confidence)
for _, step := range result.FixSteps {
    fmt.Printf("  - %s\n    Command: %s\n", step.Description, step.Command)
}
```

The agent produces analysis but does **not** execute fixes — the consumer decides what to do with `FixStep.Command`.

`AgentResult` fields: `RootCause`, `Confidence`, `Explanation`, `FixSteps`.

## Performance

Zero-allocation hot paths. Benchmarks on AMD Ryzen 9 7950X:

| Operation                    | Time    | Allocs |
| ---------------------------- | ------- | ------ |
| `Classify` (built-in Error)  | ~9 ns   | 0      |
| `Classify` (plain error)     | ~30 ns  | 0      |
| `IsRetryable`                | ~9 ns   | 0      |
| `ExitCode`                   | ~9 ns   | 0      |
| `WithContext`                | ~8 ns   | 0      |
| `ParseFamily`                | ~12 ns  | 0      |
| `HandleError` (with context) | ~450 ns | 5      |
| `Runner.Run` (1 rule)        | ~420 ns | 5      |

`HandleError` includes template resolution and stderr write. Use `HandleErrorDetailed` when you only need the structured result.

## Architecture

```
go-error-family/
├── family.go               — Family enum + data-driven familyData
├── interfaces.go           — Coded, Classified, Contextual, Retryable (each embeds error)
├── error.go                — Reference Error struct (Is, Unwrap, Format, WithContext, accessors)
├── classify.go             — Classify, IsRetryable, ExitCode, RegisterClassification(s)
├── constructors.go         — New, Wrap, Newf, Wrapf + family-specific shortcuts
├── handle.go               — HandleError, HandleErrorWithContext, template system
├── diagnose/
│   ├── diagnose.go         — Runner, DiagnosticRule, RuleSpec, CommandRunner, ContextKey
│   ├── context.go          — RunCommand, CommandExists (exported for rule authors)
│   ├── rules_filesystem.go — FilesystemRule
│   ├── rules_network.go    — NetworkRule
│   ├── git/                — submodule: GitRule
│   └── postgres/           — submodule: PostgresRule
├── agent/
│   └── agent.go            — DebugAgent interface, Config, AgentResult, FixStep
└── examples/
    ├── cmd/cli             — CLI boundary handler example
    ├── cmd/http            — HTTP middleware with status code mapping
    └── cmd/custom_rule     — Writing your own DiagnosticRule
```

## When to Use / When Not To

**Use this when** you need behavior derived from errors: retry decisions, exit codes, user-facing messages, or diagnostic investigation — especially at program boundaries (CLI top-level, HTTP handlers, gRPC interceptors).

**Don't use this when** a simple `fmt.Errorf("...: %w", err)` is enough. If you only need to wrap and propagate, the standard library is fine. This library shines when errors need to _drive behavior_.

## Philosophy

1. **Protocol, not framework** — share the vocabulary, not the implementation
2. **Each error type implements what it needs** — no god struct
3. **Presentation is separate from the error** — the CLI layer formats for humans
4. **Family = exit code = retry decision = tone** — one concept, many audiences
5. **Generic errors are structurally impossible** — Code + Family + Context guarantee specificity

## License

[MIT](LICENSE) — Copyright (c) 2026 Lars Artmann
