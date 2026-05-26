# Examples

Runnable examples demonstrating go-error-family patterns.

## CLI Handler

```bash
go run ./examples/cmd/cli
```

Shows `HandleError` with contextual messages, exit codes, and error wrapping.

## HTTP Handler

```bash
go run ./examples/cmd/http
```

Shows error classification mapped to HTTP status codes, JSON responses, and retry hints.

## Custom Diagnostic Rule

```bash
go run ./examples/cmd/custom_rule
```

Shows how to implement `diagnose.DiagnosticRule` with custom matching logic and fix suggestions.
