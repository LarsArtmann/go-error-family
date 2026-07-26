# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and
traceable to its source. When an item ships, remove it here and record it in
`CHANGELOG.md` under the version it shipped in.

**Last updated:** 2026-07-26

---

## Active

### Medium Priority

- [ ] **Add test and lint CI steps for the examples module** — CI currently only runs `go build ./...` for examples. The bridge reference implementation added 19 tests and lint-sensitive code, but neither runs in CI. Add `go test -race -count=1 ./...` and `golangci-lint run ./...` steps to `ci.yml`. Source: bridge reference session 2026-07-26.
- [ ] **Add bridge guide to the website** — `website/src/content/docs/guides/` has pages for classification, diagnostics, HTTP/CLI, logs, benchmarks, and error-types, but NOT bridge patterns. The reference implementation (`examples/cmd/bridge/`) is the #1 adoption unblocker and should have a corresponding guide page. Link from `related-tools.mdx`. Source: bridge reference session 2026-07-26.
- [ ] **Link the reference implementation from `related-tools.mdx`** — the website page mentions bridge APIs but doesn't link to `examples/cmd/bridge/`. Source: bridge reference session 2026-07-26.

### Low Priority

- [ ] **Apply ACME TXT DNS record** — staged in Terraform but not applied (Namecheap API key is a placeholder). The HTTP challenge works now, but DNS-based verification is more robust for cert renewals. Source: status report 2026-07-23_05-07 section b.1.

---

## Design Decisions Resolved (2026-07-23)

All six design decisions from the "Design Decisions Needed" section have been resolved:

1. **Per-error HTTP status override** → **SHIPPED.** `Error.WithHTTPStatus(code int)` + `HTTPStatuser` interface. Mirrors the `ExitCoder`/`WithExitCode` pattern exactly: per-error override of family-level default, 0 = use family default. `HTTPStatus(err)` and `HTTPHandler` both check the interface. Rationale: `WithExitCode` already set the precedent — per-error overrides of family defaults are an accepted pattern. `battle.not_found` = 404 is undeniable.

2. **`Classify(nil)` semantics** → **KEPT Rejection.** Nil = caller bug. Changing to Transient would make `HTTPStatus(nil)` → 503 (success becomes "service unavailable"). The fail-open principle applies to _unknown_ errors, not _nil_ errors — they are fundamentally different situations. Changing is also breaking.

3. **Constructor context ergonomics** → **WON'T FIX.** `WithContextMap(map[string]string{...})` already exists for multi-value context. Functional options would conflict with copy-on-write design. The chain complaint is cosmetic, not structural.

4. **"Frozen" registry flag** → **WON'T FIX.** `atomic.Pointer` makes late registrations safe — no correctness issue to catch. Would break config-driven registration. Document the expected lifecycle instead of enforcing it.

5. **`RegisterClassificationType[T error]`** → **SHIPPED.** Two top-level functions: `RegisterClassificationType[T](family)` (DefaultRegistry) and `RegisterClassificationTypeFor[T](r, family)` (custom Registry). Go doesn't allow type parameters on methods, so the Registry-specific variant is a top-level function rather than a method. Non-breaking, pure sugar over `RegisterClassifier`.

6. **json/v2 migration strategy** → **REVERTED to `encoding/json`.** The root module no longer imports `encoding/json/v2`. Only 2 call sites marshaled tiny structs — v1 produces identical output. The `GOEXPERIMENT=jsonv2` requirement was the #1 adoption barrier for a zero-dependency library. Removed from flake.nix, CI workflows, and AGENTS.md.

---

## Completed

Completed items are logged in `CHANGELOG.md` under the version they shipped in. Do not list them here.
