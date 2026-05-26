# Decision: Modularize Diagnostic Rules + Typed Context Keys

**Date:** 2026-05-26
**Status:** Proposed — awaiting approval
**Decision-makers:** Lars

---

## Context

The `diagnose` package contains 4 diagnostic rules: PostgresRule, FilesystemRule, NetworkRule, GitRule. All are compiled into every consumer's binary via `DefaultRunner()`, even if the consumer never encounters git/postgres/filesystem/network errors. Two problems:

1. **Forced dependencies:** GitRule shells out to `git`. Consumers who never use git still carry the rule. Worse, if we want to use `go-git` (pure Go, no PATH dependency), every consumer would inherit that dependency.
2. **Untestable rules:** Rules that shell out to system commands are integration-test territory. The diagnose package coverage is 58% and unlikely to improve without significant mock infrastructure.

## Decision

**#1 Priority: Extract GitRule into its own submodule (`diagnose/git/`).**

**#2 Priority (future): Add typed ContextKey constants (Option A).**

---

## #1: Modularization

### Proposed Structure

```
github.com/larsartmann/go-error-family         (root module)
├── go.mod
├── classify.go, family.go, error.go, ...      (core types, zero deps)
├── handle.go                                  (CLI boundary)
├── agent/
│   └── agent.go                               (analysis agent)
├── diagnose/
│   ├── go.mod                                 (depends on root only)
│   ├── diagnose.go                            (Runner, interfaces, helpers)
│   ├── context.go                             (runCommand, commandExists)
│   ├── rules_filesystem.go                    (zero-dep rule)
│   └── rules_network.go                       (zero-dep rule)
├── diagnose/git/                              ← NEW submodule
│   ├── go.mod                                 (depends on root + diagnose + go-git)
│   └── rules_git.go                           (GitRule using go-git)
├── diagnose/postgres/                         ← FUTURE submodule
│   └── rules_postgres.go
├── go.work                                    ← workspace file
└── flake.nix                                  (updated for multi-module)
```

### Why GitRule First?

| Rule           | Current                           | After    | Dependency         |
| -------------- | --------------------------------- | -------- | ------------------ |
| FilesystemRule | `os.Stat`, `os.Create`            | same     | stdlib only        |
| NetworkRule    | `net.DialTimeout`                 | same     | stdlib only        |
| GitRule        | `exec.Command("git", ...)`        | `go-git` | `go-git` (new dep) |
| PostgresRule   | `exec.Command("pg_isready", ...)` | same     | still shells out   |

GitRule is the only rule that benefits from a non-stdlib dependency (`go-git`). Extracting it first proves the pattern. PostgresRule could follow later if we find a pure-Go postgres client worth depending on.

### Breaking Change

`DefaultRunner()` currently includes all 4 rules. After extraction, it would include only FilesystemRule and NetworkRule.

**Before:**

```go
runner := diagnose.DefaultRunner() // Postgres + Filesystem + Network + Git
```

**After:**

```go
runner := diagnose.DefaultRunner()  // Filesystem + Network only
runner.Register(gitrule.New())      // opt into git diagnostics
```

**Mitigation:** Bump to v0.2.0 (or v1.0.0 if we consider this API-stable). Document the change in CHANGELOG.md and README.md.

### Alternative: Keep DefaultRunner() Unchanged

Add `DefaultRunnerMinimal()` and keep `DefaultRunner()` as-is. But this creates API bloat and doesn't solve the dependency problem — `DefaultRunner()` still imports GitRule.

**Rejected.** The whole point is to make GitRule optional.

### Consumer Impact

| Consumer Type  | Before                     | After                                                         |
| -------------- | -------------------------- | ------------------------------------------------------------- |
| Uses all rules | `diagnose.DefaultRunner()` | `diagnose.DefaultRunner()` + `runner.Register(gitrule.New())` |
| Uses no rules  | `NewRunner()`              | Same                                                          |
| Wants git only | `DefaultRunner()`          | `gitrule.NewRunner()` (convenience)                           |

---

## #2: Typed Context Keys (Option A)

