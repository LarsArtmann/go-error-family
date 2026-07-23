package errorfamilytest_test

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/errorfamilytest"
)

func TestAssertFamily(t *testing.T) {
	errorfamilytest.AssertFamily(t, errorfamily.NewRejection("c", "m"), errorfamily.Rejection)
	errorfamilytest.AssertFamily(t, errorfamily.NewTransient("c", "m"), errorfamily.Transient)
}

func TestAssertFamilyMismatch(t *testing.T) {
	rec := newFailureRecorder(t)
	rec.run(func() {
		errorfamilytest.AssertFamily(rec, errorfamily.NewRejection("c", "m"), errorfamily.Transient)
	})

	if !rec.failed {
		t.Fatal("expected AssertFamily to fail on family mismatch")
	}
}

func TestAssertCode(t *testing.T) {
	errorfamilytest.AssertCode(t, errorfamily.NewConflict("order.conflict", "m"), "order.conflict")
	// Plain error has no code → want "" passes.
	errorfamilytest.AssertCode(t, errors.New("plain"), "")
}

func TestAssertCodeMismatch(t *testing.T) {
	rec := newFailureRecorder(t)
	rec.run(func() {
		errorfamilytest.AssertCode(rec, errorfamily.NewRejection("want.this", "m"), "got.that")
	})

	if !rec.failed {
		t.Fatal("expected AssertCode to fail on code mismatch")
	}
}

func TestAssertRetryable(t *testing.T) {
	errorfamilytest.AssertRetryable(t, errorfamily.NewTransient("c", "m"), true)
	errorfamilytest.AssertRetryable(t, errorfamily.NewRejection("c", "m"), false)
}

func TestAssertRetryableMismatch(t *testing.T) {
	rec := newFailureRecorder(t)
	rec.run(func() {
		errorfamilytest.AssertRetryable(rec, errorfamily.NewTransient("c", "m"), false)
	})

	if !rec.failed {
		t.Fatal("expected AssertRetryable to fail on retryable mismatch")
	}
}

func TestAssertContext(t *testing.T) {
	err := errorfamily.NewRejection("c", "m").WithContext("field", "email")
	errorfamilytest.AssertContext(t, err, "field", "email")
}

func TestAssertContextWrongValue(t *testing.T) {
	err := errorfamily.NewRejection("c", "m").WithContext("field", "email")
	rec := newFailureRecorder(t)
	rec.run(func() {
		errorfamilytest.AssertContext(rec, err, "field", "phone")
	})

	if !rec.failed {
		t.Fatal("expected AssertContext to fail on value mismatch")
	}
}

func TestAssertContextNotContextual(t *testing.T) {
	rec := newFailureRecorder(t)
	rec.run(func() {
		errorfamilytest.AssertContext(rec, errors.New("plain"), "key", "val")
	})

	if !rec.failed {
		t.Fatal("expected AssertContext to fail when error is not Contextual")
	}
}

func TestAssertContextMissing(t *testing.T) {
	err := errorfamily.NewRejection("c", "m").WithContext("field", "email")
	errorfamilytest.AssertContextMissing(t, err, "absent")

	// Pristine sentinel has no context.
	sentinel := errorfamily.NewRejection("sentinel", "m")
	errorfamilytest.AssertContextMissing(t, sentinel, "anything")
}

func TestAssertContextMissingButPresent(t *testing.T) {
	err := errorfamily.NewRejection("c", "m").WithContext("field", "email")
	rec := newFailureRecorder(t)
	rec.run(func() {
		errorfamilytest.AssertContextMissing(rec, err, "field")
	})

	if !rec.failed {
		t.Fatal("expected AssertContextMissing to fail when key is unexpectedly present")
	}
}

func TestAssertExitCode(t *testing.T) {
	errorfamilytest.AssertExitCode(t, errorfamily.NewRejection("c", "m"), 1)
	errorfamilytest.AssertExitCode(t, errorfamily.NewTransient("c", "m"), 75)
	errorfamilytest.AssertExitCode(t, errorfamily.NewRejection("c", "m").WithExitCode(42), 42)
	errorfamilytest.AssertExitCode(t, nil, 0)
}

func TestAssertExitCodeMismatch(t *testing.T) {
	rec := newFailureRecorder(t)
	rec.run(func() {
		errorfamilytest.AssertExitCode(rec, errorfamily.NewRejection("c", "m"), 99)
	})

	if !rec.failed {
		t.Fatal("expected AssertExitCode to fail on mismatch")
	}
}

// --- failureRecorder: intercepts test failures without failing the outer test ---

type fatalSentinel struct{}

type failureRecorder struct {
	testing.TB

	failed bool
}

func newFailureRecorder(tb testing.TB) *failureRecorder {
	tb.Helper()

	return &failureRecorder{TB: tb}
}

func (r *failureRecorder) Error(args ...any)     { r.failed = true }
func (r *failureRecorder) Errorf(string, ...any) { r.failed = true }
func (r *failureRecorder) Fatal(args ...any)     { r.failed = true; panic(fatalSentinel{}) }
func (r *failureRecorder) Fatalf(string, ...any) { r.failed = true; panic(fatalSentinel{}) }
func (r *failureRecorder) Fail()                 { r.failed = true }
func (r *failureRecorder) FailNow()              { r.failed = true; panic(fatalSentinel{}) }
func (r *failureRecorder) Failed() bool          { return r.failed }

func (r *failureRecorder) run(fn func()) {
	defer func() {
		if rv := recover(); rv != nil {
			if _, ok := rv.(fatalSentinel); !ok {
				panic(rv)
			}
		}
	}()

	fn()
}
