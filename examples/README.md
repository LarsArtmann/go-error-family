# Examples

Runnable examples demonstrating go-error-family patterns.

## CLI Handler

```bash
go run ./examples/cmd/cli
```

Demonstrates `HandleError` at the CLI boundary — contextual messages, exit codes, and error wrapping.

```
Error: startup.failed
Check your input and try again.
```

Exit code: `1` (Rejection).

## HTTP Handler

```bash
go run ./examples/cmd/http
```

Maps error families to HTTP status codes with structured JSON responses and retry hints.

```bash
# After starting the server:
curl http://localhost:8080/user              # 400 {"code":"user.missing_id","retryable":false}
curl http://localhost:8080/user?id=notfound  # 400 {"code":"user.not_found","retryable":false}
curl http://localhost:8080/user?id=dbfail    # 503 {"code":"db.timeout","retryable":true}
```

## Custom Diagnostic Rule

```bash
go run ./examples/cmd/custom_rule
```

Shows how to implement `diagnose.DiagnosticRule` from scratch — matching by context keys and error codes, producing actionable findings.

```
[rate_limit] healthy: Rate limited — wait 12 before retrying
  Fix: Wait for the duration specified in the Retry-After header
```

This example demonstrates the pattern for writing your own rules: implement `Name()`, `Applicable()`, and `Run()`, then compose rules into a `diagnose.Runner`.