After modularization, add typed constants for context keys used in ruleSpec matching. This is a non-breaking enhancement.

```go
package diagnose

type ContextKey string

const (
    HostKey     ContextKey = "host"
    PortKey     ContextKey = "port"
    PathKey     ContextKey = "path"
    RepoPathKey ContextKey = "repo_path"
    DBHostKey   ContextKey = "db_host"
    DBPortKey   ContextKey = "db_port"
)

var filesystemSpec = ruleSpec{
    ContextKeys: []ContextKey{PathKey, "file", "dir", "directory"},
}
```

**Why this helps:**

- Prevents typos in context keys (compile-time vs runtime)
- Makes the "vocabulary" of the library explicit
- IDEs can autocomplete context keys

**Why this is #2:** It's a polish improvement, not a structural fix. The current string-based matching works fine.

---

## Execution Plan

### Phase 1: GitRule Extraction (this session)

| Step | File                             | Action                                                                        |
| ---- | -------------------------------- | ----------------------------------------------------------------------------- |
| 1    | `diagnose/git/go.mod`            | Create with module path `github.com/larsartmann/go-error-family/diagnose/git` |
| 2    | `diagnose/git/rules_git.go`      | Move GitRule from `diagnose/`. Replace `exec.Command` calls with `go-git`     |
| 3    | `diagnose/git/rules_git_test.go` | Move git tests from `diagnose/diagnose_test.go`                               |
| 4    | `diagnose/diagnose.go`           | Remove GitRule from `DefaultRunner()`                                         |
| 5    | `diagnose/diagnose_test.go`      | Remove git-specific tests                                                     |
| 6    | `go.work`                        | Create workspace file linking root, diagnose, diagnose/git                    |
| 7    | `flake.nix`                      | Update build for multi-module workspace                                       |
| 8    | Test                             | `go test ./...` in all modules                                                |
| 9    | Lint                             | `golangci-lint run` in all modules                                            |
| 10   | Commit                           | "feat(diagnose/git): extract GitRule to submodule with go-git"                |

### Phase 2: Documentation & Release

| Step | File           | Action                                       |
| ---- | -------------- | -------------------------------------------- |
| 11   | `README.md`    | Document new import path for GitRule         |
| 12   | `SKILL.md`     | Update diagnostic rules section              |
| 13   | `CHANGELOG.md` | Add v0.2.0 entry with breaking change notice |
| 14   | `AGENTS.md`    | Update module structure docs                 |
| 15   | Tag            | `git tag v0.2.0`                             |

### Phase 3: Typed Context Keys (future)

| Step | File                   | Action                                   |
| ---- | ---------------------- | ---------------------------------------- |
| 16   | `diagnose/diagnose.go` | Add `ContextKey` type and constants      |
| 17   | All rule files         | Replace string literals with constants   |
| 18   | Test                   | Verify no functional change              |
| 19   | Commit                 | "refactor(diagnose): typed context keys" |

---

## Open Questions

1. **go-git vs shell-out?** `go-git` is pure Go and testable, but adds a non-trivial dependency tree. Is the tradeoff worth it for a diagnostic rule that runs once per error?

2. **PostgresRule extraction?** Should we extract PostgresRule too (even without a pure-Go replacement), or leave it in the core diagnose package since it has no new dependencies?

3. **Version bump?** v0.2.0 (semver minor) or v1.0.0 (if we consider the API stable enough)?

4. **go.work vs replace directives?** `go.work` is the modern approach for local multi-module development. But it requires consumers to also use workspaces. `replace` directives in go.mod are more explicit but leak into published modules.

---

## Risks

| Risk                          | Likelihood | Impact | Mitigation                                              |
| ----------------------------- | ---------- | ------ | ------------------------------------------------------- |
| Breaking existing consumers   | High       | Medium | Clear CHANGELOG, version bump, README migration guide   |
| go-git dependency bloat       | Medium     | Low    | Only affects consumers who import `diagnose/git`        |
| Multi-module build complexity | Medium     | Medium | Test in CI, document workspace setup                    |
| Coverage drops further        | Medium     | Low    | Expected — extracted code is integration-test territory |
