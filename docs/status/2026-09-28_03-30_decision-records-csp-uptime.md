# Decision Records — Website CSP & Uptime Monitoring (M18)

**Date:** 2026-09-28 · **Status:** both DECIDED (declined, with revisit triggers)

## DR-1: Content-Security-Policy for the website — DECLINED (for now)

**Context.** `errorfamily.lars.software` is a fully static Astro/Starlight docs
site on Firebase Hosting: no backend, no user input, no user-generated content,
self-hosted search (pagefind), no third-party runtime scripts. Starlight renders
inline `<script>`/`<style>` blocks, so a meaningful CSP needs per-build
nonces/hashes and breaks on Astro upgrades (maintenance cost with no external
enforcer — Firebase Hosting custom headers carry no report destination by
default).

**Decision.** Formally decline a CSP today. The XSS surface of a static site
with no user input is the build supply chain — which is now guarded
structurally: `pnpm install --frozen-lockfile`, `minimumReleaseAgeStrict: true`,
`trustLockfile` verification ("Lockfile passes supply-chain policies"), and the
`website-check` canary (typescript pin, astro check, astro build) on every
`website/**` change.

**Revisit trigger.** The site gains user input, query-parameter-rendered
content, third-party runtime scripts, or an API boundary — any of these makes
CSP the correct next control.

## DR-2: External uptime monitoring for the website — DECLINED

**Context.** The site is static content behind Firebase Hosting's CDN
(multi-region by construction). The realistic failure modes are (a) a bad
deploy — already gated by `website-check` + `website-deploy` (both must pass
before publish) and (b) a platform outage — visible on Google's public Firebase
status page. An external monitor would add an account, another integration, and
alert noise for a failure class (single-region app server down) this site does
not have.

**Decision.** Decline a dedicated uptime monitor. The deploy pipeline's
build+check gates are the practical "is the site healthy" signal after every
change; Firebase's status page covers platform outages.

**Revisit trigger.** Dynamic features (search API, forms, redirects with
backend logic) land on the domain, or the site becomes commercially load-bearing.

---
~~Both decisions close the corresponding ROADMAP futures items.~~ Verified 2026-09-29 (docs-health pass): neither CSP nor an uptime monitor appears anywhere in ROADMAP.md — both closed here. No code changes.
