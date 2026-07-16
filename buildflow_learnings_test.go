package errorfamily

import (
	"errors"
	"fmt"
	"testing"
)

// --- WrapOnce ---

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

// --- ExitCoder / WithExitCode ---

func TestErrorExitCode(t *testing.T) {
	t.Parallel()

	t.Run("default is zero", func(t *testing.T) {
		t.Parallel()
		err := NewRejection("code", "msg")
		if err.ExitCode() != 0 {
			t.Errorf("ExitCode() = %d, want 0 (default)", err.ExitCode())
		}
	})

	t.Run("custom exit code", func(t *testing.T) {
		t.Parallel()
		err := NewRejection("code", "msg").WithExitCode(42)
		if err.ExitCode() != 42 {
			t.Errorf("ExitCode() = %d, want 42", err.ExitCode())
		}
	})

	t.Run("copy-on-write", func(t *testing.T) {
		t.Parallel()
		original := NewTransient("code", "msg")
		modified := original.WithExitCode(99)

		if modified == original {
			t.Error("WithExitCode should return a new pointer")
		}
		if original.ExitCode() != 0 {
			t.Errorf("original ExitCode = %d, want 0 (unchanged)", original.ExitCode())
		}
	})

	t.Run("preserves context and fields", func(t *testing.T) {
		t.Parallel()
		err := NewTransient("db.timeout", "db timed out").
			WithContext("host", "db1.example.com").
			WithExitCode(7)

		if err.ContextValue("host") != "db1.example.com" {
			t.Error("WithExitCode should preserve context")
		}
		if err.Code() != "db.timeout" {
			t.Error("WithExitCode should preserve code")
		}
		if err.ExitCode() != 7 {
			t.Errorf("ExitCode = %d, want 7", err.ExitCode())
		}
	})

	t.Run("chained WithExitCode preserves exit code across WithContext", func(t *testing.T) {
		t.Parallel()
		err := NewRejection("code", "msg").
			WithExitCode(3).
			WithContext("key", "val")

		if err.ExitCode() != 3 {
			t.Errorf("ExitCode = %d, want 3 after WithContext", err.ExitCode())
		}
	})
}

func TestPackageExitCodeRespectsExitCoder(t *testing.T) {
	t.Parallel()

	t.Run("ExitCoder override wins over family", func(t *testing.T) {
		t.Parallel()
		err := NewTransient("db.timeout", "db timed out").WithExitCode(42)
		if got := ExitCode(err); got != 42 {
			t.Errorf("ExitCode() = %d, want 42 (ExitCoder override)", got)
		}
	})

	t.Run("zero ExitCoder falls back to family", func(t *testing.T) {
		t.Parallel()
		err := NewTransient("db.timeout", "db timed out")
		if got := ExitCode(err); got != 75 {
			t.Errorf("ExitCode() = %d, want 75 (Transient family default)", got)
		}
	})

	t.Run("external ExitCoder type works", func(t *testing.T) {
		t.Parallel()
		err := fmt.Errorf("wrapped: %w", &customExitError{code: 5})
		if got := ExitCode(err); got != 5 {
			t.Errorf("ExitCode() = %d, want 5 (external ExitCoder)", got)
		}
	})

	t.Run("nil returns zero", func(t *testing.T) {
		t.Parallel()
		if got := ExitCode(nil); got != 0 {
			t.Errorf("ExitCode(nil) = %d, want 0", got)
		}
	})
}

func TestHandleErrorDetailedRespectsExitCoder(t *testing.T) {
	t.Parallel()

	err := NewTransient("db.timeout", "db timed out").WithExitCode(42)
	result := HandleErrorDetailed(err)

	if result.ExitCode != 42 {
		t.Errorf("HandleResult.ExitCode = %d, want 42 (ExitCoder override)", result.ExitCode)
	}
}

// --- WithContextAny ---

func TestWithContextAny(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		key   string
		value any
		want  string
	}{
		{"string", "s", "hello", "hello"},
		{"int", "n", 42, "42"},
		{"int64", "n", int64(99), "99"},
		{"uint", "n", uint(7), "7"},
		{"uint64", "n", uint64(123), "123"},
		{"float64", "f", 3.14, "3.14"},
		{"bool_true", "b", true, "true"},
		{"bool_false", "b", false, "false"},
		{"nil", "x", nil, ""},
		{"struct", "o", struct{ X int }{X: 5}, "{5}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := NewRejection("test", "msg").WithContextAny(tc.key, tc.value)
			if got := err.ContextValue(tc.key); got != tc.want {
				t.Errorf("ContextValue(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

func TestWithContextAnyCopyOnWrite(t *testing.T) {
	t.Parallel()

	original := NewRejection("code", "msg")
	modified := original.WithContextAny("count", 42)

	if modified == original {
		t.Error("WithContextAny should return a new pointer")
	}
	if original.HasContext("count") {
		t.Error("original should not have the new context key")
	}
}

// --- safeCauseString / panic recovery ---

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

		// Should contain the prefix but not panic
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

// --- helpers ---

type customExitError struct {
	code int
}

func (e *customExitError) Error() string { return "custom exit error" }
func (e *customExitError) ExitCode() int { return e.code }
