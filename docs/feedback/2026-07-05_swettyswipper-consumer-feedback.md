# Consumer Feedback: go-error-family

**From:** SwettySwipperWeb integration session (2026-07-05)
**Perspective:** AI agent using the library for structured error handling across HTTP + CLI boundaries
**Tone:** Honest, direct, grateful but critical where warranted

---

## What Works Superbly

### 1. The Five Families — Correct Taxonomy

The five-family taxonomy is the best error classification model I've used:

| Family         | Retry?  | Exit | HTTP | Use case          |
| -------------- | ------- | ---- | ---- | ----------------- |
| Rejection      | No      | 1    | 400  | Bad input         |
| Conflict       | No      | 1    | 409  | State clash       |
| Transient      | **Yes** | 75   | 503  | Temporary failure |
| Corruption     | No      | 65   | 500  | Data integrity    |
| Infrastructure | No      | 69   | 503  | System down       |

Only `Transient` is retryable. This is the core design decision and it's correct. It makes retry logic trivial: `if errorfamily.IsRetryable(err) { retry() }`.

**This session:** We registered 7 Go sentinel errors (`sql.ErrNoRows` → Rejection, `context.DeadlineExceeded` → Transient, etc.) so that `Classify()` returns accurate families. The `RegisterClassifications` API was clean and worked perfectly.

### 2. Constructor API — Chainable and Type-Safe

```go
err := errorfamily.NewRejection("battle.min_items", "battles require at least 2 items")
err := errorfamily.WrapInfrastructure(dbErr, "api.error", "open database")
err := errorfamily.WrapTransient(httpErr, "discord.api", "fetch messages").
    WithContext("channel_id", channelID).
    WithContext("attempt", "3")
```

The typed family shortcuts (`NewRejection`, `WrapInfrastructure`, etc.) are preferred over the generic `Newf(family, code, msg)` because they encode the family at compile time. Our project uses both styles — we should standardize on the typed constructors.

### 3. CLI Boundary (`HandleError`)

```go
func main() {
    if err := run(); err != nil {
        os.Exit(errorfamily.HandleError(err))
    }
}
```

One line replaces `log.Fatal(err)` + `os.Exit(1)`. The BSD sysexits.h exit codes give operators actionable signals:

- Exit 1 = user error (Rejection/Conflict)
- Exit 65 = data corruption (Corruption)
- Exit 69 = infrastructure failure (Infrastructure)
- Exit 75 = temporary failure, try again (Transient)

**This session:** We replaced all `log.Fatal` + `os.Exit(1)` in `main.go` and `app.go` with `errorfamily.HandleError()`. Clean, correct, done in minutes.

### 4. Classification Chain

The classification precedence is well-designed:

1. Multi-error (`errors.Join`) → first non-Transient wins
2. `Classified` interface → `ErrorFamily()`
3. `Retryable` interface → infer Transient/Rejection
4. Registered sentinels via `errors.Is` chain
5. Default → Transient (fail-open)

The fail-open default is the correct choice — unknown errors should be retried, not silently dropped.

### 5. Zero External Dependencies

The library has zero external deps. This makes it trivial to adopt — no transitive dependency concerns, no version conflicts. Every Go project should use this.

---

## What's Confusing or Hard to Discover

### 1. `Classify(nil)` Returns `Rejection`

**Problem:** `Classify(nil)` returns `Rejection`, not `Infrastructure` or panicking.

**Why:** "nil error = caller's fault" — a Rejection.

**Impact:** This is surprising. Most error libraries return a neutral/default value for nil, not an active classification. We had to read the gotchas table to learn this.

**Ask:** Document this prominently in the godoc for `Classify`, not just in the SKILL.md.

### 2. `errors.Is` Matches on Code + Family Only

**Problem:** Two `*Error` instances with different messages but same code + family match via `errors.Is`.

**Why:** The comparison is intentionally code+family-based, not message-based.

**Impact:** This is correct for sentinel comparison but surprising if you expect `errors.Is` to compare full error equality.

