package bridge

import (
	"errors"
	"fmt"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/samber/oops"
)

func TestAutoWrap_InfersAndWraps(t *testing.T) {
	err := oops.In("database").With("host", "db1").Errorf("timeout")
	classified := AutoWrap(err)

	family := errorfamily.Classify(classified)
	if family != errorfamily.Transient {
		t.Errorf("AutoWrap(database domain) classify = %v, want Transient", family)
	}
	if classified.Family() != errorfamily.Transient {
		t.Errorf("AutoWrap(database domain).Family() = %v, want Transient", classified.Family())
	}
	if !errorfamily.IsRetryable(classified) {
		t.Error("AutoWrap(database domain) should be retryable")
	}
}

func TestAutoWrap_ValidationDomainIsRejection(t *testing.T) {
	err := oops.In("validation").Errorf("bad input")
	classified := AutoWrap(err)

	if errorfamily.IsRetryable(classified) {
		t.Error("validation domain should not be retryable")
	}
	if classified.Family() != errorfamily.Rejection {
		t.Errorf("validation domain should be Rejection, got %v", classified.Family())
	}
}

func TestAutoWrap_TagOverridesDomain(t *testing.T) {
	err := oops.In("validation").Tags("retryable").Errorf("override")
	classified := AutoWrap(err)

	if !errorfamily.IsRetryable(classified) {
		t.Error("tag 'retryable' should override validation domain to Transient")
	}
}

func TestAutoWrap_PlainError(t *testing.T) {
	err := errors.New("plain stdlib error")
	classified := AutoWrap(err)

	if classified.Family() != errorfamily.Transient {
		t.Errorf("plain error should infer Transient (fail-open), got %v", classified.Family())
	}
	if !errors.Is(classified, err) {
		t.Error("plain error should be reachable via errors.Is")
	}
}

func TestBridge_Integration_FullStack(t *testing.T) {
	err := oops.In("database").
		Tags("timeout").
		Code("db.timeout").
		With("host", "db1").
		With("port", 5432).
		Errorf("connection failed after 5000ms")

	classified := AutoWrap(err)

	if errorfamily.ExitCode(classified) != 75 {
		t.Errorf("ExitCode = %d, want 75 (EX_TEMPFAIL)", errorfamily.ExitCode(classified))
	}

	ctx := classified.ErrorContext()
	if ctx["host"] != "db1" {
		t.Errorf("context host = %q, want %q", ctx["host"], "db1")
	}
	if ctx["domain"] != "database" {
		t.Errorf("context domain = %q, want %q", ctx["domain"], "database")
	}

	if !errorfamily.IsRetryable(classified) {
		t.Error("database timeout should be retryable")
	}

	if classified.Domain() != "database" {
		t.Errorf("oops domain = %q, want %q", classified.Domain(), "database")
	}

	if classified.ErrorCode() != "db.timeout" {
		t.Errorf("ErrorCode() = %q, want %q", classified.ErrorCode(), "db.timeout")
	}
}

func BenchmarkWrap(b *testing.B) {
	base := oops.With("host", "db1").With("port", "5432").Errorf("test")
	b.ResetTimer()
	for range b.N {
		Wrap(base, errorfamily.Transient)
	}
}

func BenchmarkInferFamily(b *testing.B) {
	err := oops.In("database").Tags("timeout").With("host", "db1").Errorf("test")
	b.ResetTimer()
	for range b.N {
		InferFamily(err)
	}
}

func BenchmarkAutoWrap(b *testing.B) {
	err := oops.In("database").Tags("timeout").With("host", "db1").Errorf("test")
	b.ResetTimer()
	for range b.N {
		AutoWrap(err)
	}
}

func BenchmarkErrorContext(b *testing.B) {
	base := oops.
		With("host", "db1").
		With("port", "5432").
		With("user", "admin").
		Errorf("test")
	classified := Wrap(base, errorfamily.Transient)
	b.ResetTimer()
	for range b.N {
		_ = classified.ErrorContext()
	}
}

func ExampleAutoWrap() {
	err := oops.In("database").
		Tags("timeout").
		Code("db.timeout").
		With("host", "db1.example.com").
		Errorf("connection failed")

	classified := AutoWrap(err)

	fmt.Println(errorfamily.IsRetryable(classified))
	fmt.Println(errorfamily.ExitCode(classified))
	fmt.Println(classified.Family().String())
	fmt.Println(classified.ErrorCode())
	// Output:
	// true
	// 75
	// transient
	// db.timeout
}

func ExampleInferFamily_tagOverride() {
	err := oops.In("validation").Tags("retryable").Errorf("override domain default")
	classified := AutoWrap(err)

	fmt.Println(classified.Family().String())
	// Output: transient
}

func ExampleWrap_plainError() {
	err := errors.New("file not found")
	classified := Wrap(err, errorfamily.Rejection)

	fmt.Println(classified.Error())
	fmt.Println(classified.Family().String())
	fmt.Println(errors.Is(classified, err))
	// Output:
	// [rejection] file not found
	// rejection
	// true
}
