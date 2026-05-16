# Resolving the Top 5 Stupidest Things

## 1. `ApplyFixes` is fraud → Delete it. The library should propose, not execute.

The library's job is error classification and diagnosis. Executing shell commands is a **consumer concern**. No library should ever run `mkdir`, `chmod`, or `git` commands on behalf of the caller — that's a security nightmare.

**Fix:** Remove `ApplyFixes` from the `DebugAgent` interface. Remove `shouldApply`, `AllowedCommands`, `ForbiddenCommands`, `ConfirmFunc`, `MaxRetries` from `Config`. The agent's contract becomes:

```go
type DebugAgent interface {
    Analyze(ctx context.Context, err error, diagnosis []*diagnose.DiagnosticResult) (*AgentResult, error)
}
```

`AgentResult` already contains `FixSteps` with `Command` and `Risk`. The consumer reads them and decides what to do. This makes the library honest — it proposes fixes, the consumer applies them. Zero security surface. Zero fraud.

The `Involvement` enum stays but becomes the consumer's concern, not the library's. If a consumer wants a `ConfirmFunc`, they build it themselves around `FixStep.Risk`.

---

## 2. `codeToWhat` / `codeToFix` → Replace with explicit registry, keep fallback

The problem: implicit substring matching with hidden precedence. `"git.config_timeout"` matches three patterns and you can't predict which wins.

**Fix:** Two-tier system — explicit lookup first, family-based fallback second.

```go
var defaultTemplates = map[string]MessageTemplate{
    "file.not_found":    {What: "A required resource was not found.", Fix: "Check that the path and resource name are correct."},
    "permission.denied": {What: "Permission was denied.", Fix: "Check file permissions or run with appropriate privileges."},
    "db.timeout":        {What: "The database operation timed out.", Fix: "Increase the timeout or check system resources."},
    // ...
}
```

Kill `codeToWhat` and `codeToFix`. Replace with:

1. Check `HandleConfig.TemplateOverride[code]` (consumer override — already exists)
2. Check `defaultTemplates[code]` (exact match — new)
3. Fall back to `familyDefaultMessage(family)` (already exists, works well)

No substring matching. No ambiguity. Consumers add templates by exact code. Unknown codes get a sensible family-based message. The `register` function becomes `RegisterTemplate(code string, tmpl MessageTemplate)` — thread-safe, explicit, predictable.

---

## 3. `DiagnosticRunner` returns `any` → Move `HandleError` to a `cli` package

The root cause: `handle.go` can't import `diagnose` (circular dependency). So it defines its own interface with `any` return type, erasing the real type.

**Fix:** Move `HandleError`, `HandleConfig`, `HandleResult`, `MessageTemplate` into a `cli` subpackage. Then `cli` imports both root (for `Classify`, `Family`) and `diagnose` (for `DiagnosticResult`). No circular dependency, no `any`, no type erasure.

```
errorfamily/       — Family, Error, Classify, constructors (pure protocol)
diagnose/          — Runner, rules, DiagnosticResult (diagnosis)
cli/               — HandleError, HandleConfig, MessageTemplate (presentation)
agent/             — DebugAgent, Analyze (AI-assisted debugging)
```

This is the correct package split. The current structure mixes protocol and presentation in the root package.

---

## 4. `SystemSnapshot` → Delete it. YAGNI.

Zero callers. Zero tests. 47 lines of dead code with a secret-redaction regex guarding data nobody reads.

**Fix:** Delete `SystemSnapshot`, `GatherSystemSnapshot`, `isSecretKey`, `secretPattern`, `mustGetwd`. If someone needs system snapshots later, they write them in the consumer when they have an actual use case. The `runCommand` and `commandExists` helpers stay — those are used by the rules.

---

## 5. `lookupRegistered` O(n) under lock → Snapshot the map, iterate lock-free

The performance (O(n)) is fine for reasonable n (<50 sentinels). The **real** danger is the deadlock: `errors.Is` under `RLock` can call back into `Classify` if a sentinel's `Is()` method classifies.

**Fix:** Copy the map, release the lock, then iterate:

```go
func lookupRegistered(err error) (Family, bool) {
    registry.mu.RLock()
    snapshot := make(map[error]Family, len(registry.entries))
    maps.Copy(snapshot, registry.entries)
    registry.mu.RUnlock()

    for sentinel, family := range snapshot {
        if errors.Is(err, sentinel) {
            return family, true
        }
    }
    return Rejection, false
}
```

No lock held during `errors.Is`. No deadlock possible. Small copy cost (n is tiny). Registration remains thread-safe. The copy is a snapshot — registrations after the copy won't be seen, but `RegisterClassification` is called from `init()`, so this is irrelevant in practice.

---

## Priority Order

| # | Fix | Effort | Impact |
|---|---|---|---|
| 4 | Delete `SystemSnapshot` | 5min | Remove dead code |
| 5 | Snapshot map in `lookupRegistered` | 10min | Eliminate deadlock risk |
| 1 | Remove `ApplyFixes` and command execution config | 30min | Stop lying to callers |
| 2 | Replace `codeToWhat`/`codeToFix` with template registry | 1h | Predictable, testable presentation |
| 3 | Move `HandleError` to `cli` package | 1h | Kill `any`, proper package split |

Items 4 and 5 are no-brainers. Items 1–3 are architectural — do them before v0.1.0 or the API will be harder to change.
