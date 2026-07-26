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
curl http://localhost:8080/user              # 400 {"code":"user.missing_id","message":"id query parameter is required","retryable":false}
curl http://localhost:8080/user?id=notfound  # 400 {"code":"user.not_found","message":"user not found","retryable":false}
curl http://localhost:8080/user?id=dbfail    # 503 {"code":"db.timeout","message":"database connection timed out","retryable":true}
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

## Bridge Reference Implementation (oops + bridge + error-family)

```bash
go run ./examples/cmd/bridge
```

The canonical example of the **classify→enrich→handle** flow — the reference
implementation for the `bridge/` module. Demonstrates how `samber/oops`
(enrichment) and `go-error-family` (classification) work together in a real
HTTP checkout service.

Three patterns are shown:

1. **Pass-through** — library errors (classified with error-family) flow through
   oops enrichment unchanged. `Classify()` finds the family through the chain.
   No bridge call needed.
2. **AutoWrap** — application-created errors built with oops are classified by
   `bridge.AutoWrap`, which infers the family from oops tags and domain.
3. **Explicit Wrap** — `bridge.Wrap` combines oops enrichment with an explicit
   family when the application knows the classification.

The library layer (`examples/checkout/`) imports ONLY `go-error-family` —
proving the "libraries classify, applications enrich" architecture.

See `cmd/bridge/README.md` for the full pattern documentation and decision guide.

```bash
# After starting the server:
curl 'http://localhost:8090/orders?id=order-42'              # 200 success
curl 'http://localhost:8090/orders?id='                      # 400 Rejection (pass-through)
curl 'http://localhost:8090/orders?id=order-42&fail=db'      # 503 Transient (pass-through)
curl 'http://localhost:8090/orders?id=BADID'                 # 400 Rejection (AutoWrap)
curl 'http://localhost:8090/orders?id=order-42&fail=inv'     # 409 Conflict (explicit Wrap)
```
