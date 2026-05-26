package errorfamily

import (
	"errors"
	"testing"
)

func BenchmarkClassifyErrorStruct(b *testing.B) {
	e := NewTransient("db.timeout", "query took too long")
	for b.Loop() {
		_ = Classify(e)
	}
}

func BenchmarkClassifyPlainError(b *testing.B) {
	e := errors.New("generic failure")
	for b.Loop() {
		_ = Classify(e)
	}
}

func BenchmarkClassifyRegistered(b *testing.B) {
	e := errors.New("registered sentinel")
	RegisterClassification(e, Transient)
	b.ResetTimer()
	for b.Loop() {
		_ = Classify(e)
	}
}

func BenchmarkClassifyRetryable(b *testing.B) {
	e := &retryablePlainError{retryable: true}
	for b.Loop() {
		_ = Classify(e)
	}
}

func BenchmarkClassifyMultiError(b *testing.B) {
	e := errors.Join(
		NewTransient("db.timeout", "msg"),
		NewRejection("config.invalid", "msg"),
	)
	for b.Loop() {
		_ = Classify(e)
	}
}

func BenchmarkHandleError(b *testing.B) {
	e := NewTransient("db.timeout", "query took too long").
		WithContext("host", "localhost").
		WithContext("port", "5432")
	for b.Loop() {
		_ = HandleError(e)
	}
}

func BenchmarkExitCode(b *testing.B) {
	e := NewTransient("db.timeout", "msg")
	for b.Loop() {
		_ = ExitCode(e)
	}
}

func BenchmarkIsRetryable(b *testing.B) {
	e := NewRejection("config.invalid", "msg")
	for b.Loop() {
		_ = IsRetryable(e)
	}
}

func BenchmarkErrorString(b *testing.B) {
	e := NewTransient("db.timeout", "query took too long")
	for b.Loop() {
		_ = e.Error()
	}
}

func BenchmarkErrorContext(b *testing.B) {
	e := NewTransient("db.timeout", "msg").
		WithContext("host", "localhost")
	for b.Loop() {
		_ = e.ErrorContext()
	}
}

func BenchmarkSummary(b *testing.B) {
	e := NewTransient("db.timeout", "query took too long")
	for b.Loop() {
		_ = e.Summary()
	}
}

func BenchmarkWithContext(b *testing.B) {
	e := NewTransient("db.timeout", "msg")
	for b.Loop() {
		_ = e.WithContext("host", "localhost")
	}
}

func BenchmarkParseFamily(b *testing.B) {
	for b.Loop() {
		_ = ParseFamily("transient")
	}
}

func BenchmarkFamilyString(b *testing.B) {
	for b.Loop() {
		_ = Transient.String()
	}
}

type retryablePlainError struct {
	retryable bool
}

func (r *retryablePlainError) Error() string     { return "retryable plain error" }
func (r *retryablePlainError) IsRetryable() bool { return r.retryable }
