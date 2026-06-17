# Execution Plan: Module Extraction

**Date:** 2026-06-17
**Prerequisite:** PROPOSAL.md reviewed and approved

---

## Ordered Task List

Each task is the smallest unit that leaves the project buildable and testable. Each is independently revertable (single commit).

### Tier 1 — Root (foundational)

#### Task 1: Create diagnose/go.mod

**What:** Extract `diagnose/` package into its own module.
**Dependencies:** None (first step).
**Steps:**

1. Create `diagnose/go.mod`:
   ```
   module github.com/larsartmann/go-error-family/diagnose
   go 1.26.3
   require github.com/larsartmann/go-error-family v0.5.0
   ```
2. Run `cd diagnose && go mod tidy`
3. Update `go.work` to add `./diagnose`
4. Verify: `go build ./...` from root
5. Verify: `cd diagnose && go test ./... -race`
6. Commit: `feat(modularize): extract diagnose/ into separate module`

**Rollback:** `rm diagnose/go.mod`, revert go.work

#### Task 2: Create agent/go.mod

**What:** Extract `agent/` package into its own module.
**Dependencies:** Task 1 (agent imports diagnose).
**Steps:**

1. Create `agent/go.mod`:
   ```
   module github.com/larsartmann/go-error-family/agent
   go 1.26.3
   require (
     github.com/larsartmann/go-error-family v0.5.0
     github.com/larsartmann/go-error-family/diagnose v0.0.0
   )
   ```
   (Use `v0.0.0` initially — go.work overrides for local dev. Tag later.)
2. Run `cd agent && go mod tidy`
3. Update `go.work` to add `./agent`
4. Verify: `go build ./...` from root
5. Verify: `cd agent && go test ./... -race`
6. Commit: `feat(modularize): extract agent/ into separate module`

**Rollback:** `rm agent/go.mod`, revert go.work

### Tier 2 — Untangle (fix version pins)

#### Task 3: Bump bridge root pin

**What:** Update bridge/go.mod to pin root at v0.5.0.
**Dependencies:** None (independent of Tasks 1-2).
**Steps:**

1. `cd bridge && go get github.com/larsartmann/go-error-family@v0.5.0 && go mod tidy`
2. Verify: `cd bridge && GOWORK=off go build ./...`
3. Commit: `chore(bridge): bump root dependency to v0.5.0`

#### Task 4: Bump git submodule deps

**What:** Update diagnose/git/go.mod to pin root at v0.5.0 and add diagnose dep.
**Dependencies:** Task 1 (git imports diagnose).
**Steps:**

1. `cd diagnose/git && go get github.com/larsartmann/go-error-family@v0.5.0`
2. `cd diagnose/git && go get github.com/larsartmann/go-error-family/diagnose@v0.0.0`
3. `cd diagnose/git && go mod tidy`
4. Verify: `cd diagnose/git && GOWORK=off go build ./...`
5. Commit: `chore(diagnose/git): bump root to v0.5.0, add diagnose dependency`

#### Task 5: Bump postgres submodule deps

**What:** Same pattern as Task 4 for diagnose/postgres.
**Dependencies:** Task 1 (postgres imports diagnose).
**Steps:** Same as Task 4, replacing `git` with `postgres`.
**Commit:** `chore(diagnose/postgres): bump root to v0.5.0, add diagnose dependency`

### Tier 3 — Infrastructure

#### Task 6: Full workspace verification

**What:** Run all tests across all modules.
**Dependencies:** Tasks 1-5.
**Steps:**

1. `go work sync`
2. `go build ./...` (workspace)
3. `go test ./... -race -count=1` (workspace)
4. For each module: `GOWORK=off go build ./... && GOWORK=off go test ./... -race`
5. `golangci-lint run ./...` in each module
6. Commit (if any changes from go work sync): `chore: sync go.work after module extraction`

#### Task 7: Update CI

**What:** Add test/lint steps for new diagnose/ and agent/ modules.
**Dependencies:** Task 6.
**Steps:**

1. Add CI steps for `./diagnose` (test + lint)
2. Add CI steps for `./agent` (test + lint)
3. Commit: `ci: add test and lint steps for diagnose and agent modules`

### Tier 4 — Polish

#### Task 8: Update documentation

**What:** Update README, AGENTS.md, SKILL.md, CHANGELOG.
**Dependencies:** Task 7.
**Steps:**

1. README: update module structure section
2. AGENTS.md: update build commands per module
3. SKILL.md: update architecture overview
4. CHANGELOG: add v1.0.0 entry (or v0.6.0 if not tagging v1.0 yet)
5. Commit: `docs: update for module extraction`

#### Task 9: Tag releases (if shipping v1.0)

**What:** Tag root as v1.0.0, diagnose as v0.1.0, agent as v0.1.0.
**Dependencies:** Task 8 + user approval.
**Steps:**

1. `git tag v1.0.0` (core)
2. `git tag diagnose/v0.1.0` (diagnose)
3. `git tag agent/v0.1.0` (agent)
4. Bump submodule deps from v0.0.0 to v0.1.0 where appropriate
5. Push tags

---

## Verification Checklist (per module after extraction)

- [ ] `go build ./...` passes in the module directory
- [ ] `go test ./...` passes in the module directory
- [ ] `go vet ./...` reports no issues
- [ ] `go mod tidy` changes nothing (already clean)
- [ ] No production dependency on test-only modules
- [ ] Import paths are correct (module path matches directory structure)
- [ ] Error types are accessible from consuming modules
- [ ] `internal/` packages are not imported from outside the module tree
- [ ] Root-level `go build ./...` still works (via go.work)
- [ ] `go.work` and `go.work.sum` are committed together
