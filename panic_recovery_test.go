package errorfamily

import (
	"errors"
	"fmt"
	"testing"
)

type panickingError struct{}

func (p *panickingError) Error() string {
	panic("intentional panic from misbehaving error type")
}

func TestSafeCauseString(t *testing.T) {
	t.Parallel()

	t.Run("normal cause", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("disk full")
		got := safeCauseString(cause)
		if got != "disk full" {
			t.Errorf("safeCauseString = %q, want %q", got, "disk full")
		}
	})

	t.Run("panicking cause returns empty", func(t *testing.T) {
		t.Parallel()
		got := safeCauseString(&panickingError{})
		if got != "" {
			t.Errorf("safeCauseString = %q, want empty string", got)
		}
	})
}

func TestErrorPanicRecovery(t *testing.T) {
	t.Parallel()

	t.Run("Error does not propagate cause panic", func(t *testing.T) {
		t.Parallel()
		err := Wrap(&panickingError{}, Transient, "test.panic", "something went wrong")

		var got string
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Error() should not panic, got: %v", r)
				}
			}()
			got = err.Error()
		}()

		if got != "[transient:test.panic] something went wrong" {
			t.Errorf("Error() = %q, want message without cause (panic suppressed)", got)
		}
	})

	t.Run("Summary does not propagate cause panic", func(t *testing.T) {
		t.Parallel()
		err := Wrap(&panickingError{}, Transient, "test.panic", "boom")

		var got string
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Summary() should not panic, got: %v", r)
				}
			}()
			got = err.Summary()
		}()

		if got != "test.panic: boom" {
			t.Errorf("Summary() = %q, want message without cause", got)
		}
	})

	t.Run("formatVerbose does not propagate cause panic", func(t *testing.T) {
		t.Parallel()
		err := Wrap(&panickingError{}, Transient, "test.panic", "boom")

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("formatVerbose should not panic, got: %v", r)
				}
			}()
			_ = fmt.Sprintf("%+v", err)
		}()
	})
}
