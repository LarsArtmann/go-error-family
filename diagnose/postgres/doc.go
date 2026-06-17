// Package postgres provides a diagnostic rule that checks PostgreSQL availability
// via pg_isready, TCP connectivity, and service status detection.
//
// Stability: experimental (v0.x). The API may change between minor versions.
// The classification core (errorfamily root package) is the stable foundation;
// this submodule extends it with postgres-specific diagnostics.
package postgres
