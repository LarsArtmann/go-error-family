# Bridge Patterns announcement — DRAFT (user review gate: M08/F08.4–F08.5)

**Not published.** Target: a linkable GitHub Discussion on `LarsArtmann/go-error-family`
(adaptable to r/golang). Voice: github-voice external-body register. Self-review notes
at the bottom.

---

## Classify and enrich are different jobs — patterns for combining go-error-family with samber/oops

Error libraries tend to promise one of two things: **classification** (is this
retryable? what HTTP status? what exit code?) or **enrichment** (stack traces,
trace IDs, request context). They are different jobs, and most codebases only
need one of them at the library boundary.

I maintain [go-error-family](https://github.com/LarsArtmann/go-error-family) —
a classification library: six families (Transient, Rejection, Conflict,
Infrastructure, Orchestration, Corruption), one `Classify(err)` call, and
behavioral decisions (retry policy, HTTP status, BSD exit code) derived from
the family. Zero dependencies; the domain error carries the classification,
`fmt.Errorf("...: %w", err)` preserves it.

For enrichment I use [samber/oops](https://github.com/samber/oops) in
application code — stack traces, log attributes, trace IDs. The two never
meet inside a library: libraries classify, applications enrich. The seam
between them is the interesting part, so v0.10.1 added a small
[bridge module](https://github.com/LarsArtmann/go-error-family/tree/master/bridge)
and a full walkthrough:
[guides/bridge](https://errorfamily.lars.software/guides/bridge/).

Three patterns cover essentially every combination:

**1. Pass through (most common).** The library returned a classified error,
the application enriches it with oops, and the oops chain still satisfies the
classification interfaces — `Classify`, `IsRetryable`, `HTTPStatus` keep
working through `Unwrap`. No bridge import needed.

**2. AutoWrap (oops-first).** The application raised the error itself via
oops. `bridge.AutoWrap(err)` infers the family from oops metadata — explicit
tags first (`retryable`, `conflict`, `infrastructure`, ...), then a domain
default (`database`/`network` → Transient, `validation`/`auth` → Rejection,
...), then Transient fail-open. One line at the boundary.

**3. Explicit Wrap (application knows best).** `bridge.Wrap(err,
errorfamily.Conflict)` when only the application can know the family. The
wrapped error still exposes the oops context.

Decision guide in one sentence: library errors → pass through; app errors
with good oops tags/domains → `AutoWrap`; app errors where the family is a
domain decision → explicit `Wrap`.

The full flow (classify → enrich → handle) has a runnable reference
implementation with 19 tests: `examples/cmd/bridge/` +
`examples/checkout/` — a checkout domain that classifies in the library
layer, enriches in the application layer, and handles at the CLI boundary.

Honest status: the bridge is new and has no external consumers yet, and
samber/oops adoption is small — this post is the announcement, not a
popularity claim. If you already use oops and want retry/HTTP/exit-code
decisions from the same error, the guide above is the shortest path. Issues
welcome — especially from anyone running the two together in anger.

💘 Generated with Crush

---

## Self-review (F08.3) — checked against the guide content

- [x] Three patterns named exactly as the guide names them (pass through / AutoWrap / explicit Wrap) — no invented terminology
- [x] Zero-consumers honesty stated explicitly ("no external consumers yet") — no overclaiming
- [x] Every claim traces to shipped code: `AutoWrap` inference cascade (tags → domain → Transient) matches `bridge.InferFamily`; pass-through claim matches `ClassifiedError` interface satisfaction (verified in bridge tests)
- [x] "19 tests" verified in the audit table (examples/cmd/bridge)
- [x] Links all real: repo tree, guide URL (deployed domain), oops repo
- [x] "six families ... zero dependencies" matches v0.10.3 reality
- [x] **PUBLISHED 2026-09-28, WITHDRAWN 2026-09-29** — went live as GitHub Discussion #12 (Announcements; repo Discussions enabled for it), then was deleted on reflection: a link-post in an otherwise empty Discussions tab serves no purpose as a "Discussion" (no audience, no replies, content already lives in the guide/README). Discussions disabled again. The text above is the canonical draft; adapt for a channel with real audience (r/golang candidate) when announcing.
