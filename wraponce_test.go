package errorfamily

import (
	"errors"
	"fmt"
	"testing"
)

func TestWrapOnce(t *testing.T) {
	t.Parallel()

	t.Run("wraps non-Error", func(t *testing.T) {
		t.Parallel()

		cause := errors.New("disk full")

		err := WrapOnce(cause, Infrastructure, "disk.full", "disk is full")
		if err == nil {
			t.Fatal("WrapOnce should return non-nil for non-nil input")
		}

		if err.Code() != "disk.full" {
			t.Errorf("Code() = %q, want disk.full", err.Code())
		}

		if !errors.Is(err.Unwrap(), cause) {
			t.Error("Unwrap should return the original cause")
		}
	})

	t.Run("returns existing Error unchanged", func(t *testing.T) {
		t.Parallel()

		original := NewTransient("db.timeout", "database timed out")
		result := WrapOnce(original, Infrastructure, "other.code", "other message")

		if result != original {
			t.Error("WrapOnce should return the exact same *Error when already classified")
		}

		if result.Code() != "db.timeout" {
			t.Errorf("Code() = %q, want db.timeout (original)", result.Code())
		}

		if result.ErrorFamily() != Transient {
			t.Errorf("Family = %v, want Transient (original)", result.ErrorFamily())
		}
	})

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		if WrapOnce(nil, Transient, "code", "msg") != nil {
			t.Error("WrapOnce(nil, ...) should return nil")
		}
	})

	t.Run("detects Error in chain", func(t *testing.T) {
		t.Parallel()

		inner := NewRejection("auth.denied", "not authorized")
		outer := fmt.Errorf("operation failed: %w", inner)
		result := WrapOnce(outer, Infrastructure, "other", "msg")

		if result != inner {
			t.Error("WrapOnce should find the *Error in the unwrap chain")
		}
	})
}

func TestWrapOncef(t *testing.T) {
	t.Parallel()

	t.Run("formats message", func(t *testing.T) {
		t.Parallel()

		cause := errors.New("disk full")
		err := WrapOncef(
			cause,
			Infrastructure,
			"disk.full",
			"disk %s is %d%% full",
			"/dev/sda1",
			99,
		)

		if err.Message() != "disk /dev/sda1 is 99% full" {
			t.Errorf("Message = %q", err.Message())
		}
	})

	t.Run("returns existing unchanged", func(t *testing.T) {
		t.Parallel()

		original := NewTransient("db.timeout", "database timed out")
		result := WrapOncef(original, Infrastructure, "other", "formatted %s", "msg")

		if result != original {
			t.Error("WrapOncef should return the same *Error when already classified")
		}
	})

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		if WrapOncef(nil, Transient, "code", "msg %d", 1) != nil {
			t.Error("WrapOncef(nil, ...) should return nil")
		}
	})
}
