# TODO List

Short- and mid-term actionable improvement tasks. Each item is bounded and
traceable to its source. When an item ships, remove it here and update
`FEATURES.md` + `CHANGELOG.md`.

**Last updated:** 2026-07-16

---

## Active

### High Priority

- [ ] **Add godoc for surprising behaviors on the types themselves** — `Classify(nil)`→Rejection, `Wrap(nil)`→nil, `errors.Is` code+family matching, and `{key}` substitution belong in godoc on `Classify`/`Wrap`/`Error.Is`/`MessageTemplate`, not just SKILL.md. Consumers look at godoc, not SKILL.md. Sources: SwettySwipper S1-S4, DiscordSync feedback.
  - `classify.go`: add "Classify(nil) returns Rejection" to `Classify` godoc
  - `error.go`: add `errors.Is` matching example to `Error.Is` godoc
  - `constructors.go`: add "Returns nil if err is nil — use New\* for errors without a cause" to `Wrap` godoc
  - `handle.go`: add `{key}` substitution note to `MessageTemplate` godoc

- [ ] **Add CI gate: `GOWORK=off go list -m all` per module** — prevents recurrence of the phantom-`replace`/`require` bug that broke v0.6.0 for consumers. `go.work` masked it locally. Source: status report 2026-07-05_20-26.

- [ ] **Add CI consumer-simulation job** — a throwaway module that does `go get github.com/larsartmann/go-error-family@<tag>; go list -m all`. The only honest proof a release works downstream. Source: status report 2026-07-05_20-26.

### Medium Priority

- [ ] **Add `New*` vs `Wrap*` guidance to SKILL.md** — one paragraph: "Use `New*` when creating from scratch. Use `Wrap*` when you have an underlying error to chain." Source: DiscordSync D9.

- [ ] **Add `RegisterClassifications` map variant to SKILL.md examples** — only the singular `RegisterClassification` is shown; the batch map variant is more ergonomic. Source: DiscordSync D11.

- [ ] **Clarify `RegisterTemplate` is on `DefaultRegistry`** in SKILL.md — consumers are confused whether templates are global or per-registry. They ARE on DefaultRegistry; the new `TemplateForCode` proves it. Source: DiscordSync D7.

- [ ] **Add batch/partial-success canonical example to SKILL.md** — code block showing collect→join→classify pattern. (Already partially done in SKILL.md "Partial Success" section — verify it's complete.) Source: DiscordSync D8.

- [ ] **Add `errkit` consumer pattern example to SKILL.md** — show the nil-safe wrapper pattern every non-trivial consumer builds. Source: DiscordSync D10.

- [ ] **Add "skip diagnose/ unless infrastructure debugging" note to SKILL.md** — one-liner: "Skip diagnose/ unless your failure modes include infrastructure health issues (database down, disk full)." Source: DiscordSync D6.

### Low Priority

- [ ] **Add `RegisterClassifier` (singular) test coverage** — the singular variant is 0% covered (only the plural `RegisterClassifiers` is tested). Source: status report 2026-07-05_03-21.

- [ ] **Add `writeHTTPError` error-branch test** — inject a failing `http.ResponseWriter` to cover the json-encode error path (currently 90.9%). Source: status report 2026-07-05_03-21.

- [ ] **Update `examples/cmd/http`** to use `HTTPHandler` instead of a custom handler — now that `HTTPHandler` exists. Source: status report 2026-07-05_03-21.

---

## Design Decisions Needed

These require a product decision before implementation. They are NOT actionable tasks yet.

- [ ] **Per-error HTTP status override (`Error.WithHTTPStatus(code int)`)** — SwettySwipper's #1 request. `battle.not_found` should be 404, not the family default 400. Core tension: is HTTP status a classification concern (library) or a presentation concern (HTTP handler)? Source: SwettySwipper S5, status report 2026-07-05_03-21.

- [ ] **`Classify(nil)` semantics** — DiscordSync argues Rejection is inconsistent with fail-open. Options: keep Rejection (current), change to Infrastructure (programming error), or change to Transient (fail-open). Changing is breaking. Source: DiscordSync D4.

- [ ] **Constructor context ergonomics** — builder pattern, variadic context, or functional options to avoid 3-line `.WithContext().WithContext()` chains. Source: DiscordSync D1.

- [ ] **"Frozen" registry flag** — prevent runtime mutation of `DefaultRegistry` after first `Classify` call to detect programming errors. Source: DiscordSync D2.

- [ ] **`RegisterClassificationType[T error](family Family)`** — generic type-based registration via `errors.As`. More ergonomic than `RegisterClassifier` closure for the common case. Source: DiscordSync D5.

- [ ] **json/v2 migration strategy** — the root is on `encoding/json/v2` (experimental). Decide: keep until stable, or revert to `encoding/json` until Go makes json/v2 non-experimental. Source: status report 2026-07-09.

---

## Completed

Completed items are logged in `CHANGELOG.md` under the version they shipped in. Do not list them here.
