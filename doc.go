// Package errorfamily provides structured error classification and handling for Go.
// Every error gets a behavioral Family (retry/no-retry), a machine-readable code,
// human-readable context, and optional diagnostics. Designed for CLI and HTTP boundaries.
//
// Stability: the root package (classification core) is the stable foundation.
// The agent, diagnose, and bridge packages are experimental (v0.x).
package errorfamily
