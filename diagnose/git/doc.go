// Package git provides a diagnostic rule that checks git repository state,
// working tree cleanliness, merge conflicts, and remote reachability.
//
// Stability: experimental (v0.x). The API may change between minor versions.
// The classification core (errorfamily root package) is the stable foundation;
// this submodule extends it with git-specific diagnostics.
package git
