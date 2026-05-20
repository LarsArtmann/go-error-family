# Domain Language

A **Unified Language** for **go-error-family** — shared across Developer, Consumer, and AI.
Inspired by Domain-Driven Design (DDD) Ubiquitous Language.

Every term below should mean the **same thing** to everyone who reads it.

## Glossary

| Term | Definition | Context |
| --- | --- | --- |
| **Error** | A structured value representing something that went wrong, carrying code, message, family, context, cause, and timestamp | Core domain type |
| **Family** | A behavioral classification of an error that determines retry behavior, exit code, tone, and audience | One of: Rejection, Conflict, Transient, Corruption, Infrastructure |
| **Code** | A machine-readable dot-notation identifier for an error (e.g., `db.timeout`, `file.not_found`) | Used for metrics, log fields, template lookup |
| **Context** | Factual key-value pairs attached to an error (e.g., `host: "localhost"`, `port: "5432"`) | Always `string → string`, never `any` |
| **Classification** | The process of determining an error's Family from its type, interfaces, or registered sentinels | First-match-wins precedence: Classified → Retryable → Registered → Transient default |
| **Sentinel** | A known third-party error mapped to a Family via `RegisterClassification` | For errors you don't own (stdlib, libraries) |
| **Classification Precedence** | The ordered list of sources checked when classifying: Classified interface → Retryable interface → Registered sentinels → Transient default | First match wins |
| **Retryable** | Whether the operation that produced this error should be attempted again | Only `Transient` is retryable. All other families are not. |
| **CLI Boundary** | The point where errors leave the program and meet a human (main.go) or downstream system (HTTP/gRPC) | Where `HandleError` formats and outputs |
| **HandleError** | The CLI boundary handler — classifies, formats a Wix-style message, writes to stderr, returns exit code | Called exactly once at the top of `main()` |
| **MessageTemplate** | Wix-style presentation: What / Why / Fix / WayOut with `{{.key}}` placeholders | Resolved by error code, then family fallback |
| **DiagnosticRule** | A deterministic check that matches specific error patterns and investigates the system | Runs concurrently via `Runner`, sorted by confidence |
| **Confidence** | A 0.0–1.0 score indicating how likely a diagnostic result explains the error | Named constants: `ConfidenceNone` through `ConfidenceCertain` |
| **DiagnosticResult** | The outcome of a single diagnostic check: status, summary, details, suggested fix, confidence | Produced by `Runner.Run` |
| **Status** | The health assessment of a diagnostic check: Healthy, Degraded, Failed, or Unknown | `StatusHealthy` through `StatusUnknown` |
| **DebugAgent** | An analysis-only interface that examines errors with diagnostic context to produce root cause analysis and fix suggestions | Does NOT execute fixes — consumer decides |
| **FixStep** | A single actionable suggestion from the agent: description, shell command, rationale | Belongs to the consumer, not the library |
| **Tone** | A presentation hint for error messages: instructional, explanatory, reassuring, urgent, apologetic | Determined by Family |
| **Audience** | Who should be notified about an error: User, Ops, or All | Rejection/Conflict → User, Corruption/Infrastructure → Ops, Transient → All |
| **Exit Code** | BSD sysexits.h compatible process exit code for CLI boundary | Mapped from Family (e.g., Transient=75, Corruption=65) |
| **Fail-open** | The design decision that unknown errors default to Transient (retryable) | Unknown errors get retried rather than rejected |

## Value Objects

| Term | Definition | Context |
| --- | --- | --- |
| **Family** | `int` enum — Rejection=0, Conflict=1, Transient=2, Corruption=3, Infrastructure=4 | Core classification type |
| **Audience** | `int` enum — AudienceUser=0, AudienceOps=1, AudienceAll=2 | Notification target |
| **Tone** | `string` type — one of 5 defined constants | Presentation style |
| **Status** | `int` enum — StatusHealthy=0, StatusDegraded=1, StatusFailed=2, StatusUnknown=3 | Diagnostic outcome |
| **Confidence** | `float64` — 0.0 to 1.0 | Named constants: ConfidenceNone, ConfidenceNotCause, ConfidencePartial, ConfidenceLikely, ConfidenceHigh, ConfidenceVeryHigh, ConfidenceCertain |

## Interfaces (Consumer Protocols)

| Interface | Method | Embeds | Purpose |
| --- | --- | --- | --- |
| **Coded** | `ErrorCode() string` | `error` | Machine-readable identity |
| **Classified** | `ErrorFamily() Family` | `error` | Behavioral classification |
| **Contextual** | `ErrorContext() map[string]string` | `error` | Factual key-value details |
| **Retryable** | `IsRetryable() bool` | `error` | Explicit retry hint |
| **DiagnosticRule** | `Name()`, `Applicable(err)`, `Run(ctx, err)` | — | Deterministic error investigation |
| **DebugAgent** | `Analyze(ctx, err, diagnosis)` | — | Root cause analysis |

## Bounded Contexts

| Context | Description |
| --- | --- |
| **Classification** | Determining an error's behavioral family — the core concern |
| **Presentation** | Formatting errors for humans at the CLI/HTTP boundary (HandleError, templates) |
| **Diagnostics** | Investigating the system state when an error occurs (Runner, rules) |
| **Agent** | Synthesizing diagnostic results into root cause analysis and fix suggestions |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
