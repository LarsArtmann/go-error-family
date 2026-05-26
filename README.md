# go-error-family

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-error-family.svg)](https://pkg.go.dev/github.com/larsartmann/go-error-family)
[![Go Report Card](https://goreportcard.com/badge/github.com/larsartmann/go-error-family)](https://goreportcard.com/report/github.com/larsartmann/go-error-family)

Structured error protocol for Go — behavioral classification, exit codes, and diagnostic rules.

**Share the protocol, not the implementation.**

## Installation

```bash
go get github.com/larsartmann/go-error-family
```

Requires Go 1.26+ (uses `errors.AsType`).

## What It Gives You

- **`Family`** — behavioral classification (Rejection, Conflict, Transient, Corruption, Infrastructure) that maps to retry decisions, exit codes, and user-facing tone
- **Small interfaces** — `Coded`, `Classified`, `Contextual`, `Retryable` — each error type implements what it needs
- **`Classify(err)`** — universal classification for any error (interface → registered sentinels → default)
- **`ExitCode(err)`** — BSD sysexits.h exit codes derived from Family
- **`HandleError(err)`** — CLI boundary handler with structured messages (What / Why / Fix / WayOut)
- **Diagnostic rules** — deterministic checks (PostgreSQL, filesystem, network, git) that auto-discover why an error occurred
- **AI debug agent** — root cause analysis and `FixStep` suggestions from diagnostic context

## Quick Start

```go
package main

import (
    "os"

    "github.com/larsartmann/go-error-family"
)

func run() error {
    return errorfamily.NewRejection("file.not_found", "config file missing").
        WithContext("path", "/etc/app/config.yaml")
}

func main() {
    if err := run(); err != nil {
        os.Exit(errorfamily.HandleError(err))
    }
}
```

Output:

```
A required resource was not found.
Check that the path and resource name are correct.
```

Exit code: `1` (Rejection → user's fault)

## The Five Families

| Family             | Retryable | Exit Code | Whose Fault             | Tone          |
| ------------------ | --------- | --------- | ----------------------- | ------------- |
| **Rejection**      | no        | 1         | User                    | Instructional |
| **Conflict**       | no        | 1         | User (needs to resolve) | Explanatory   |
| **Transient**      | **yes**   | 75        | System                  | Reassuring    |
| **Corruption**     | no        | 65        | Data damage             | Urgent        |
| **Infrastructure** | no        | 69        | System                  | Apologetic    |

Each family also exposes `Audience()` (User / Ops / All) and `Tone()` for presentation-layer decisions.

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

1. `Classified` interface → `ErrorFamily()`
2. `Retryable` interface → infer Transient (true) or Rejection (false)
3. Registered sentinels via `errors.Is` chain walk
4. Default → Transient (fail-open for retry)

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

// Custom config — diagnostics, template overrides, custom output
exitCode := errorfamily.HandleErrorWithConfig(err, errorfamily.HandleConfig{
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

## Architecture

```
go-error-family/
├── family.go               — Family enum + data-driven familyData (Name, Exit, Tone, Message, Why, Fix)
├── interfaces.go           — Coded, Classified, Contextual, Retryable (each embeds error)
├── error.go                — Reference Error struct (Is, Unwrap, Format, WithContext, accessors)
├── classify.go             — Classify, IsRetryable, ExitCode, RegisterClassification(s)
├── constructors.go         — New, Wrap, Newf, Wrapf + family-specific shortcuts
├── handle.go               — HandleError, HandleErrorDetailed, template system, defaultMessages
├── diagnose/
│   ├── diagnose.go         — Runner, DiagnosticRule interface, RuleSpec (data-driven matching), helpers
│   ├── context.go          — RunCommand, CommandExists (exported for rule authors)
│   ├── rules_filesystem.go — FilesystemRule
│   ├── rules_network.go    — NetworkRule
│   ├── git/                — submodule: GitRule
│   └── postgres/           — submodule: PostgresRule
├── agent/
│   └── agent.go            — DebugAgent interface, Config, AgentResult, FixStep
```

## Philosophy

1. **Protocol, not framework** — share the vocabulary, not the implementation
2. **Each error type implements what it needs** — no god struct
3. **Presentation is separate from the error** — the CLI layer formats for humans
4. **Family = exit code = retry decision = tone** — one concept, many audiences
5. **Generic errors are structurally impossible** — Code + Family + Context guarantee specificity
