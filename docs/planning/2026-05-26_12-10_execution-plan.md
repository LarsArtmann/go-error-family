# Execution Plan — 2026-05-26 12:10

## What I Forgot / Should Improve

1. **No benchmarks anywhere** — zero performance baselines for `Classify`, `Runner.Run`, `HandleError`
2. **`RunCommand` ignores `ctx.Done()`** — `context.go` sets up timeout but doesn't check cancellation
3. **No `examples/` directory** — library has zero example programs
4. **`*Error` doesn't support `Unwrap() []error`** — can't compose multiple errors natively
5. **DiagnosticResult.Duration not tested** — tracked but not verified
6. **golangci.yml exclusions missing git/postgres paths** — no `wrapcheck`, `noctx` exclusions

## Execution Plan (Pareto-sorted)

### P1: Benchmarks (Low effort, High value)
1. Add `benchmark_test.go`: `BenchmarkClassify`, `BenchmarkClassifyMultiError`, `BenchmarkHandleError`, `BenchmarkRunnerRun`

### P2: Correctness fixes (Medium effort, High value)
2. Check `ctx.Done()` in `RunCommand` — respect cancellation mid-execution
3. Add `Duration` assertion to `TestDiagnosticResultDuration` in diagnose_test.go

### P3: Type safety (Low effort, Medium value)
4. Add `ContextKey` type + constants in `diagnose/diagnose.go`

### P4: Documentation / Examples (Medium effort, Medium value)
5. Add `examples/` with CLI, HTTP handler, custom rule examples

### P5: Nice-to-have (Low effort, Low value)
6. Extract `partsBuilder` helper from handle.go duplication

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

1. Commit benchmarks (self-contained)
2. Commit context cancellation fix (self-contained)
3. Commit ContextKey types (self-contained)
4. Commit examples (self-contained)
5. Final verification + push
