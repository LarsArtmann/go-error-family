# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and
traceable to its source. When an item ships, remove it here and record it in
`CHANGELOG.md` under the version it shipped in.

**Last updated:** 2026-07-23

---

## Active

### High Priority

- [ ] **Rebuild and deploy website** — the live site at `errorfamily.lars.software` is stale. API changes from v0.8.0 (ExitCoder, WrapOnce, WithContextAny, WithHTTPStatus, RegisterClassificationType) have not been deployed. The website docs have been audited and fixed (stale `SuggestedFix` refs corrected, missing v0.8.0 APIs added to api-reference.mdx, error-types.mdx, and changelog.mdx), but the build was never verified (`astro check`/`astro build` not run after the 12-factor guide was added). Source: status report 2026-07-23_15-08 section c.1.

### Medium Priority

- [ ] **Create reference implementation for oops + bridge stack** — the `bridge/` module has zero external consumers. Root cause: near-zero `samber/oops` adoption and no project demonstrating the classify→enrich→handle flow. Pick one real application, wire it through oops + bridge + error-family end-to-end, and document the pattern. This is the #1 unblocker for bridge adoption. Source: adoption audit 2026-07-23.

- [ ] **Verify full `buildflow` pipeline passes** — individual tools pass (go test, golangci-lint, nix flake check, hierarchical-errors), but the actual `buildflow` command was never run end-to-end. Source: status report 2026-07-23_15-52 section d.3.

- [ ] **Reduce hierarchical-errors nolint noise** — 50 `//nolint:hierarchical-errors` directives across the codebase, and golangci-lint warns "unknown linters: hierarchical-errors" on every run. Investigate config-file support (`.hierarchical-errors.toml`) or type-aware exemptions for `fmt.Formatter` and cleanup patterns. Source: status report 2026-07-23_15-52 section d.2.

- [ ] **Pin `version: latest` in `release.yml`** — 3 occurrences of `version: latest` for golangci-lint-action in the release workflow (CI workflow pins `v2.12.2`). Supply-chain reproducibility concern. Source: status report 2026-07-23_15-52 section c.3.

- [ ] **Investigate `gitignore-upserter:repair` failure** — showing as not-passing in BuildFlow output, never investigated. Source: status report 2026-07-23_15-52 section c.1.

### Low Priority

- [ ] **Apply ACME TXT DNS record** — staged in Terraform but not applied (Namecheap API key is a placeholder). The HTTP challenge works now, but DNS-based verification is more robust for cert renewals. Source: status report 2026-07-23_05-07 section b.1.

- [ ] **Set up CI/CD for website deploys** — no GitHub Actions workflow for automated deploys. Without it, the site depends on manual deploys and can silently rot. Source: status report 2026-07-23_05-07 section e.3.

---

## Design Decisions Resolved (2026-07-23)

All six design decisions from the "Design Decisions Needed" section have been resolved:

1. **Per-error HTTP status override** → **SHIPPED.** `Error.WithHTTPStatus(code int)` + `HTTPStatuser` interface. Mirrors the `ExitCoder`/`WithExitCode` pattern exactly: per-error override of family-level default, 0 = use family default. `HTTPStatus(err)` and `HTTPHandler` both check the interface. Rationale: `WithExitCode` already set the precedent — per-error overrides of family defaults are an accepted pattern. `battle.not_found` = 404 is undeniable.

2. **`Classify(nil)` semantics** → **KEPT Rejection.** Nil = caller bug. Changing to Transient would make `HTTPStatus(nil)` → 503 (success becomes "service unavailable"). The fail-open principle applies to _unknown_ errors, not _nil_ errors — they are fundamentally different situations. Changing is also breaking.

3. **Constructor context ergonomics** → **WON'T FIX.** `WithContextMap(map[string]string{...})` already exists for multi-value context. Functional options would conflict with copy-on-write design. The chain complaint is cosmetic, not structural.

4. **"Frozen" registry flag** → **WON'T FIX.** `atomic.Pointer` makes late registrations safe — no correctness issue to catch. Would break config-driven registration. Document the expected lifecycle instead of enforcing it.

5. **`RegisterClassificationType[T error]`** → **SHIPPED.** Two top-level functions: `RegisterClassificationType[T](family)` (DefaultRegistry) and `RegisterClassificationTypeFor[T](r, family)` (custom Registry). Go doesn't allow type parameters on methods, so the Registry-specific variant is a top-level function rather than a method. Non-breaking, pure sugar over `RegisterClassifier`.

6. **json/v2 migration strategy** → **REVERTED to `encoding/json`.** The root module no longer imports `encoding/json/v2`. Only 2 call sites marshaled tiny structs — v1 produces identical output. The `GOEXPERIMENT=jsonv2` requirement was the #1 adoption barrier for a zero-dependency library. Removed from flake.nix, CI workflows, and AGENTS.md.

- [ ] **v0.8.0 release** — v0.8.0 code is committed at HEAD but has **not been tagged** (latest tag is `v0.7.0`). The CHANGELOG `[Unreleased]` entry is prepared. A deliberate tag-and-release decision is needed.

---

## Completed

Completed items are logged in `CHANGELOG.md` under the version they shipped in. Do not list them here.
