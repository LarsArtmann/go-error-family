package errorfamilytest_test

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/errorfamilytest"
)

// The assertion helpers mirror httptest: they keep the testing package out of
// the production module while giving consumers one-line assertions in their
// own tests. Each example shows the call shape; the placeholder *testing.T is
// whatever your test receives.

//nolint:testableexamples // documents call shape; the helpers need a live testing.TB, which an example cannot provide
func ExampleAssertFamily() {
	var t *testing.T // your test's *testing.T

	err := errorfamily.NewConflict("game.already_finished", "move after game over")

	errorfamilytest.AssertFamily(t, err, errorfamily.Conflict)
}

//nolint:testableexamples // documents call shape; the helpers need a live testing.TB, which an example cannot provide
func ExampleAssertCode() {
	var t *testing.T // your test's *testing.T

	err := errorfamily.NewRejection("file.not_found", "config missing")

	errorfamilytest.AssertCode(t, err, "file.not_found")
}

//nolint:testableexamples // documents call shape; the helpers need a live testing.TB, which an example cannot provide
func ExampleAssertRetryable() {
	var t *testing.T // your test's *testing.T

	err := errorfamily.NewTransient("db.timeout", "query took too long")

	errorfamilytest.AssertRetryable(t, err, true)
}

//nolint:testableexamples // documents call shape; the helpers need a live testing.TB, which an example cannot provide
func ExampleAssertContext() {
	var t *testing.T // your test's *testing.T

	err := errorfamily.NewTransient("db.timeout", "query took too long").
		WithContext("host", "localhost")

	errorfamilytest.AssertContext(t, err, "host", "localhost")
}

//nolint:testableexamples // documents call shape; the helpers need a live testing.TB, which an example cannot provide
func ExampleAssertContextMissing() {
	var t *testing.T // your test's *testing.T

	err := errorfamily.NewTransient("db.timeout", "query took too long")

	errorfamilytest.AssertContextMissing(t, err, "host")
}

//nolint:testableexamples // documents call shape; the helpers need a live testing.TB, which an example cannot provide
func ExampleAssertExitCode() {
	var t *testing.T // your test's *testing.T

	err := errorfamily.NewRejection("file.not_found", "config missing")

	errorfamilytest.AssertExitCode(t, err, 1)
}

//nolint:testableexamples // documents call shape; the helpers need a live testing.TB, which an example cannot provide
func ExampleAssertHTTPStatus() {
	var t *testing.T // your test's *testing.T

	err := errorfamily.NewRejection("file.not_found", "config missing")

	errorfamilytest.AssertHTTPStatus(t, err, 400)
}
