package errorfamily

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

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
	cause := errors.New("root cause")
	err := Wrap(cause, Transient, "db.timeout", "database timed out")

	if !errors.Is(err.Cause(), cause) {
		t.Error("Cause() should return the wrapped error")
	}
	if !errors.Is(err.Unwrap(), cause) {
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

func TestErrorWithContextCopyOnWrite(t *testing.T) {
	original := NewRejection("test", "msg").
		WithContext("key1", "val1")

	derived := original.WithContext("key2", "val2")

	if original.ContextValue("key2") != "" {
		t.Error("WithContext should not mutate the original error")
	}
	if original.ContextValue("key1") != "val1" {
		t.Error("original context should be preserved after WithContext on derived")
	}
	if derived.ContextValue("key1") != "val1" {
		t.Error("derived should inherit existing context")
	}
	if derived.ContextValue("key2") != "val2" {
		t.Error("derived should have the new context key")
	}
}

func TestErrorWithCauseCopyOnWrite(t *testing.T) {
	original := NewRejection("test", "msg")
	cause1 := errors.New("first")
	cause2 := errors.New("second")

	original = original.WithCause(cause1)
	derived := original.WithCause(cause2)

	if !errors.Is(original.Cause(), cause1) {
		t.Error("WithCause should not mutate the original error's cause")
	}
	if !errors.Is(derived.Cause(), cause2) {
		t.Error("derived should have the new cause")
	}
}

func TestErrorWithTimestampCopyOnWrite(t *testing.T) {
	original := NewRejection("test", "msg")
	originalTS := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	original = original.WithTimestamp(originalTS)

	newTS := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)
	derived := original.WithTimestamp(newTS)

	if original.Timestamp() != originalTS {
		t.Error("WithTimestamp should not mutate the original error")
	}
	if derived.Timestamp() != newTS {
		t.Error("derived should have the new timestamp")
	}
}

func TestErrorFormat(t *testing.T) {
	err := NewTransient("db.timeout", "connection refused").
		WithContext("host", "localhost")

	if fmt.Sprintf("%s", err) != "connection refused" {
		t.Errorf("%%s = %q", fmt.Sprintf("%s", err))
	}

	compact := fmt.Sprintf("%v", err)
	if !strings.Contains(compact, "[transient:db.timeout]") {
		t.Errorf("%%v = %q", compact)
	}

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

func assertFamily(t *testing.T, err *Error, want Family) {
	t.Helper()
	if err.ErrorFamily() != want {
		t.Errorf("family = %v, want %v", err.ErrorFamily(), want)
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
			assertFamily(t, tt.err, tt.family)
			if tt.err.ErrorCode() != tt.code {
				t.Errorf("code = %q, want %q", tt.err.ErrorCode(), tt.code)
			}
		})
	}
}

func TestWrapConstructors(t *testing.T) {
	cause := errors.New("root")

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
			assertFamily(t, tt.err, tt.family)
			if !errors.Is(tt.err.Unwrap(), cause) {
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
	cause := errors.New("root")
	err := Wrapf(cause, Transient, "code", "failed: %s", "reason")
	if !strings.Contains(err.Message(), "failed: reason") {
		t.Errorf("Wrapf message = %q", err.Message())
	}
}

func TestErrorTimestamp(t *testing.T) {
	before := time.Now().UTC()
	err := NewRejection("test", "msg")
	after := time.Now().UTC()

	ts := err.Timestamp()
	if ts.Before(before) || ts.After(after) {
		t.Errorf("Timestamp() = %v, expected between %v and %v", ts, before, after)
	}
}

func TestErrorFamilyAccessor(t *testing.T) {
	err := NewTransient("test", "msg")
	if err.Family() != Transient {
		t.Errorf("Family() = %v, want Transient", err.Family())
	}
}

func TestErrorAudience(t *testing.T) {
	tests := []struct {
		family   Family
		expected Audience
	}{
		{Rejection, AudienceUser},
		{Conflict, AudienceUser},
		{Transient, AudienceAll},
		{Corruption, AudienceOps},
		{Infrastructure, AudienceOps},
	}
	for _, tt := range tests {
		if got := tt.family.Audience(); got != tt.expected {
			t.Errorf("Family(%d).Audience() = %v, want %v", tt.family, got, tt.expected)
		}
	}
}

func TestErrorWithCauseBuilder(t *testing.T) {
	cause := errors.New("root")
	err := NewRejection("test", "msg").WithCause(cause)

	if !errors.Is(err.Cause(), cause) {
		t.Error("WithCause should set the cause")
	}
	if !errors.Is(err.Unwrap(), cause) {
		t.Error("Unwrap should return cause set by WithCause")
	}
}

func TestErrorIsNonErrorTarget(t *testing.T) {
	err := NewRejection("test", "msg")
	target := errors.New("plain error")
	if errors.Is(err, target) {
		t.Error("errors.Is should not match non-*Error target")
	}
}

func TestErrorFormatVerboseWithCause(t *testing.T) {
	cause := NewTransient("db.timeout", "connection lost")
	err := WrapRejection(cause, "handler.failed", "handler error")

	verbose := fmt.Sprintf("%+v", err)
	if !strings.Contains(verbose, "caused by:") {
		t.Errorf("%%+v with cause should contain 'caused by:': %q", verbose)
	}
	if !strings.Contains(verbose, "db.timeout") {
		t.Errorf("%%+v should show cause chain: %q", verbose)
	}
}

func TestErrorSummaryWithCause(t *testing.T) {
	cause := errors.New("root")
	err := WrapRejection(cause, "test.code", "something failed")

	summary := err.Summary()
	if !strings.Contains(summary, "test.code") {
		t.Errorf("Summary() should contain code: %q", summary)
	}
	if !strings.Contains(summary, "root") {
		t.Errorf("Summary() should contain cause: %q", summary)
	}
}

func TestErrorContextValueMissing(t *testing.T) {
	err := NewRejection("test", "msg")
	if v := err.ContextValue("nonexistent"); v != "" {
		t.Errorf("ContextValue on empty context should return empty string, got %q", v)
	}
}

func TestErrorContextEmptyOrNil(t *testing.T) {
	err := NewRejection("test", "msg")
	ctx := err.ErrorContext()
	if len(ctx) != 0 {
		t.Errorf("ErrorContext() on error without context should be empty, got %v", ctx)
	}
	if err.HasContext("anything") {
		t.Error("HasContext should be false for error without context")
	}
}

func TestErrorChain(t *testing.T) {
	root := errors.New("root cause")
	mid := Wrap(root, Transient, "db.error", "database failed")
	top := fmt.Errorf("handler: %w", mid)

	if Classify(top) != Transient {
		t.Error("Classify should find Family through chain")
	}

	if !errors.Is(top, mid) {
		t.Error("errors.Is should match through chain")
	}

	coded, ok := errors.AsType[Coded](top)
	if !ok || coded.ErrorCode() != "db.error" {
		t.Error("AsType[Coded] should find through chain")
	}
}

func testFamilyProperty[T comparable](t *testing.T, name string, cases []struct {
	family Family
	want   T
}, get func(Family) T,
) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.family.String(), func(t *testing.T) {
			if got := get(tt.family); got != tt.want {
				t.Errorf("%s() = %v, want %v", name, got, tt.want)
			}
		})
	}
}
