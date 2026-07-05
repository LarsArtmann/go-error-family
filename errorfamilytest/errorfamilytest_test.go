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

func TestAssertCode(t *testing.T) {
	errorfamilytest.AssertCode(t, errorfamily.NewConflict("order.conflict", "m"), "order.conflict")
	// Plain error has no code → want "" passes.
	errorfamilytest.AssertCode(t, errors.New("plain"), "")
}

func TestAssertRetryable(t *testing.T) {
	errorfamilytest.AssertRetryable(t, errorfamily.NewTransient("c", "m"), true)
	errorfamilytest.AssertRetryable(t, errorfamily.NewRejection("c", "m"), false)
}

func TestAssertContext(t *testing.T) {
	err := errorfamily.NewRejection("c", "m").WithContext("field", "email")
	errorfamilytest.AssertContext(t, err, "field", "email")
}

func TestAssertContextMissing(t *testing.T) {
	err := errorfamily.NewRejection("c", "m").WithContext("field", "email")
	errorfamilytest.AssertContextMissing(t, err, "absent")

	// Pristine sentinel has no context.
	sentinel := errorfamily.NewRejection("sentinel", "m")
	errorfamilytest.AssertContextMissing(t, sentinel, "anything")
}
