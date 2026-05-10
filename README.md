# go-error-family

Structured error protocol for Go — behavioral classification, exit codes, and diagnostic rules.

**Share the protocol, not the implementation.**

## What It Gives You

- **`Family`** — behavioral classification (Rejection, Conflict, Transient, Corruption, Infrastructure) that maps to retry decisions, exit codes, and user-facing tone
- **Small interfaces** — `Coded`, `Classified`, `Contextual`, `Retryable` — each error type implements what it needs
- **`Classify(err)`** — universal classification for any error (interface → registered sentinels → default)
- **`ExitCode(err)`** — BSD sysexits.h exit codes from Family for shell scripts
- **`HandleError(err)`** — CLI boundary handler with Wix-quality messages
- **Diagnostic rules** — deterministic checks (PostgreSQL, filesystem, network, git) that auto-discover why an error occurred
- **AI debug agent** — configurable involvement levels (silent → autonomous)

## Quick Start

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/larsartmann/go-error-family"
)

func run() error {
    // Create classified errors with code + context
    return errorfamily.NewRejection("file.not_found", "config file missing").
        WithContext("path", "/etc/app/config.yaml")
}

func main() {
    if err := run(); err != nil {
        // HandleError: classifies → picks exit code → formats message → writes stderr
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

| Family | Retryable | Exit Code | Whose Fault | Tone |
|---|---|---|---|---|
| **Rejection** | no | 1 | User | Instructional |
| **Conflict** | no | 1 | User (needs to resolve) | Explanatory |
| **Transient** | **yes** | 75 | System | Reassuring |
| **Corruption** | no | 65 | Data damage | Urgent |
| **Infrastructure** | no | 69 | System | Apologetic |

## Constructors

```go
// Simple
err := errorfamily.NewRejection("config.invalid", "bad config")

// With cause
err := errorfamily.WrapTransient(dbErr, "db.timeout", "query timed out")

// With context
err := errorfamily.NewRejection("file.not_found", "config missing").
    WithContext("path", "/etc/app/config.yaml").
    WithContext("format", "yaml")

// Formatted
err := errorfamily.Newf(Rejection, "file.not_found", "missing: %s", path)
```

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
```

## Registering Third-Party Errors

For errors you don't own (stdlib, libraries):

```go
func init() {
    errorfamily.RegisterClassification(sql.ErrConnDone, errorfamily.Transient)
    errorfamily.RegisterClassification(os.ErrPermission, errorfamily.Rejection)
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

Now `errorfamily.Classify()`, `errorfamily.IsRetryable()`, `errorfamily.ExitCode()`, and `errorfamily.HandleError()` all work with your type.

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

Built-in rules:
- **PostgresRule** — checks `pg_isready`, TCP connectivity
- **FilesystemRule** — checks path existence, permissions, writability
- **NetworkRule** — checks DNS resolution, TCP connectivity
- **GitRule** — checks repo state, merge conflicts, remote reachability

## AI Debug Agent

Configurable AI-assisted debugging:

```go
cfg := agent.DefaultConfig()
cfg.Enabled = true
cfg.Involvement = agent.InvolvementSuggest // or Silent, Assist, Autonomous
cfg.ConfirmFunc = func(action string) bool {
    fmt.Println("Proposed:", action)
    return true // user approves
}

ag := agent.New(cfg)
result, _ := ag.Analyze(ctx, err, diagnosis)
for _, step := range result.FixSteps {
    fmt.Printf("  - %s (risk: %s)\n", step.Description, step.Risk)
}
```

Involvement levels:

| Level | Analyzes | Suggests | Applies Safe | Applies Risky |
|---|---|---|---|---|
| Silent | yes | no | no | no |
| Suggest | yes | yes | with approval | with approval |
| Assist | yes | yes | **auto** | with approval |
| Autonomous | yes | yes | **auto** | **auto** |

## Architecture

```
go-error-family/
├── family.go        — Family int enum, String, ExitCode, IsRetryable, Tone
├── interfaces.go    — Coded, Classified, Contextual, Retryable (embed error)
├── error.go         — Reference Error struct (Is, Unwrap, Format, Context)
├── classify.go      — Classify, IsRetryable, ExitCode, RegisterClassification
├── constructors.go  — New, Wrap, NewRejection, WrapTransient, etc.
├── handle.go        — HandleError (CLI boundary), HandleErrorDetailed
├── diagnose/
│   ├── diagnose.go  — Runner, DiagnosticRule, DiagnosticResult, helpers
│   ├── context.go   — SystemSnapshot, command runner, secret redaction
│   ├── rules_*.go   — PostgresRule, FilesystemRule, NetworkRule, GitRule
├── agent/
│   └── agent.go     — DebugAgent, Involvement levels, FixStep, Config
```

## Philosophy

Read the full design document: [`docs/2026-05-09_23-30_structured-errors-first-principles-design.md`](https://github.com/larsartmann/go-error-family/blob/main/docs/)

Key principles:
1. **Protocol, not framework** — share the vocabulary, not the implementation
2. **Each error type implements what it needs** — no god struct
3. **Presentation is separate from the error** — the CLI layer formats for humans
4. **Family = exit code = retry decision = tone** — one concept, many audiences
5. **Generic errors are structurally impossible** — Code + Family + Context guarantee specificity
