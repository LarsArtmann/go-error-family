package errorfamily

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestLogError(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))

	err := NewRejection("user.not_found", "no such user").
		WithContext("user_id", "42")
	LogError(err, logger)

	out := buf.String()
	for _, want := range []string{
		"level=ERROR",
		"family=rejection",
		"code=user.not_found",
		"retryable=false",
		"exit_code=1",
		"context.user_id=42",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log output missing %q:\n%s", want, out)
		}
	}
}

func TestLogErrorTransientLogsAtWarn(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))

	LogError(NewTransient("db.timeout", "timed out"), logger)

	out := buf.String()
	if !strings.Contains(out, "level=WARN") {
		t.Errorf("transient should log at WARN:\n%s", out)
	}

	if !strings.Contains(out, "retryable=true") {
		t.Errorf("transient should be retryable=true:\n%s", out)
	}
}

func TestLogErrorNilIsNoop(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))

	LogError(nil, logger)

	if buf.Len() != 0 {
		t.Errorf("nil error should produce no log output, got: %q", buf.String())
	}
}

func TestLogErrorNilLoggerUsesDefault(t *testing.T) {
	// Should not panic when logger is nil (falls back to slog.Default).
	LogError(errors.New("boom"), nil)
}

func TestLogErrorContextPropagation(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-123")

	handler := &ctxRecordingHandler{}
	logger := slog.New(handler)

	LogErrorContext(ctx, NewRejection("c", "m"), logger)

	if handler.lastCtx == nil {
		t.Fatal("handler did not receive a context")
	}

	if got := handler.lastCtx.Value(ctxKey{}); got != "trace-123" {
		t.Errorf("context not propagated: got %v, want trace-123", got)
	}
}

type ctxKey struct{}

type ctxRecordingHandler struct {
	lastCtx context.Context
}

func (h *ctxRecordingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *ctxRecordingHandler) Handle(ctx context.Context, _ slog.Record) error {
	h.lastCtx = ctx

	return nil
}
func (h *ctxRecordingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *ctxRecordingHandler) WithGroup(_ string) slog.Handler      { return h }

func TestLogErrorExitCodeAttribute(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))

	LogError(NewTransient("db.timeout", "timed out"), logger)

	out := buf.String()
	// Transient → exit code 75 (EX_TEMPFAIL)
	if !strings.Contains(out, "exit_code=75") {
		t.Errorf("transient should have exit_code=75:\n%s", out)
	}
}

func TestHandleErrorWithLoggerEmitsStructuredLog(t *testing.T) {
	var (
		logBuf bytes.Buffer
		outBuf bytes.Buffer
	)

	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	err := NewRejection("user.not_found", "no such user").
		WithContext("user_id", "42")

	exitCode := HandleErrorWithConfig(err, HandleConfig{
		Output: &outBuf,
		Logger: logger,
	})

	if exitCode != 1 {
		t.Errorf("exit code: got %d, want 1", exitCode)
	}

	// Human output went to outBuf
	if !strings.Contains(outBuf.String(), "user.not_found") {
		t.Errorf("human output missing error code:\n%s", outBuf.String())
	}

	// Structured log went to logBuf
	logOut := logBuf.String()
	for _, want := range []string{
		"level=ERROR",
		"family=rejection",
		"code=user.not_found",
		"retryable=false",
		"exit_code=1",
		"context.user_id=42",
	} {
		if !strings.Contains(logOut, want) {
			t.Errorf("structured log missing %q:\n%s", want, logOut)
		}
	}
}

func TestHandleErrorWithoutLoggerSkipsLogging(t *testing.T) {
	var outBuf bytes.Buffer

	err := NewRejection("user.not_found", "no such user")
	HandleErrorWithConfig(err, HandleConfig{
		Output: &outBuf,
	})

	// Human output present
	if !strings.Contains(outBuf.String(), "user.not_found") {
		t.Errorf("human output missing error code:\n%s", outBuf.String())
	}
}

func TestHandleErrorWithLoggerTransientLogsAtWarn(t *testing.T) {
	var logBuf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	HandleErrorWithConfig(NewTransient("db.timeout", "timed out"), HandleConfig{
		Logger: logger,
	})

	logOut := logBuf.String()
	if !strings.Contains(logOut, "level=WARN") {
		t.Errorf("transient via HandleError should log at WARN:\n%s", logOut)
	}

	if !strings.Contains(logOut, "exit_code=75") {
		t.Errorf("transient should have exit_code=75:\n%s", logOut)
	}
}
