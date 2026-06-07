package errorfamily

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

type plainError string

func (e plainError) Error() string { return string(e) }

func TestHandleErrorDetailedNil(t *testing.T) {
	result := HandleErrorDetailed(nil)
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestHandleErrorDetailedRejection(t *testing.T) {
	err := NewRejection("file.not_found", "config missing")
	result := HandleErrorDetailed(err)

	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if result.Message == "" {
		t.Error("Message should not be empty")
	}
	if result.SuggestedFix == "" {
		t.Error("SuggestedFix should not be empty for non-retryable errors")
	}
}

func TestHandleErrorDetailedTransient(t *testing.T) {
	err := NewTransient("db.timeout", "timed out")
	result := HandleErrorDetailed(err)

	if result.ExitCode != 75 {
		t.Errorf("ExitCode = %d, want 75", result.ExitCode)
	}
	if result.SuggestedFix != "" {
		t.Errorf("SuggestedFix should be empty for retryable errors, got %q", result.SuggestedFix)
	}
}

func TestHandleErrorDetailedWithCode(t *testing.T) {
	err := NewConflict("state.conflict", "version mismatch")
	result := HandleErrorDetailed(err)

	if !strings.Contains(result.Message, "conflict") {
		t.Errorf("Message should mention conflict: %q", result.Message)
	}
}

func TestHandleErrorDetailedPlainError(t *testing.T) {
	result := HandleErrorDetailed(plainError("something went wrong"))

	if result.ExitCode != 75 {
		t.Errorf("plain error should default to Transient (exit 75), got %d", result.ExitCode)
	}
}

type testContextKey string

func TestHandleErrorWithContextPropagatesContext(t *testing.T) {
	var receivedCtx context.Context
	diagFunc := func(ctx context.Context, _ error) []DiagnosticFinding {
		receivedCtx = ctx
		return nil
	}

	ctx := context.WithValue(context.Background(), testContextKey("test-key"), "test-value")
	err := NewTransient("db.timeout", "timed out")

	var buf bytes.Buffer
	code := HandleErrorWithContext(ctx, err, HandleConfig{
		Output:         &buf,
		DiagnosticFunc: diagFunc,
	})
	if code != 75 {
		t.Errorf("exit code = %d, want 75", code)
	}
	if receivedCtx == nil {
		t.Fatal("DiagnosticFunc was never called")
	}
	if receivedCtx.Value(testContextKey("test-key")) != "test-value" {
		t.Error("context not propagated to DiagnosticFunc")
	}
}

func TestHandleErrorWithContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	diagFunc := func(_ context.Context, _ error) []DiagnosticFinding {
		called = true
		return nil
	}

	err := NewTransient("db.timeout", "timed out")
	var buf bytes.Buffer
	_ = HandleErrorWithContext(ctx, err, HandleConfig{
		Output:         &buf,
		DiagnosticFunc: diagFunc,
	})
	if !called {
		t.Error("DiagnosticFunc should still be called even with cancelled context")
	}
}

func TestHandleErrorDetailedWithConfigTemplateOverride(t *testing.T) {
	err := NewRejection("file.not_found", "missing").WithContext("path", "/etc/config")
	result := HandleErrorDetailedWithConfig(err, HandleConfig{
		TemplateOverride: map[string]MessageTemplate{
			"file.not_found": {
				What: "Custom: {{.path}} not found",
				Fix:  "Create {{.path}}",
			},
		},
	})

	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if !strings.Contains(result.Message, "Custom: /etc/config not found") {
		t.Errorf("Message should use template override: %q", result.Message)
	}
}

func TestHandleErrorDetailedWithRegisteredTemplate(t *testing.T) {
	RegisterTemplate("test.detailed.registered", MessageTemplate{
		What: "Registered template for detailed",
		Fix:  "Fix from registered",
	})
	t.Cleanup(func() { UnregisterTemplate("test.detailed.registered") })

	err := NewRejection("test.detailed.registered", "msg")
	result := HandleErrorDetailed(err)

	if !strings.Contains(result.Message, "Registered template for detailed") {
		t.Errorf("HandleErrorDetailed should use registered templates: %q", result.Message)
	}
}

func TestHandleErrorWithContextNilError(t *testing.T) {
	code := HandleErrorWithContext(context.Background(), nil, HandleConfig{})
	if code != 0 {
		t.Errorf("HandleErrorWithContext(nil) = %d, want 0", code)
	}
}
