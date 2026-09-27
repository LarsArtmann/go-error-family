# Execution Plan — 2026-05-26 12:10

## What I Forgot / Should Improve

1. ~~**No benchmarks anywhere** — zero performance baselines for `Classify`, `Runner.Run`, `HandleError`~~ done — benchmark suite shipped
2. ~~**`RunCommand` ignores `ctx.Done()`** — `context.go` sets up timeout but doesn't check cancellation~~ done — ctx cancellation honored
3. ~~**No `examples/` directory** — library has zero example programs~~ done — examples/ module shipped
4. ~~**`*Error` doesn't support `Unwrap() []error`** — can't compose multiple errors natively~~ **Won't implement — declined — errors.Join + Classify chosen instead.**
5. ~~**DiagnosticResult.Duration not tested** — tracked but not verified~~ done — Duration tested
6. ~~**golangci.yml exclusions missing git/postgres paths** — no `wrapcheck`, `noctx` exclusions~~ done — golangci exclusions curated

## Execution Plan (Pareto-sorted)

### P1: Benchmarks (Low effort, High value)

1. ~~Add `benchmark_test.go`: `BenchmarkClassify`, `BenchmarkClassifyMultiError`, `BenchmarkHandleError`, `BenchmarkRunnerRun`~~ done — benchmark_test.go shipped

### P2: Correctness fixes (Medium effort, High value)

2. ~~Check `ctx.Done()` in `RunCommand` — respect cancellation mid-execution~~ done — ctx fix shipped
3. ~~Add `Duration` assertion to `TestDiagnosticResultDuration` in diagnose_test.go~~ done — Duration test shipped

### P3: Type safety (Low effort, Medium value)

4. ~~Add `ContextKey` type + constants in `diagnose/diagnose.go`~~ done — ContextKey type + constants shipped

### P4: Documentation / Examples (Medium effort, Medium value)

5. ~~Add `examples/` with CLI, HTTP handler, custom rule examples~~ done — examples/ shipped (cmd/cli, http, custom_rule, bridge)

### P5: Nice-to-have (Low effort, Low value)

6. ~~Extract `partsBuilder` helper from handle.go duplication~~ **Won't implement — declined — superseded by diagnose helper design.**

## Design Decisions

- **Benchmarks:** Use standard `testing.B` — no deps needed
- **Context cancellation:** Poll `ctx.Done()` in `RunCommand` loop
- **ContextKey:** `type ContextKey string` — same pattern as `http.Header` keys
- **Examples:** Keep in root `examples/` (separate modules if they need deps, but simple for now)
- **partsBuilder:** Skip if benchmarks prove handle.go is not hot path

## Libraries to Consider

- `github.com/google/go-cmp` — rejected (adds dep, zero-dep policy)
- `github.com/benbjohnson/clock` — rejected (adds dep for tests)
- `golang.org/x/exp/maps` — we already use stdlib `maps` (Go 1.21+)
- All new code uses stdlib only

## Order of Execution

1. ~~Commit benchmarks (self-contained)~~ done
2. ~~Commit context cancellation fix (self-contained)~~ done
3. ~~Commit ContextKey types (self-contained)~~ done
4. ~~Commit examples (self-contained)~~ done
5. ~~Final verification + push~~ done — verified and pushed
