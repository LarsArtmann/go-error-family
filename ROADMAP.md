# Roadmap

Long-term direction and raw ideas not yet refined into actionable tasks.
When an idea becomes bounded and actionable, it moves to `TODO_LIST.md`.

**Last updated:** 2026-07-23

---

## Direction

go-error-family is a stable classification core with a growing ecosystem
of opt-in modules (`diagnose`, `agent`, `bridge`). The taxonomy is proven across
multiple consumers (DiscordSync, browser-history, SwettySwipperWeb). The CLI
story was strengthened with per-error exit code overrides (`ExitCoder`),
idempotent wrapping (`WrapOnce`), typed context values (`WithContextAny`), and
panic-safe cause rendering (`safeCauseString`). v0.8.0 is released. The
focus now is: deploying the v0.8.0 API changes to
the website, verifying the full BuildFlow toolchain, and resolving the few open
design tensions from consumer feedback. The CI module-graph gate and
consumer-simulation job shipped; lint is at zero golangci-lint issues (though
the hierarchical-errors tool adds 50 nolint directives that warrant cleanup).

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

- ~~Per-error HTTP status override (`WithHTTPStatus`)~~ — **SHIPPED (v0.8.0)**, mirrors `ExitCoder`/`WithExitCode` pattern. Update: still under-adopted (~5 consumers for `HTTPHandler`); needs discoverability work.
- Error code in JSON responses is solved for `HTTPHandler` but not for consumers using their own HTTP layer
- Consider an `httperror` subpackage with richer response shaping
- OpenAPI/schema generation for the error JSON shape

### 3. Release Pipeline Hardening

The v0.6.0 phantom-`replace` incident exposed that `go.work` masks
consumer-facing bugs. CI needs to verify the module graph from the consumer's
perspective. The `GOWORK=off go list -m all` gate and consumer-simulation job
shipped in the `[Unreleased]` work (commit `e9c7219`); the remaining gaps are
tooling-level.

**Raw ideas:**

- Release automation script for coordinated multi-module tag cutting
- Deprecation notes for broken tags (v0.6.0 family)
- Pin-bump hygiene: submodules should bump root pins in lockstep on releases

### 4. Ecosystem Growth

The library has 50+ consumers importing the root classification core, but
higher-level APIs are under-adopted: `LogError` (~3 consumers), `HTTPHandler`
(~5), `errorfamilytest` (~3), `diagnose` (~3). The bridge module has **zero**
external consumers despite being correct, tested (95.6%), and fuzzed.

The bridge gap is demand and demonstration, not quality. The root cause is
that `samber/oops` adoption is near-zero across the ecosystem, and consumers
skip the enrichment layer entirely (classify→handle, not classify→enrich→
handle). Before building more bridge packages, the existing oops bridge needs
a **reference implementation** showing the full classify→enrich→handle flow
in a real application.

**Raw ideas:**

- **Reference implementation for oops + bridge + error-family stack** — the #1 unblocker for bridge adoption. Pick one real application and wire it end-to-end as the living example.
- More diagnostic submodules (`redis`, `docker`, `kubectl`)
- Bridge packages for other error enrichment libraries beyond oops (only after oops bridge has proven consumers)
- Integration guides for common frameworks (Echo, Gin, Chi, gRPC interceptors)
- Benchmark suite comparing classification overhead across versions
