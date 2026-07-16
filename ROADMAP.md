# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks.
When an idea becomes bounded and actionable, it moves to `TODO_LIST.md`.

**Last updated:** 2026-07-16

---

## Direction

go-error-family is a stable classification core (v0.8.0) with a growing ecosystem
of opt-in modules (`diagnose`, `agent`, `bridge`). The taxonomy is proven across
multiple consumers (DiscordSync, browser-history, SwettySwipperWeb). The CLI
story was strengthened in v0.8.0 with per-error exit code overrides (`ExitCoder`),
idempotent wrapping (`WrapOnce`), typed context values (`WithContextAny`), and
panic-safe cause rendering (`safeCauseString`). The focus now is: hardening the
release pipeline, improving discoverability (godoc), and resolving the few open
design tensions from consumer feedback.

## Themes

### 1. Consumer Discoverability

The #1 cross-cutting theme from all three consumer feedback sessions: surprising
behaviors are documented in SKILL.md but NOT in godoc where consumers actually
look. The library is correct; the documentation surface needs to meet consumers
where they are.

**Raw ideas:**

- Example functions (`ExampleClassify`, `ExampleWrap`, etc.) visible on pkg.go.dev
- A "common patterns" section that grows from real consumer usage
- Consider whether `Code()` vs `ErrorCode()` dual accessors should converge in a future major version

### 2. HTTP Story Parity

The CLI story (`HandleError`) is universally praised (10/10). The HTTP story is
weaker (6/10). `HTTPHandler` and `HTTPStatus` exist but consumers still build
custom layers.

**Raw ideas:**

- Per-error HTTP status override (`WithHTTPStatus`) — pending design decision
- Error code in JSON responses is solved for `HTTPHandler` but not for consumers using their own HTTP layer
- Consider an `httperror` subpackage with richer response shaping
- OpenAPI/schema generation for the error JSON shape

### 3. Release Pipeline Hardening

The v0.6.0 phantom-`replace` incident exposed that `go.work` masks
consumer-facing bugs. CI needs to verify the module graph from the consumer's
perspective.

**Raw ideas:**

- CI gate: `GOWORK=off go list -m all` per module
- CI consumer-simulation job (`go get @tag` in throwaway module)
- Release automation script for coordinated multi-module tag cutting
- Deprecation notes for broken tags (v0.6.0 family)
- Pin-bump hygiene: submodules should bump root pins in lockstep on releases

### 4. json/v2 Strategy

The root module uses `encoding/json/v2` (Go 1.26 experimental). This requires
`GOEXPERIMENT=jsonv2` from every consumer. Options:

- **Keep** until json/v2 becomes stable in a future Go release (then drop the GOEXPERIMENT requirement)
- **Revert** to `encoding/json` until json/v2 is stable (removes the consumer friction)
- **Centralize** behind a wrapper package so the choice is isolated

This needs a deliberate decision based on how long json/v2 stays experimental.

### 5. Ecosystem Growth

**Raw ideas:**

- More diagnostic submodules (`redis`, `docker`, `kubectl`)
- Bridge packages for other error enrichment libraries beyond oops
- Integration guides for common frameworks (Echo, Gin, Chi, gRPC interceptors)
- Benchmark suite comparing classification overhead across versions
