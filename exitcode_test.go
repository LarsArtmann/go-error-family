package errorfamily

import (
	"fmt"
	"strings"
	"testing"
)

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

func TestHandleErrorRespectsExitCoder(t *testing.T) {
	t.Parallel()

	err := NewTransient("db.timeout", "db timed out").WithExitCode(42)
	got := HandleErrorWithConfig(err, HandleConfig{Output: &strings.Builder{}})

	if got != 42 {
		t.Errorf("HandleErrorWithConfig exit code = %d, want 42 (ExitCoder override)", got)
	}

	plain := NewTransient("db.timeout", "db timed out")
	defaultCode := HandleErrorWithConfig(plain, HandleConfig{Output: &strings.Builder{}})
	if defaultCode != 75 {
		t.Errorf("default exit code = %d, want 75 (Transient family default)", defaultCode)
	}
}

func TestFormatVerboseShowsExitCode(t *testing.T) {
	t.Parallel()

	err := NewTransient("db.timeout", "db timed out").WithExitCode(42)
	output := fmt.Sprintf("%+v", err)

	if !strings.Contains(output, "exit_code: 42") {
		t.Errorf("formatVerbose output should contain exit_code: 42, got:\n%s", output)
	}

	plain := NewRejection("code", "msg")
	plainOutput := fmt.Sprintf("%+v", plain)
	if strings.Contains(plainOutput, "exit_code") {
		t.Errorf("formatVerbose should NOT show exit_code when zero, got:\n%s", plainOutput)
	}
}

type customExitError struct {
	code int
}

func (e *customExitError) Error() string { return "custom exit error" }
func (e *customExitError) ExitCode() int { return e.code }
