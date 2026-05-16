# Top 5 Stupidest Things in This Project

## 1. `ApplyFixes` is fraud

`agent/agent.go:269` — sets `step.Applied = true` and returns the step. That's it. No execution. No sandboxing. The entire `InvolvementAutonomous` level is a lie: it "approves" everything and then does nothing. The `AllowedCommands` and `ForbiddenCommands` configs gate... nothing. This isn't a scaffold — it's **performative theater** that makes the caller think something happened when it didn't.

## 2. `codeToWhat` / `codeToFix` — magic string matching on error codes

`handle.go:261-299` — decides what message the user sees by running `strings.Contains` on the error code. So `"db.not_found"` hits the `"not_found"` case AND would hit `"db"` if the order were different. `"git.config_timeout"` matches "git", "config", AND "timeout" — first one wins, silently. This is the library's **entire presentation layer** depending on substring matching with no registration, no override mechanism per code, and no way to know which pattern won. The `MessageTemplate` override exists but you have to discover it.

## 3. `DiagnosticRunner.Run` returns `any`

`handle.go:47` — the interface method returns `any` instead of `[]*DiagnosticResult`. This defeats Go's type system entirely. `OnDiagnosed` callback also receives `any`. The actual `diagnose.Runner.Run` returns `[]*DiagnosticResult` — a perfectly typed slice — but the interface erases it to `any` so the consumer gets to play type-assertion roulette. This exists because `handle.go` lives in the root package and can't import `diagnose` without a circular dependency... which is itself a smell that the packages are split wrong.

## 4. `SystemSnapshot` — built, exported, documented, zero consumers

`diagnose/context.go:16-63` — a 47-line exported struct with `GatherSystemSnapshot` that captures OS, arch, hostname, PID, working dir, all environment variables (redacted). Exported. Documented. **Called by exactly nothing.** No rule uses it. No test uses it. The agent doesn't use it. It's 47 lines of dead code with a regex for secret redaction that guards data nobody reads.

## 5. `lookupRegistered` — O(n) linear scan with `errors.Is` under lock

`classify.go:88-99` — every call to `Classify()` on an unclassified error iterates every registered sentinel calling `errors.Is` (which itself walks the unwrap chain) while holding `RLock`. If you register 200 sentinels and have a 5-deep error chain, that's 1,000 comparisons per classification. Under a read lock that blocks all registrations. And if any sentinel's `Is()` method calls `Classify()` (which it could, since sentinels are `error` interface values from third-party packages), you get a deadlock. The fix is trivial — use `sync.Map` or a typed registry — but the current design scales poorly and has a latent deadlock trap.
