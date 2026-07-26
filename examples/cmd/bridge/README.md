# Bridge Reference Implementation: oops + bridge + error-family

The canonical example of the **classify→enrich→handle** flow. This is the
reference implementation that the bridge module (`go-error-family/bridge`)
was built for — demonstrating how `samber/oops` (enrichment) and
`go-error-family` (classification) work together in a real application.

## The Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ LIBRARY LAYER (examples/checkout)                           │
│                                                             │
│ imports: go-error-family ONLY                               │
│ never imports: oops                                         │
│                                                             │
│ Returns classified errors: NewRejection, NewTransient, etc. │
│ The library knows its domain contract. It does NOT know     │
│ how the application will observe or enrich those errors.    │
└──────────────────────────┬──────────────────────────────────┘
                           │ errors flow up
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ APPLICATION LAYER (this file)                               │
│                                                             │
│ imports: oops, bridge, errorfamily, checkout                │
│                                                             │
│ Enriches errors with oops (stack traces, trace IDs, domain  │
│ context). Uses the bridge to combine enrichment with         │
│ classification where the application creates its own errors. │
└──────────────────────────┬──────────────────────────────────┘
                           │ enriched + classified errors
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ BOUNDARY (errorfamily.HTTPHandler / HandleError)            │
│                                                             │
│ Classifies once → HTTP status code / exit code              │
│ Writes safe JSON response (never leaks err.Error())         │
│ Logs structured fields (family, code, context.*)            │
└─────────────────────────────────────────────────────────────┘
```

## Why This Exists

The bridge module had **zero external consumers** (adoption audit 2026-07-23).
The root causes were:

1. Near-zero `samber/oops` adoption in the ecosystem
2. No project demonstrating the classify→enrich→handle flow end-to-end
3. No reference implementation showing when to use the bridge vs. raw classification

This example is the fix for cause #2 and #3. It proves the pattern works and
shows exactly when each bridge API is the right tool.

## The Three Patterns

### Pattern 1: Pass-through (no bridge needed)

**When:** The library already classified the error.

The library returns `errorfamily.NewRejection(...)`. The application wraps it
with `oops.Wrap(err)` for enrichment. `Classify()` finds the family through the
chain — **no bridge call needed**.

```go
order, err := store.GetOrder(orderID)  // library returns NewRejection
if err != nil {
    enriched := oops.In("checkout").
        Trace(traceID).
        With("user_id", userID).
        Wrap(err)              // oops enrichment wraps the classified error

    return enriched            // Classify() finds Rejection through the chain
}
```

**Why no bridge?** `errors.AsType[Classified]` traverses the Unwrap chain.
The OopsError wrapper's `Unwrap()` returns the library's `*Error`, which
implements `Classified`. Classification survives automatically.

### Pattern 2: AutoWrap (oops-first, bridge infers family)

**When:** The application creates its own errors with oops.

The application builds a rich error with oops (domain, tags, code, context).
`bridge.AutoWrap(err)` infers the family from the oops metadata:

```go
rich := oops.In("validation").
    Tags("rejection").
    Code("order.id_format").
    With("received", orderID).
    Errorf("invalid order ID format")

return bridge.AutoWrap(rich)   // infers Rejection from "validation" domain
```

**Inference cascade:** tags (developer-intentional) → domain (structural) →
Transient (fail-open). See `bridge.InferFamily`.

### Pattern 3: Explicit Wrap (application knows the family)

**When:** The application knows the exact family and wants full control.

```go
rich := oops.In("checkout").
    Code("checkout.inventory_blocked").
    Trace(traceID).
    Wrap(invErr)

return bridge.Wrap(rich, errorfamily.Conflict)  // explicit family
```

## The Killer Feature: One Error, Two Representations

The same enriched+classified error object produces **two completely different
outputs** depending on the audience:

| Audience      | What they see                                    | How                              |
| ------------- | ------------------------------------------------ | -------------------------------- |
| **Operator**  | Full stack trace, trace ID, all context fields   | `logger.Error(fmt.Sprintf("%+v", err))` |
| **HTTP client** | Safe `{family, code, message}` — no internals  | `errorfamily.HTTPHandler`        |

```go
// Internal log: operator sees everything
logger.Error(fmt.Sprintf("%+v", enriched), "trace_id", traceID)
// → full oops stack trace, domain, tags, all context keys

// HTTP response: client sees only safe fields
return enriched  // → HTTPHandler writes {"family":"transient","code":"db.timeout"}
```

The raw `err.Error()` is **never** sent to the client. This is by design:
`HTTPHandler` uses only the family, code, and optionally a registered
`MessageTemplate`.

## Running the Example

```bash
go run ./examples/cmd/bridge
```

Then in another terminal:

```bash
# Success
curl 'http://localhost:8090/orders?id=order-42'
# → {"order_id":"order-42","amount_cents":9900,...}

# Pattern 1: pass-through — library Rejection flows through oops enrichment
curl 'http://localhost:8090/orders?id='
# → 400 {"family":"rejection","code":"order.missing_id"}

# Pattern 1: pass-through — library Transient
curl 'http://localhost:8090/orders?id=order-42&fail=db'
# → 503 {"family":"transient","code":"order.db_timeout"}

# Pattern 2: AutoWrap — application validation via bridge.AutoWrap
curl 'http://localhost:8090/orders?id=BADID'
# → 400 {"family":"rejection","code":"order.id_format"}

# Pattern 3: explicit Wrap — bridge.Wrap with Conflict family
curl 'http://localhost:8090/orders?id=order-42&fail=inv'
# → 409 {"family":"conflict","code":"checkout.inventory_blocked"}
```

## Decision Guide: When to Use What

| Situation                                       | Use                        |
| ----------------------------------------------- | -------------------------- |
| Library already classified the error            | `oops.Wrap(err)` — done    |
| Application creates errors with oops            | `bridge.AutoWrap(err)`     |
| Application knows the exact family              | `bridge.Wrap(err, family)` |
| Need just the family from an oops error         | `bridge.InferFamily(err)`  |
| Don't use oops at all                           | `errorfamily.New*` / `Wrap*` |

## File Layout

```
examples/
  checkout/                  ← LIBRARY: imports only errorfamily
    checkout.go                Returns classified errors
    checkout_test.go           Verifies classification at the library level
  cmd/bridge/                ← APPLICATION: this reference implementation
    main.go                    HTTP server with all three patterns
    main_test.go               Tests proving each pattern works end-to-end
    README.md                  This file
```

## Key Insight

The bridge is **not** a mandatory layer between oops and error-family. In the
most common case (Pattern 1), you don't need it at all — library classification
survives oops enrichment automatically. The bridge adds value when the
**application** is the one creating errors with oops and needs to classify them
(Patterns 2 and 3).

This is why the architecture says "libraries classify, applications enrich" —
not "libraries classify, applications bridge." The bridge is a tool for the
application layer, not a required pipeline stage.
