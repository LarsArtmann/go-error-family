package errorfamily

import (
	"errors"
	"maps"
	"sync"
)

// Classify returns the Family of any error by checking multiple sources:
//
//  1. Multi-error support (errors.Join) — first non-Transient wins
//  2. Classified interface — the error itself declares its family
//  3. Retryable interface — infer from retryability if no family
//  4. Registered sentinels — known third-party errors mapped in init()
//  5. Default — Transient (fail-open for retry)
//
// Returns Rejection for nil errors.
func Classify(err error) Family {
	if err == nil {
		return Rejection
	}

	// 1. Multi-error support (errors.Join).
	// errors.AsType traverses multi-errors and returns the first match,
	// but we want fail-closed behavior: if any sub-error is not retryable,
	// the whole operation should not be retried.
	if u, ok := err.(interface{ Unwrap() []error }); ok {
		for _, sub := range u.Unwrap() {
			family := Classify(sub)
			if family != Transient {
				return family
			}
		}
		return Transient
	}

	// 2. Check for explicit classification.
	if c, ok := errors.AsType[Classified](err); ok {
		return c.ErrorFamily()
	}

	// 3. Check for retryability (infer family).
	if r, ok := errors.AsType[Retryable](err); ok {
		if r.IsRetryable() {
			return Transient
		}
		return Rejection
	}

	// 4. Check registered third-party sentinels.
	if family, ok := lookupRegistered(err); ok {
		return family
	}

	// 5. Default: Transient (fail-open so unknown errors get retried).
	return Transient
}

// IsRetryable reports whether the error is worth retrying.
// Uses Classify() and checks if the result is Transient.
func IsRetryable(err error) bool {
	return Classify(err).IsRetryable()
}

// ExitCode returns the appropriate process exit code for an error.
// Nil errors return 0. Other errors are classified and mapped to BSD sysexits.h codes.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	return Classify(err).ExitCode()
}

// Compose combines multiple errors using errors.Join. The result can be
// passed to Classify to determine the worst Family (first non-Transient wins)
// or to ExitCode for the appropriate exit code.
//
// Returns nil if no errors are provided or all are nil.
func Compose(errs ...error) error {
	return errors.Join(errs...)
}

// RegisterClassification maps a third-party sentinel error to a Family.
// Thread-safe. Call from init() in external packages:
//
//	func init() {
//	    errorfamily.RegisterClassification(sql.ErrConnDone, errorfamily.Transient)
//	}
//
// This is for errors you don't own (stdlib, libraries).
// For your own errors, implement the Classified interface instead.
func RegisterClassification(sentinel error, family Family) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.entries[sentinel] = family
}

// UnregisterClassification removes a previously registered sentinel mapping.
// Thread-safe. No-op if the sentinel has no registered classification.
func UnregisterClassification(sentinel error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	delete(registry.entries, sentinel)
}

// RegisterClassifications registers multiple sentinel-to-Family mappings at once.
// Thread-safe. Call from init() in external packages.
func RegisterClassifications(classifications map[error]Family) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	maps.Copy(registry.entries, classifications)
}

var registry = struct { //nolint:gochecknoglobals // Mutex-protected classification registry, populated via RegisterClassification.
	mu      sync.RWMutex
	entries map[error]Family
}{
	entries: make(map[error]Family),
}

func lookupRegistered(err error) (Family, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	for sentinel, family := range registry.entries {
		if errors.Is(err, sentinel) {
			return family, true
		}
	}

	return Rejection, false
}
