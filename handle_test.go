package errorfamily

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHandleErrorNil(t *testing.T) {
	if code := HandleError(nil); code != 0 {
		t.Errorf("HandleError(nil) = %d, want 0", code)
	}
}

func TestHandleErrorRejection(t *testing.T) {
	err := NewRejection("file.not_found", "config missing")
	code := HandleError(err)
	if code != 1 {
		t.Errorf("HandleError(rejection) = %d, want 1", code)
	}
}

func TestHandleErrorTransient(t *testing.T) {
	err := NewTransient("db.timeout", "database timed out")
	code := HandleError(err)
	if code != 75 {
		t.Errorf("HandleError(transient) = %d, want 75", code)
	}
}

func TestHandleErrorWithConfigCustomOutput(t *testing.T) {
	var buf bytes.Buffer
	err := NewRejection("file.not_found", "config missing")

	code := HandleErrorWithConfig(err, HandleConfig{Output: &buf})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	output := buf.String()
	if !strings.Contains(output, "not found") {
		t.Errorf("output should contain 'not found': %q", output)
	}
}

func TestHandleErrorWithConfigTemplateOverride(t *testing.T) {
	var buf bytes.Buffer
	err := NewRejection("file.not_found", "missing")

	code := HandleErrorWithConfig(err, HandleConfig{
		Output: &buf,
		TemplateOverride: map[string]MessageTemplate{
			"file.not_found": {
				What: "Could not find {{.path}}",
				Fix:  "Create the file at {{.path}}",
			},
		},
	})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	output := buf.String()
	if !strings.Contains(output, "Could not find") {
		t.Errorf("output should use template: %q", output)
	}
}

func TestHandleErrorWithConfigNilError(t *testing.T) {
	code := HandleErrorWithConfig(nil, HandleConfig{})
	if code != 0 {
		t.Errorf("HandleErrorWithConfig(nil) = %d, want 0", code)
	}
}

func TestHandleErrorWithConfigDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	called := false
	err := NewTransient("db.timeout", "timed out")

	code := HandleErrorWithConfig(err, HandleConfig{
		Output:           &buf,
		Diagnose:         true,
		DiagnosticRunner: &mockDiagnosticRunner{results: "diagnostic result"},
		OnDiagnosed: func(e error, results any) {
			called = true
		},
	})
	if code != 75 {
		t.Errorf("exit code = %d, want 75", code)
	}
	if !called {
		t.Error("OnDiagnosed should have been called")
	}
}

func TestHandleErrorWithConfigNoDiagnoseWhenDisabled(t *testing.T) {
	var buf bytes.Buffer
	called := false
	err := NewTransient("test", "msg")

	HandleErrorWithConfig(err, HandleConfig{
		Output:           &buf,
		Diagnose:         false,
		DiagnosticRunner: &mockDiagnosticRunner{results: "diagnostic result"},
		OnDiagnosed: func(e error, results any) {
			called = true
		},
	})
	if called {
		t.Error("OnDiagnosed should NOT be called when Diagnose is false")
	}
}

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
	// Transient errors are retryable, so no SuggestedFix.
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
	result := HandleErrorDetailed(plainErr("something went wrong"))

	if result.ExitCode != 75 {
		t.Errorf("plain error should default to Transient (exit 75), got %d", result.ExitCode)
	}
}

func TestHandleErrorPlainError(t *testing.T) {
	code := HandleError(plainErr("unknown"))
	if code != 75 {
		t.Errorf("HandleError(plain) = %d, want 75", code)
	}
}

func TestMessageTemplateApply(t *testing.T) {
	tmpl := MessageTemplate{
		What:   "Could not connect to {{.host}}:{{.port}}",
		Why:    "The server is not responding.",
		Fix:    "Check {{.host}} is running.",
		WayOut: "Run with --verbose for details.",
	}

	err := NewInfrastructure("db.connection", "connection refused").
		WithContext("host", "localhost").
		WithContext("port", "5432")

	var buf bytes.Buffer
	code := HandleErrorWithConfig(err, HandleConfig{
		Output: &buf,
		TemplateOverride: map[string]MessageTemplate{
			"db.connection": tmpl,
		},
	})
	if code != 69 {
		t.Errorf("exit code = %d, want 69", code)
	}

	output := buf.String()
	if !strings.Contains(output, "localhost") {
		t.Errorf("template should have host substituted: %q", output)
	}
	if !strings.Contains(output, "5432") {
		t.Errorf("template should have port substituted: %q", output)
	}
}

type plainErr string

func (e plainErr) Error() string { return string(e) }

type mockDiagnosticRunner struct {
	results any
}

func (m *mockDiagnosticRunner) Run(_ context.Context, _ error) any {
	return m.results
}
