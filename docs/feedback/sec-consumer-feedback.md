# go-error-family — SDK Feedback from SEC

**Consumer:** [SEC](https://github.com/larsartmann/sec) — dice-based game (CQRS + HTMX)
**Date:** 2026-07-05
**Version used:** v0.5.1
**Session:** Deep integration of error classification across HTTP boundary, CLI boundary, and domain layer

---

## What worked superbly

### 1. `Classify(err)` + `HTTPStatus()` — the core value prop

The classify-then-map pattern is excellent. One call (`errorfamily.Classify(err)`) gives me the family, and `.HTTPStatus()` maps it to the right HTTP status. Replacing 7 blanket `writeInternalError(500)` sites with `writeDispatchError` was mechanical and safe once the pattern was in place.

### 2. Classification through wrapping

`fmt.Errorf("dispatch command: %w", err)` preserves the error chain so `Classify` still detects the `Classified` interface via `errors.As`. This is critical — without it, every wrapping site would need manual unwrapping. The design is correct and I trusted it immediately.

### 3. `errors.Is` matching by code+family

`GameNotActive("play", "finished")` matches `ErrGameNotActive` via `errors.Is` because they share the same code+family. This lets me use sentinel checks for broad categories while getting specific messages from factories. Elegant.

### 4. `RegisterStdlibDefaults`

One call registers `context.DeadlineExceeded→Transient`, `sql.ErrNoRows→Rejection`, etc. I just called it in `init()` and forgot about it. The fail-open-to-Transient default for unknown errors is the right choice for retry semantics.

### 5. `MessageTemplate` with domain-specific messages

`RegisterTemplate` lets me register user-facing messages per error code (`"game.not_found" → "This game doesn't exist."`). The `{key}` placeholder system for error context interpolation is well-designed.

---

## Pain points and friction

### 1. No `errors.As` equivalent in the package

I initially tried `errorfamily.As(err, &coded)` and got a compile error. There's no `As` function — you must use stdlib `errors.As` with the `errorfamily.Coded` interface. This is correct (stdlib `errors.As` is the right tool) but it's not discoverable. I had to grep the source to find the `Coded` interface.

**Suggestion:** Add a doc comment on `Classify` pointing to `errors.As(err, &coded)` for code extraction, or export a helper like `ErrorCode(err) string` that does the `errors.As` internally.

### 2. `RegisterClassification` vs `Classified` interface — when to use which?

The distinction between "implement Classified on your own errors" vs "RegisterClassification for third-party errors" was clear from docs, but the practical boundary needed thought. Domain errors that I own → `NewRejection/NewConflict`. Third-party errors → `RegisterClassification`. This is correct but could use a decision-tree diagram in the README.

### 3. `HandleConfig.TemplateOverride` unused — unclear how to wire it

I registered templates via `RegisterTemplate` but the `HandleConfig` / `HandleError` pipeline isn't wired into my HTTP handlers. I extract the code manually via `errors.As` and look up my own `domainMessage(code)` function. The `HandleError` family of functions seems designed for a different use case (CLI? batch processing?) and the connection to HTTP error responses isn't obvious.

**Suggestion:** Add an example showing `HandleErrorWithConfig` used at an HTTP boundary, or export a `TemplateForCode(code) (MessageTemplate, bool)` helper so consumers can look up registered templates without reimplementing the lookup.

### 4. Five families — is Corruption really 422?

`Corruption → HTTP 422 (Unprocessable Entity)` surprised me. Corruption (stored data damage, unmarshal failure) feels more like a 500 (server error) than a 422 (client sent unprocessable data). The client didn't do anything wrong — the stored data is damaged. I'd expect Corruption → 500 and Infrastructure → 500, distinguished only by severity/logging.

**Suggestion:** Document the rationale for each HTTP mapping. The current table in `family.go` is a good start but doesn't explain _why_.

### 5. Missing: `IsRetryable(err) bool` convenience

`Retryable` is an interface but there's no top-level `IsRetryable(err) bool` helper. I have to do `errors.As(err, &retryable)` manually. A convenience function would match the `Classify(err)` pattern.

---

## Ideas for improvement

### 1. `errorfamily.Code(err) string` — one-liner code extraction

```go
func Code(err error) string {
    var coded Coded
    if errors.As(err, &coded) {
        return coded.ErrorCode()
    }
    return ""
}
```

Would eliminate the 5-line boilerplate I wrote in `dispatchErrorCode`.

### 2. HTTP middleware adapter

An `errorfamily.HTTPErrorHandler(fn ErrorHandler)` that wraps an `http.HandlerFunc` and catches errors returned from a handler. This would bridge the gap between the classification system and HTTP frameworks.

### 3. Structured logging integration

`HandleError` returns an `int` (exit code). Consider a `LogError(err, slog.Logger)` or similar that logs structured fields (family, code, message, retryable) — this is what every consumer will build anyway.

### 4. Testing helpers

Export test assertion helpers: `AssertFamily(t, err, Family)`, `AssertCode(t, err, string)`. I wrote these myself in 3 projects now.

---

## Overall verdict

The library solves a real problem well: error classification without leaking internals, with correct HTTP mapping, and through error wrapping. The `NewRejection/NewConflict/NewTransient` constructors are ergonomic. The fail-open-to-Transient default is the right call.

The main gap is discoverability — the `Coded` interface extraction pattern, the `HandleConfig` pipeline, and the template system all required source-reading to understand. More examples and a decision-tree doc would close that gap.

---

## Appendix: Resolution Status (2026-07-05)

### Pain Points

| #   | Item                                                               | Status                 | Resolution                                                                                                                                                                                                                                     |
| --- | ------------------------------------------------------------------ | ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| PP1 | No `errors.As` equivalent — `Code(err)` helper                     | ✅ **DONE**            | Added `errorfamily.Code(err) string` in `classify.go`. Walks the unwrap chain via `errors.AsType[Coded]`. Returns `""` if no code found. `HandleError`'s internal `extractCode` refactored to delegate to it.                                  |
| PP2 | Decision-tree for `RegisterClassification` vs `Classified`         | ✅ **DONE**            | Added ASCII decision tree to README: own→Classified, sentinel→RegisterClassification, dynamic→RegisterClassifier, else→Transient default.                                                                                                      |
| PP3 | `HandleConfig.TemplateOverride` unclear; no template lookup helper | ✅ **DONE**            | Added `TemplateForCode(code) (MessageTemplate, bool)` — both as `Registry.TemplateForCode` and package-level convenience. Checks registered templates → built-in defaults. Lets HTTP/gRPC consumers look up messages without the CLI pipeline. |
| PP4 | Corruption → 422 concern; document HTTP rationale                  | ✅ **RESOLVED**        | Corruption was already 500 (not 422 — this was based on an older version). Added per-family rationale to `Family.HTTPStatus()` godoc explaining each mapping and why Corruption→500 (not 422).                                                 |
| PP5 | Missing `IsRetryable(err) bool` convenience                        | ✅ **ALREADY EXISTED** | `errorfamily.IsRetryable(err) bool` already existed in `classify.go` since v0.5.0. No action needed.                                                                                                                                           |

### Ideas for Improvement

| #     | Item                           | Status      | Resolution                                                                                                                                                                                                                                                                                |
| ----- | ------------------------------ | ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| IDEA1 | `errorfamily.Code(err) string` | ✅ **DONE** | Implemented exactly as suggested. See PP1 above.                                                                                                                                                                                                                                          |
| IDEA2 | HTTP middleware adapter        | ✅ **DONE** | Added `HTTPHandler(fn) http.Handler` and `HTTPStatus(err) int` in `http.go`. Wraps error-returning handlers, writes safe JSON responses (`{family, code, message}`) with the correct status code. **Never leaks `err.Error()`** — message comes only from a registered `MessageTemplate`. |
| IDEA3 | Structured logging integration | ✅ **DONE** | Added `LogError(err, *slog.Logger)` and `LogErrorContext(ctx, err, logger)` in `log.go`. Transient→Warn, all others→Error. Logs `family`, `code`, `retryable`, and each context key prefixed with `context.`. Nil error = no-op; nil logger = `slog.Default()`.                           |
| IDEA4 | Testing helpers                | ✅ **DONE** | Added `errorfamilytest` subpackage (`AssertFamily`, `AssertCode`, `AssertRetryable`, `AssertContext`, `AssertContextMissing`). Mirrors `net/http/httptest` — keeps `testing` out of the production package.                                                                               |
