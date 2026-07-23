# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and
traceable to its source. When an item ships, remove it here and record it in
`CHANGELOG.md` under the version it shipped in.

**Last updated:** 2026-07-23

---

## Active

### High Priority

- [ ] **Add CI gate: `GOWORK=off go list -m all` per module** — prevents recurrence of the phantom-`replace`/`require` bug that broke v0.6.0 for consumers. `go.work` masked it locally; only `go list -m all` with `GOWORK=off` catches unresolvable module-graph edges. Source: status report 2026-07-05_20-26.

- [ ] **Add CI consumer-simulation job** — a throwaway module that does `go get github.com/larsartmann/go-error-family@<tag>; go list -m all`. The only honest proof a release works downstream. Source: status report 2026-07-05_20-26.

- [ ] **Audit website docs for stale API references** — `website/src/content/docs/` likely contains pre-v0.8.0 references (`SuggestedFix` on DiagnosticResult, `Diagnose: true` in HandleConfig, `context.go` ghost, missing `GOEXPERIMENT`). The docs-health audit (2026-07-13) flagged this as Critical for public trust but it was never addressed. Source: status report 2026-07-13_22-15 section d.1.

- [ ] **Add mutators section to website `api-reference.mdx`** — `WithContext`, `WithContextMap`, `WithContextf`, `WithCause`, `WithTimestamp`, `WithContextAny`, `WithExitCode` are entirely absent from the website API reference. The v0.8.0 APIs (`WithContextAny`, `WithExitCode`) should be documented alongside the existing ones. Source: status report 2026-07-16_06-30 section c.1.

- [ ] **Rebuild and deploy website** — the live site at `errorfamily.lars.software` is stale. API changes from v0.8.0 (ExitCoder, WrapOnce, WithContextAny) have not been deployed. Source: status report 2026-07-16_06-30 section c.2.

### Medium Priority

- [ ] **Add `New*` vs `Wrap*` guidance to SKILL.md** — one paragraph: "Use `New*` when creating from scratch. Use `Wrap*` when you have an underlying error to chain." Source: DiscordSync feedback D9.

- [ ] **Add `errkit` consumer pattern example to SKILL.md** — show the nil-safe wrapper pattern every non-trivial consumer builds. Source: DiscordSync feedback D10.

- [ ] **Add `writeHTTPError` error-branch test** — inject a failing `http.ResponseWriter` to cover the json-encode error path. Currently uncovered. Source: status report 2026-07-05_03-21.

- [ ] **Document or validate negative exit codes** — `WithExitCode(-1)` is accepted by the API. `os.Exit(-1)` wraps to 255 on POSIX. Either reject negative values or document the behavior. Source: status report 2026-07-16_06-30 section c.6.

### Low Priority

- [ ] **Refactor `contextValueToString` to eliminate `//nolint:cyclop`** — split into `scalarToString` + `complexToString` dispatch instead of suppressing the linter on a 12-case type switch. Source: status report 2026-07-16_06-30 section d.1.

- [ ] **Add `time.Duration` case to `contextValueToString`** — extremely common context value type (timeouts, retry intervals). Currently renders via `fmt.Sprint` as `5s`, which works but is inconsistent with the explicit `time.Time` RFC3339 case. Source: status report 2026-07-16_06-30 section c.7.

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
