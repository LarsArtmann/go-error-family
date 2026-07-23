// Package errorfamilytest provides assertion helpers for testing code that
// returns go-error-family classified errors.
//
// Import as:
//
//	import (
//	    errorfamily "github.com/larsartmann/go-error-family"
//	    "github.com/larsartmann/go-error-family/errorfamilytest"
//	)
//
//	func TestHandler(t *testing.T) {
//	    err := doSomething()
//	    errorfamilytest.AssertFamily(t, err, errorfamily.Rejection)
//	    errorfamilytest.AssertCode(t, err, "user.not_found")
//	}
//
// These helpers live in a separate package (mirroring net/http/httptest) so the
// main package never imports "testing" in production code.
package errorfamilytest

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// AssertFamily asserts that err classifies to the expected [errorfamily.Family].
// It uses [errorfamily.Classify], so classification rules (sentinels, classifiers,
// the Classified interface) all apply. A nil err classifies as Rejection.
func AssertFamily(tb testing.TB, err error, want errorfamily.Family) {
	tb.Helper()

	if got := errorfamily.Classify(err); got != want {
		tb.Errorf("Classify(%v) = %v, want %v", err, got, want)
	}
}

// AssertCode asserts that err carries the expected machine-readable code,
// extracted via [errorfamily.Code] (walking the unwrap chain for the Coded
// interface). want may be "" to assert the error has no code.
func AssertCode(tb testing.TB, err error, want string) {
	tb.Helper()

	if got := errorfamily.Code(err); got != want {
		tb.Errorf("Code(%v) = %q, want %q", err, got, want)
	}
}

// AssertRetryable asserts the retryability of err via [errorfamily.IsRetryable].
func AssertRetryable(tb testing.TB, err error, want bool) {
	tb.Helper()

	if got := errorfamily.IsRetryable(err); got != want {
		tb.Errorf("IsRetryable(%v) = %v, want %v", err, got, want)
	}
}

// AssertContext asserts that err carries the expected value for a context key,
// extracted via the [errorfamily.Contextual] interface. It fails if the error
// does not expose context at all or lacks the key.
func AssertContext(tb testing.TB, err error, key, want string) {
	tb.Helper()

	ctx, ok := errors.AsType[errorfamily.Contextual](err)
	if !ok {
		tb.Fatalf("AssertContext: %v does not implement errorfamily.Contextual", err)
	}

	if got := ctx.ErrorContext()[key]; got != want {
		tb.Errorf("ErrorContext(%v)[%q] = %q, want %q", err, key, got, want)
	}
}

// AssertContextMissing asserts that the error's context does NOT contain key.
// Useful to confirm a sentinel stays pristine (no leaked context).
func AssertContextMissing(tb testing.TB, err error, key string) {
	tb.Helper()

	ctx, ok := errors.AsType[errorfamily.Contextual](err)
	if !ok {
		return
	}

	if _, ok := ctx.ErrorContext()[key]; ok {
		tb.Errorf("ErrorContext(%v) unexpectedly contains key %q", err, key)
	}
}

// AssertExitCode asserts that err resolves to the expected process exit code
// via [errorfamily.ExitCode]. This checks the ExitCoder interface first (for
// per-error overrides set via WithExitCode), then falls back to the family
// default. A nil err yields exit code 0.
func AssertExitCode(tb testing.TB, err error, want int) {
	tb.Helper()

	if got := errorfamily.ExitCode(err); got != want {
		tb.Errorf("ExitCode(%v) = %d, want %d", err, got, want)
	}
}