**Ask:** Document this behavior in the `Error` type's godoc with an example.

### 3. `Wrap(nil, ...)` Returns `nil`

**Problem:** `errorfamily.Wrap(nil, family, code, msg)` returns `nil`. You can't construct an error wrapping nil.

**Impact:** This is actually great (nil-safe), but it means you can't use `Wrap` to create "optional" errors. The behavior should be documented.

**Ask:** Add to the constructor godoc: "Returns nil if err is nil — use `New*` constructors to create errors without a cause."

### 4. Template Substitution Uses `strings.ReplaceAll`

**Problem:** Message templates use `{key}` substitution via `strings.ReplaceAll`, not `html/template` or `text/template`.

**Impact:** No escaping, no nested templates, no conditionals. Just simple `{key}` → value replacement. This is probably fine for error messages, but worth documenting.

**Ask:** Document the substitution mechanism in `MessageTemplate` godoc.

---

## What's Missing

### 1. HTTP Status Code in the Error Type

**Problem:** The `Error` type has `Family()` → `HTTPStatus()` but no way to OVERRIDE the HTTP status for specific cases.

**Scenario:** A `Rejection` error should map to 400 by default, but `battle.not_found` should map to 404. Currently, the only way is `cqrshtmx.MapError` which checks for explicit status via `HTTPStatusCarrier` — but that interface is in cqrs-htmx, not go-error-family.

**Ask:** Add an optional `httpStatus int` field to `Error` with a `WithHTTPStatus(code int)` chainable method. This keeps the family-based default but allows per-error overrides.

### 2. Error Code in HTTP Error Responses

**Problem:** When `cqrshtmx.JSONErrorHandler` writes an error response, it includes `{"error": "...", "status": NNN}` but NOT the error code. API consumers can't programmatically distinguish between different rejection reasons without parsing the message string.

**Ask:** Export the error code in the JSON response when available: `{"error": "...", "code": "battle.exists", "status": 409}`. This may be a cqrs-htmx change, but the `Coded` interface should be more prominently used.

### 3. Registry Isolation — Not Clear How to Test

**Problem:** `RegisterClassification` mutates global state (`DefaultRegistry`). In tests, this can leak between test cases.

**Workaround:** Use `NewRegistry()` for test isolation — but the global functions (`Classify`, `IsRetryable`) always use `DefaultRegistry`.

**Ask:** Document a testing pattern: "For tests that need isolated classification, construct a `NewRegistry()` and pass it via `HandleConfig.Registry`. Avoid `RegisterClassification` in `init()` if tests need isolation."

### 4. No `ErrorFamily.HTTPStatus()` in Root Module

**Problem:** `Family.HTTPStatus()` may not exist directly on the `Family` type — it seems to be in cqrs-htmx's `event.Classify(err).HTTPStatus()`. The mapping from family → HTTP status is fundamental and should be in go-error-family itself.

**Ask:** Add `Family.HTTPStatus() int` to the root module so the family → HTTP mapping doesn't require importing cqrs-htmx or go-cqrs-lite.

---

## What's Over-Engineered

### 1. Diagnose Module (experimental)

The `diagnose/` module with filesystem/network/git/postgres rules adds goroutines and external command execution. This is valuable for CLI tools but heavy for web servers.

**Impact:** We don't use it. The experimental status is clear from the v0.x version.

### 2. Agent Module (experimental)

The `agent/` module for AI-powered error analysis is interesting but adds an LLM dependency (conceptually). The analysis-only design is correct, but most consumers won't need it.

**Impact:** We don't use it. Keep it experimental.

---

## Summary Scorecard

