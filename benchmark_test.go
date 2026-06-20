package errorfamily

import (
	"errors"
	"fmt"
	"io"
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
		_ = HandleErrorWithConfig(e, HandleConfig{Output: io.Discard})
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

// BenchmarkClassifyManySentinels guards the atomic.Pointer sentinel store: it
// must stay allocation-free as the registry grows. Regression target for the
// lookupSentinel refactor (previously allocated ~1.8KB/3 allocs per call at 50
// sentinels due to a full-map snapshot copy on every Classify).
func BenchmarkClassifyManySentinels(b *testing.B) {
	reg := NewRegistry()
	target := errors.New("target sentinel")
	for i := range 50 {
		reg.RegisterClassification(fmt.Errorf("sentinel-%d", i), Transient)
	}
	reg.RegisterClassification(target, Rejection)
	b.ResetTimer()
	for b.Loop() {
		_ = reg.Classify(target)
	}
}

// BenchmarkClassifyViaRegistryVsPackage measures the indirection cost of a
// custom Registry.Classify versus the package-level Classify (which delegates
// to DefaultRegistry). They should be within noise of each other.
func BenchmarkClassifyViaRegistryVsPackage(b *testing.B) {
	reg := NewRegistry()
	sentinel := errors.New("bench sentinel")
	reg.RegisterClassification(sentinel, Transient)
	RegisterClassification(sentinel, Transient)
	b.ResetTimer()
	b.Run("Registry.Classify", func(b *testing.B) {
		for b.Loop() {
			_ = reg.Classify(sentinel)
		}
	})
	b.Run("package Classify", func(b *testing.B) {
		for b.Loop() {
			_ = Classify(sentinel)
		}
	})
}

func BenchmarkFamilyHTTPStatus(b *testing.B) {
	for b.Loop() {
		_ = Transient.HTTPStatus()
	}
}

func BenchmarkFamilyRetryPolicy(b *testing.B) {
	for b.Loop() {
		_ = Transient.RetryPolicy()
	}
}

func BenchmarkErrorJSON(b *testing.B) {
	e := NewTransient("db.timeout", "query timed out").
		WithContext("host", "db1").
		WithContext("port", "5432")
	for b.Loop() {
		_, _ = e.JSON()
	}
}
