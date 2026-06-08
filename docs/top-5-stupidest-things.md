# Top 5 Stupidest Things in This Project

> **Status (2026-06-08):** Items 1–4 are RESOLVED. Item 5 is RESOLVED as of 2026-06-08 session (snapshot map before iterating). This file is kept for historical reference.

## 1. ~~`ApplyFixes` is fraud~~ → RESOLVED (v0.2.0)

Removed entirely. The library now proposes fixes; the consumer decides what to do.

## 2. ~~`codeToWhat` / `codeToFix` — magic string matching~~ → RESOLVED (v0.2.0)

Replaced with exact-match `defaultMessages` map + `RegisterTemplate` + family-based fallback. No substring matching.

## 3. ~~`DiagnosticRunner.Run` returns `any`~~ → RESOLVED (v0.3.0)

Replaced with `DiagnosticFinding` struct in root package + `DiagnosticFunc` type. No more `any` return types.

## 4. ~~`SystemSnapshot` — zero consumers~~ → RESOLVED (v0.2.0)

Deleted. 47 lines of dead code removed.

## 5. ~~`lookupRegistered` — O(n) under lock with deadlock risk~~ → RESOLVED (2026-06-08)

Fixed: the map is now snapshotted (copied) before iterating, and the lock is released before calling `errors.Is`. No deadlock possible.
