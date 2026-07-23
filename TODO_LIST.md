# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and
traceable to its source. When an item ships, remove it here and record it in
`CHANGELOG.md` under the version it shipped in.

**Last updated:** 2026-07-23

---

## Active

### High Priority

- [ ] **Rebuild and deploy website** — the live site at `errorfamily.lars.software` is stale. API changes from v0.8.0 (ExitCoder, WrapOnce, WithContextAny) have not been deployed. The website docs have been audited and fixed (stale `SuggestedFix` refs corrected, missing v0.8.0 APIs added to api-reference.mdx, error-types.mdx, and changelog.mdx). Source: status report 2026-07-16_06-30 section c.2.

### Low Priority

- [ ] **Apply ACME TXT DNS record** — staged in Terraform but not applied (Namecheap API key is a placeholder). The HTTP challenge works now, but DNS-based verification is more robust for cert renewals. Source: status report 2026-07-23_05-07 section b.1.

- [ ] **Set up CI/CD for website deploys** — no GitHub Actions workflow for automated deploys. Without it, the site depends on manual deploys and can silently rot. Source: status report 2026-07-23_05-07 section e.3.

---

## Design Decisions Needed

These require a product decision before implementation. They are NOT actionable tasks yet.

- [ ] **Per-error HTTP status override (`Error.WithHTTPStatus(code int)`)** — SwettySwipper's #1 request. `battle.not_found` should be 404, not the family default 400. Core tension: is HTTP status a classification concern (library) or a presentation concern (HTTP handler)? Source: SwettySwipper S5.

- [ ] **`Classify(nil)` semantics** — DiscordSync argues Rejection is inconsistent with fail-open. Options: keep Rejection (current), change to Infrastructure (programming error), or change to Transient (fail-open). Changing is breaking. Source: DiscordSync D4.

- [ ] **Constructor context ergonomics** — builder pattern, variadic context, or functional options to avoid 3-line `.WithContext().WithContext()` chains. Source: DiscordSync D1.

- [ ] **"Frozen" registry flag** — prevent runtime mutation of `DefaultRegistry` after first `Classify` call to detect programming errors. Source: DiscordSync D2.

- [ ] **`RegisterClassificationType[T error](family Family)`** — generic type-based registration via `errors.As`. More ergonomic than `RegisterClassifier` closure for the common case. Source: DiscordSync D5.

- [ ] **json/v2 migration strategy** — the root is on `encoding/json/v2` (experimental). Decide: keep until stable, or revert to `encoding/json` until Go makes json/v2 non-experimental. Source: status report 2026-07-09.

- [ ] **v0.8.0 release** — v0.8.0 code is committed at HEAD but has **not been tagged** (latest tag is `v0.7.0`). The CHANGELOG `[Unreleased]` entry is prepared. A deliberate tag-and-release decision is needed.

---

## Completed

Completed items are logged in `CHANGELOG.md` under the version they shipped in. Do not list them here.
