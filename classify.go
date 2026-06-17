package errorfamily

// Classify returns the Family of any error by checking multiple sources:
//
//  1. Multi-error support (errors.Join) — first non-Transient wins
//  2. Classified interface — the error itself declares its family
//  3. Retryable interface — infer from retryability if no family
//  4. Registered sentinels — known third-party errors mapped via RegisterClassification
//  5. Default — Transient (fail-open for retry)
//
// Returns Rejection for nil errors.
//
// This function delegates to [DefaultRegistry]. For scoped or test-isolated
// classification, construct a [Registry] via [NewRegistry] and use its
// Classify method.
func Classify(err error) Family {
	return DefaultRegistry.Classify(err)
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

// RegisterClassification maps a third-party sentinel error to a Family.
// Thread-safe. Call from init() in external packages:
//
//	func init() {
//	    errorfamily.RegisterClassification(sql.ErrConnDone, errorfamily.Transient)
//	}
//
// This is for errors you don't own (stdlib, libraries).
// For your own errors, implement the Classified interface instead.
//
// Delegates to [DefaultRegistry]. For scoped registration, use
// [Registry.RegisterClassification] on a custom Registry.
func RegisterClassification(sentinel error, family Family) {
	DefaultRegistry.RegisterClassification(sentinel, family)
}

// UnregisterClassification removes a previously registered sentinel mapping.
// Thread-safe. No-op if the sentinel has no registered classification.
func UnregisterClassification(sentinel error) {
	DefaultRegistry.UnregisterClassification(sentinel)
}

// RegisterClassifications registers multiple sentinel-to-Family mappings at once.
// Thread-safe. Call from init() in external packages.
func RegisterClassifications(classifications map[error]Family) {
	DefaultRegistry.RegisterClassifications(classifications)
}
