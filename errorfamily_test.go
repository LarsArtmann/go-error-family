package errorfamily

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestFamilyString(t *testing.T) {
	tests := []struct {
		family Family
		want   string
	}{
		{Rejection, "rejection"},
		{Conflict, "conflict"},
		{Transient, "transient"},
		{Corruption, "corruption"},
		{Infrastructure, "infrastructure"},
		{Family(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.family.String(); got != tt.want {
			t.Errorf("Family(%d).String() = %q, want %q", tt.family, got, tt.want)
		}
	}
}

func TestParseFamily(t *testing.T) {
	tests := []struct {
		input string
		want  Family
	}{
		{"rejection", Rejection},
		{"REJECTION", Rejection},
		{"conflict", Conflict},
		{"transient", Transient},
		{"corruption", Corruption},
		{"infrastructure", Infrastructure},
		{"unknown", Transient}, // default
		{"garbage", Transient}, // default
	}
	for _, tt := range tests {
		if got := ParseFamily(tt.input); got != tt.want {
			t.Errorf("ParseFamily(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFamilyIsRetryable(t *testing.T) {
	if Rejection.IsRetryable() {
		t.Error("Rejection should not be retryable")
	}
	if Conflict.IsRetryable() {
		t.Error("Conflict should not be retryable")
	}
	if !Transient.IsRetryable() {
		t.Error("Transient should be retryable")
	}
	if Corruption.IsRetryable() {
		t.Error("Corruption should not be retryable")
	}
	if Infrastructure.IsRetryable() {
		t.Error("Infrastructure should not be retryable")
	}
}

func TestFamilyExitCode(t *testing.T) {
	tests := []struct {
		family Family
		want   int
	}{
		{Rejection, 1},
		{Conflict, 1},
		{Transient, 75},
		{Corruption, 65},
		{Infrastructure, 69},
		{Family(99), 70},
	}
	for _, tt := range tests {
		if got := tt.family.ExitCode(); got != tt.want {
			t.Errorf("Family(%d).ExitCode() = %d, want %d", tt.family, got, tt.want)
		}
	}
}

func TestFamilyTone(t *testing.T) {
	if Rejection.Tone() != ToneInstructional {
		t.Error("Rejection should have instructional tone")
	}
	if Transient.Tone() != ToneReassuring {
		t.Error("Transient should have reassuring tone")
	}
}

func TestErrorBasic(t *testing.T) {
	err := NewRejection("test.not_found", "something was not found")

	if err.Error() != "[rejection:test.not_found] something was not found" {
		t.Errorf("Error() = %q", err.Error())
	}
	if err.ErrorCode() != "test.not_found" {
		t.Errorf("ErrorCode() = %q", err.ErrorCode())
	}
	if err.ErrorFamily() != Rejection {
		t.Errorf("ErrorFamily() = %v", err.ErrorFamily())
	}
	if err.Message() != "something was not found" {
		t.Errorf("Message() = %q", err.Message())
	}
	if err.Code() != "test.not_found" {
		t.Errorf("Code() = %q", err.Code())
	}
	if err.IsRetryable() {
		t.Error("Rejection should not be retryable")
	}
}

func TestErrorWithCause(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := Wrap(cause, Transient, "db.timeout", "database timed out")

	if err.Cause() != cause {
		t.Error("Cause() should return the wrapped error")
	}
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the wrapped error")
	}
	if !strings.Contains(err.Error(), "root cause") {
		t.Errorf("Error() should contain cause: %q", err.Error())
	}
}

func TestErrorIs(t *testing.T) {
	err1 := NewRejection("test.code", "msg1")
	err2 := NewRejection("test.code", "msg2")
	err3 := NewConflict("test.code", "msg1")
	err4 := NewRejection("other.code", "msg1")

	if !errors.Is(err1, err2) {
		t.Error("errors.Is should match same code+family")
	}
	if errors.Is(err1, err3) {
		t.Error("errors.Is should not match different family")
	}
	if errors.Is(err1, err4) {
		t.Error("errors.Is should not match different code")
	}
}

func TestErrorWithContext(t *testing.T) {
	err := NewRejection("file.not_found", "config missing").
		WithContext("path", "/etc/app/config.yaml").
		WithContext("format", "yaml")

	ctx := err.ErrorContext()
	if ctx["path"] != "/etc/app/config.yaml" {
		t.Errorf("context path = %q", ctx["path"])
	}
	if ctx["format"] != "yaml" {
		t.Errorf("context format = %q", ctx["format"])
	}

	if !err.HasContext("path") {
		t.Error("HasContext(path) should be true")
	}
	if err.HasContext("nonexistent") {
		t.Error("HasContext(nonexistent) should be false")
	}
	if err.ContextValue("path") != "/etc/app/config.yaml" {
		t.Errorf("ContextValue(path) = %q", err.ContextValue("path"))
	}
}

func TestErrorContextIsolation(t *testing.T) {
	err := NewRejection("test", "msg").WithContext("key", "value")
	ctx := err.ErrorContext()
	ctx["key"] = "mutated"

	if err.ContextValue("key") != "value" {
		t.Error("ErrorContext() should return a copy, not a reference")
	}
}

func TestErrorFormat(t *testing.T) {
	err := NewTransient("db.timeout", "connection refused").
		WithContext("host", "localhost")

	// %s — message only
	if fmt.Sprintf("%s", err) != "connection refused" {
		t.Errorf("%%s = %q", fmt.Sprintf("%s", err))
	}

	// %v — compact
	compact := fmt.Sprintf("%v", err)
	if !strings.Contains(compact, "[transient:db.timeout]") {
		t.Errorf("%%v = %q", compact)
	}

	// %+v — verbose
	verbose := fmt.Sprintf("%+v", err)
	if !strings.Contains(verbose, "context:") {
		t.Errorf("%%+v should contain context: %q", verbose)
	}
	if !strings.Contains(verbose, "host: localhost") {
		t.Errorf("%%+v should contain host: %q", verbose)
	}
}

func TestErrorSummary(t *testing.T) {
	err := NewRejection("file.not_found", "config missing")
	if err.Summary() != "file.not_found: config missing" {
		t.Errorf("Summary() = %q", err.Summary())
	}
}

func TestErrorMatchesContext(t *testing.T) {
	err := NewRejection("file.not_found", "msg").
		WithContext("path", "/etc/app/config.yaml")

	if !err.MatchesContext("path", "host") {
		t.Error("MatchesContext should find 'path'")
	}
	if err.MatchesContext("host", "port") {
		t.Error("MatchesContext should not find 'host'")
	}
	if !err.MatchesContextValue("config") {
		t.Error("MatchesContextValue should find 'config' in path value")
	}
	if err.MatchesContextValue("nonexistent_xyz") {
		t.Error("MatchesContextValue should not find random string")
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name   string
		err    *Error
		family Family
		code   string
	}{
		{"NewRejection", NewRejection("r", "msg"), Rejection, "r"},
		{"NewConflict", NewConflict("c", "msg"), Conflict, "c"},
		{"NewTransient", NewTransient("t", "msg"), Transient, "t"},
		{"NewCorruption", NewCorruption("cr", "msg"), Corruption, "cr"},
		{"NewInfrastructure", NewInfrastructure("i", "msg"), Infrastructure, "i"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.ErrorFamily() != tt.family {
				t.Errorf("family = %v, want %v", tt.err.ErrorFamily(), tt.family)
			}
			if tt.err.ErrorCode() != tt.code {
				t.Errorf("code = %q, want %q", tt.err.ErrorCode(), tt.code)
			}
		})
	}
}

func TestWrapConstructors(t *testing.T) {
	cause := fmt.Errorf("root")

	tests := []struct {
		name   string
		err    *Error
		family Family
	}{
		{"WrapRejection", WrapRejection(cause, "r", "msg"), Rejection},
		{"WrapConflict", WrapConflict(cause, "c", "msg"), Conflict},
		{"WrapTransient", WrapTransient(cause, "t", "msg"), Transient},
		{"WrapCorruption", WrapCorruption(cause, "cr", "msg"), Corruption},
		{"WrapInfrastructure", WrapInfrastructure(cause, "i", "msg"), Infrastructure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.ErrorFamily() != tt.family {
				t.Errorf("family = %v, want %v", tt.err.ErrorFamily(), tt.family)
			}
			if tt.err.Unwrap() != cause {
				t.Error("should wrap cause")
			}
		})
	}
}

func TestWrapNil(t *testing.T) {
	if result := Wrap(nil, Transient, "code", "msg"); result != nil {
		t.Error("Wrap(nil, ...) should return nil")
	}
}

func TestNewf(t *testing.T) {
	err := Newf(Rejection, "test.code", "value: %d", 42)
	if err.Message() != "value: 42" {
		t.Errorf("Newf message = %q", err.Message())
	}
}

func TestWrapf(t *testing.T) {
	cause := fmt.Errorf("root")
	err := Wrapf(cause, Transient, "code", "failed: %s", "reason")
	if !strings.Contains(err.Message(), "failed: reason") {
		t.Errorf("Wrapf message = %q", err.Message())
	}
}

func TestClassify(t *testing.T) {
	// nil → Rejection
	if Classify(nil) != Rejection {
		t.Error("nil should classify as Rejection")
	}

	// Error with Family → uses it
	err := NewTransient("code", "msg")
	if Classify(err) != Transient {
		t.Error("Transient error should classify as Transient")
	}

	// Plain error → default Transient
	if Classify(fmt.Errorf("unknown")) != Transient {
		t.Error("unknown error should default to Transient")
	}
}

func TestClassifyWithRegisteredSentinel(t *testing.T) {
	sentinel := fmt.Errorf("test.sentinel")

	RegisterClassification(sentinel, Corruption)

	if Classify(sentinel) != Corruption {
		t.Error("registered sentinel should classify correctly")
	}

	// Wrapped sentinel should also work.
	wrapped := fmt.Errorf("wrapper: %w", sentinel)
	if Classify(wrapped) != Corruption {
		t.Error("wrapped sentinel should classify correctly")
	}
}

func TestClassifyWithRetryable(t *testing.T) {
	// Custom error that implements Retryable but not Classified.
	err := &retryableOnlyError{retryable: true}
	if Classify(err) != Transient {
		t.Error("retryable=true should infer Transient")
	}

	err2 := &retryableOnlyError{retryable: false}
	if Classify(err2) != Rejection {
		t.Error("retryable=false should infer Rejection")
	}
}

type retryableOnlyError struct {
	retryable bool
}

func (e *retryableOnlyError) Error() string     { return "retryable-only" }
func (e *retryableOnlyError) IsRetryable() bool { return e.retryable }

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
	s1 := fmt.Errorf("sentinel.batch.1")
	s2 := fmt.Errorf("sentinel.batch.2")

	RegisterClassifications(map[error]Family{
		s1: Conflict,
		s2: Infrastructure,
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

	var _ Coded = err
	var _ Classified = err
	var _ Contextual = err
	var _ Retryable = err
}

func TestExternalTypeImplementsInterfaces(t *testing.T) {
	// Verify that an external type can implement our interfaces
	// without depending on our Error struct.
	err := &externalError{
		code:    "ext.code",
		family:  Transient,
		context: map[string]string{"key": "value"},
	}

	// errors.AsType should find our interfaces
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

	// Classify should work through the interface
	if Classify(err) != Transient {
		t.Error("Classify should use Classified interface on external type")
	}
}

type externalError struct {
	code    string
	family  Family
	context map[string]string
}

func (e *externalError) Error() string                    { return "external: " + e.code }
func (e *externalError) ErrorCode() string               { return e.code }
func (e *externalError) ErrorFamily() Family             { return e.family }
func (e *externalError) ErrorContext() map[string]string { return e.context }

func TestErrorChain(t *testing.T) {
	root := fmt.Errorf("root cause")
	mid := Wrap(root, Transient, "db.error", "database failed")
	top := fmt.Errorf("handler: %w", mid)

	// Classify should traverse the chain.
	if Classify(top) != Transient {
		t.Error("Classify should find Family through chain")
	}

	// errors.Is should match mid by code+family.
	if !errors.Is(top, mid) {
		t.Error("errors.Is should match through chain")
	}

	// AsType should find Coded through chain.
	coded, ok := errors.AsType[Coded](top)
	if !ok || coded.ErrorCode() != "db.error" {
		t.Error("AsType[Coded] should find through chain")
	}
}