| Area                  | Rating | Notes                                     |
| --------------------- | ------ | ----------------------------------------- |
| Five-family taxonomy  | ★★★★★  | Correct, complete, well-designed          |
| Constructor API       | ★★★★★  | Type-safe, chainable, nil-safe            |
| CLI boundary          | ★★★★★  | One-line replacement for log.Fatal        |
| Classification chain  | ★★★★☆  | Solid, fail-open default correct          |
| Sentinel registration | ★★★★★  | Clean, RLock-protected                    |
| HTTP integration      | ★★★☆☆  | Status mapping scattered across libraries |
| Testing isolation     | ★★★☆☆  | Global registry mutates state             |
| Documentation         | ★★★★☆  | Good SKILL.md, some gaps in godoc         |
| Zero dependencies     | ★★★★★  | Perfect                                   |

---

## Top 3 Requests

1. **Add per-error HTTP status override** — `err.WithHTTPStatus(404)` for cases where the family default isn't specific enough.
2. **Move `Family.HTTPStatus()` into go-error-family** — don't require cqrs-htmx for the family → HTTP mapping.
3. **Document nil behaviors prominently** — `Classify(nil)` → Rejection, `Wrap(nil, ...)` → nil in godoc, not just the SKILL.md.

---

_This feedback is given with gratitude for a clean, zero-dep error library that gets the fundamentals right. The critique is offered to make the HTTP integration story as strong as the CLI story._

---

## Appendix: Resolution Status (2026-07-05)

> **Note:** This feedback document was missed during the initial feedback-implementation session. It was discovered during a self-review. Items below are marked with their current status as of 2026-07-23.

### What's Confusing or Hard to Discover

| #   | Item                                                                 | Status               | Resolution                                                                                                                                    |
| --- | -------------------------------------------------------------------- | -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| S1  | `Classify(nil)` → Rejection should be in **godoc**                   | ✅ **DONE (v0.8.0)** | `classify.go:56` now reads "Returns Rejection for nil errors."                                                                                |
| S2  | `errors.Is` matches on code+family — needs **godoc example**         | ✅ **DONE (v0.8.0)** | `error.go:50-51` now reads "Is supports errors.Is by matching error code and family. Two errors match if they have the same code AND family." |
| S3  | `Wrap(nil, ...)` → nil should be in **constructor godoc**            | ✅ **DONE (v0.8.0)** | `constructors.go:27-28` now reads "Returns nil if err is nil."                                                                                |
| S4  | Template `{key}` substitution should be in **MessageTemplate godoc** | ✅ **DONE (v0.8.0)** | `handle.go:84-90` documents `{key}` placeholder support on What/Why/Fix/WayOut fields.                                                        |

### What's Missing

| #   | Item                                                           | Status                                      | Resolution                                                                                                                                                                                                                                                                                       |
| --- | -------------------------------------------------------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| S5  | **Per-error HTTP status override** (`err.WithHTTPStatus(404)`) | ⏳ **NOT STARTED — design decision needed** | This is the #1 unaddressed request. It conflicts with "Family determines HTTP status." Needs a product decision from Lars: is HTTP status a classification concern (library) or a presentation concern (HTTP handler)? See status report `2026-07-05_03-21_consumer-feedback-session-review.md`. |
| S6  | Registry isolation testing pattern in docs                     | ⏳ **NOT STARTED**                          | The `Registry` type and `HandleConfig.Registry` exist, but the testing pattern ("use `NewRegistry()` for test isolation") isn't documented prominently.                                                                                                                                          |
| S7  | Error code in HTTP error responses                             | ✅ **PARTIALLY DONE**                       | The new `HTTPHandler` writes `{"family","code","message"}` — code IS included. But this only covers the standalone `HTTPHandler`, not the consumer's cqrs-htmx integration layer.                                                                                                                |

### What's Already Solved (by the SEC/browser-history session)

| #   | Item                                                                  | How it was solved                                                                                                             |
| --- | --------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| —   | "No standalone HTTP helper" (DiscordSync D9 / SwettySwipper implicit) | `HTTPHandler(fn) http.Handler` and `HTTPStatus(err) int` added in `http.go`.                                                  |
| —   | "No `Family.HTTPStatus()` in root module"                             | Already existed since v0.5.0 (`family.go`). This feedback was based on a misunderstanding — the method is in the root module. |
