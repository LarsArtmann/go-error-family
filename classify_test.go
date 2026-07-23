package errorfamily

import (
	"errors"
	"fmt"
	"testing"
)

type retryableOnlyError struct {
	retryable bool
}

func (e *retryableOnlyError) Error() string     { return "retryable-only" }
func (e *retryableOnlyError) IsRetryable() bool { return e.retryable }

type externalError struct {
	code    string
	family  Family
	context map[string]string
}

func (e *externalError) Error() string                   { return "external: " + e.code }
func (e *externalError) ErrorCode() string               { return e.code }
func (e *externalError) ErrorFamily() Family             { return e.family }
func (e *externalError) ErrorContext() map[string]string { return e.context }

func TestClassify(t *testing.T) {
	if Classify(nil) != Rejection {
		t.Error("nil should classify as Rejection")
	}

	err := NewTransient("code", "msg")
	if Classify(err) != Transient {
		t.Error("Transient error should classify as Transient")
	}

	if Classify(errors.New("unknown")) != Transient {
		t.Error("unknown error should default to Transient")
	}
}

func TestClassifyWithRegisteredSentinel(t *testing.T) {
	sentinel := errors.New("test.sentinel")

	RegisterClassification(sentinel, Corruption)
	t.Cleanup(func() { UnregisterClassification(sentinel) })

	if Classify(sentinel) != Corruption {
		t.Error("registered sentinel should classify correctly")
	}

	wrapped := fmt.Errorf("wrapper: %w", sentinel)
	if Classify(wrapped) != Corruption {
		t.Error("wrapped sentinel should classify correctly")
	}
}

func TestClassifyWithRetryable(t *testing.T) {
	err := &retryableOnlyError{retryable: true}
	if Classify(err) != Transient {
		t.Error("retryable=true should infer Transient")
	}

	err2 := &retryableOnlyError{retryable: false}
	if Classify(err2) != Rejection {
		t.Error("retryable=false should infer Rejection")
	}
}

func TestClassifyMultiError(t *testing.T) {
	transient := NewTransient("db.timeout", "timed out")
	rejection := NewRejection("validation", "invalid")
	conflict := NewConflict("state", "stale")
	corruption := NewCorruption("decode", "bad payload")
	infrastructure := NewInfrastructure("startup", "nil dep")

	tests := []struct {
		name string
		err  error
		want Family
	}{
		{"single transient", errors.Join(transient), Transient},
		{"single rejection", errors.Join(rejection), Rejection},
		{"single conflict", errors.Join(conflict), Conflict},
		{"transient then rejection", errors.Join(transient, rejection), Rejection},
		{"rejection then transient", errors.Join(rejection, transient), Rejection},
		{"transient then conflict", errors.Join(transient, conflict), Conflict},
		{"all transient", errors.Join(transient, transient), Transient},
		{"with plain error", errors.Join(transient, errors.New("plain")), Transient},
		{"plain then rejection", errors.Join(errors.New("plain"), rejection), Rejection},
		{"nested join", errors.Join(errors.Join(transient), rejection), Rejection},
		{"all plain", errors.Join(errors.New("a"), errors.New("b")), Transient},

		// Severity-ordered: the worst (highest-severity) sub-error wins,
		// independent of argument order.
		{"worst wins: conflict over rejection", errors.Join(rejection, conflict), Conflict},
		{
			"worst wins: infrastructure over conflict",
			errors.Join(conflict, infrastructure),
			Infrastructure,
		},
		{
			"worst wins: corruption over infrastructure",
			errors.Join(infrastructure, corruption),
			Corruption,
		},
		{"order independence: corruption first", errors.Join(corruption, rejection), Corruption},
		{"order independence: corruption last", errors.Join(rejection, corruption), Corruption},
		{"order independence: conflict both orders a", errors.Join(conflict, rejection), Conflict},
		{"order independence: conflict both orders b", errors.Join(rejection, conflict), Conflict},
		{"worst wins all", errors.Join(transient, rejection, conflict, corruption), Corruption},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.err); got != tt.want {
				t.Errorf("Classify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	if IsRetryable(nil) {
		t.Error("nil should not be retryable")
	}

	if !IsRetryable(NewTransient("code", "msg")) {
		t.Error("Transient should be retryable")
	}

	if IsRetryable(NewRejection("code", "msg")) {
		t.Error("Rejection should not be retryable")
	}
}

func TestCode(t *testing.T) {
	if got := Code(nil); got != "" {
		t.Errorf("Code(nil) = %q, want empty", got)
	}

	if got := Code(errors.New("plain")); got != "" {
		t.Errorf("Code(plain error) = %q, want empty", got)
	}

	want := "db.timeout"
	if got := Code(NewTransient(want, "msg")); got != want {
		t.Errorf("Code(classified) = %q, want %q", got, want)
	}
	// Code is preserved through fmt.Errorf wrapping.
	wrapped := fmt.Errorf("call: %w", NewRejection(want, "msg"))
	if got := Code(wrapped); got != want {
		t.Errorf("Code(wrapped) = %q, want %q", got, want)
	}
}

func TestExitCode(t *testing.T) {
	if ExitCode(nil) != 0 {
		t.Error("nil should have exit code 0")
	}

	if ExitCode(NewTransient("code", "msg")) != 75 {
		t.Error("Transient should have exit code 75")
	}

	if ExitCode(NewRejection("code", "msg")) != 1 {
		t.Error("Rejection should have exit code 1")
	}
}

func TestRegisterClassifications(t *testing.T) {
	s1 := errors.New("sentinel.batch.1")
	s2 := errors.New("sentinel.batch.2")

	RegisterClassifications(map[error]Family{
		s1: Conflict,
		s2: Infrastructure,
	})
	t.Cleanup(func() {
		UnregisterClassification(s1)
		UnregisterClassification(s2)
	})

	if Classify(s1) != Conflict {
		t.Error("batch-registered s1 should classify as Conflict")
	}

	if Classify(s2) != Infrastructure {
		t.Error("batch-registered s2 should classify as Infrastructure")
	}
}

func TestErrorImplementsInterfaces(t *testing.T) {
	err := NewRejection("test", "msg")

	var (
		_ Coded      = err
		_ Classified = err
		_ Contextual = err
		_ Retryable  = err
	)
}

func TestExternalTypeImplementsInterfaces(t *testing.T) {
	err := &externalError{
		code:    "ext.code",
		family:  Transient,
		context: map[string]string{"key": "value"},
	}

	coded, ok := errors.AsType[Coded](err)
	if !ok || coded.ErrorCode() != "ext.code" {
		t.Error("external type should satisfy Coded")
	}

	classified, ok := errors.AsType[Classified](err)
	if !ok || classified.ErrorFamily() != Transient {
		t.Error("external type should satisfy Classified")
	}

	contextual, ok := errors.AsType[Contextual](err)
	if !ok || contextual.ErrorContext()["key"] != "value" {
		t.Error("external type should satisfy Contextual")
	}

	if Classify(err) != Transient {
		t.Error("Classify should use Classified interface on external type")
	}
}
