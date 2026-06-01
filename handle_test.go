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

func testDiagnosticFunc(_ context.Context, _ error) []DiagnosticFinding {
	return []DiagnosticFinding{
		{RuleName: "test", Status: "failed", Summary: "something failed", Confidence: 0.9},
	}
}

func testOnDiagnosedPtr(called *bool) func(error, []DiagnosticFinding) {
	return func(_ error, _ []DiagnosticFinding) { *called = true }
}

func TestHandleErrorWithConfigDiagnostics(t *testing.T) {
	var buf bytes.Buffer
	called := false
	err := NewTransient("db.timeout", "timed out")

	code := HandleErrorWithConfig(err, HandleConfig{
		Output:         &buf,
		Diagnose:       true,
		DiagnosticFunc: testDiagnosticFunc,
		OnDiagnosed:    testOnDiagnosedPtr(&called),
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
		Output:         &buf,
		Diagnose:       false,
		DiagnosticFunc: testDiagnosticFunc,
		OnDiagnosed:    testOnDiagnosedPtr(&called),
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
	result := HandleErrorDetailed(plainError("something went wrong"))

	if result.ExitCode != 75 {
		t.Errorf("plain error should default to Transient (exit 75), got %d", result.ExitCode)
	}
}

func TestHandleErrorPlainError(t *testing.T) {
	code := HandleError(plainError("unknown"))
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

type plainError string

func (e plainError) Error() string { return string(e) }

func TestRegisterTemplateAndLookup(t *testing.T) {
	RegisterTemplate("test.registered", MessageTemplate{
		What: "Custom message for {{.key}}",
		Fix:  "Do the thing",
	})
	t.Cleanup(func() { UnregisterTemplate("test.registered") })

	var buf bytes.Buffer
	err := NewRejection("test.registered", "msg").WithContext("key", "value")
	code := HandleErrorWithConfig(err, HandleConfig{Output: &buf})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	output := buf.String()
	if !strings.Contains(output, "Custom message for value") {
		t.Errorf("should use registered template: %q", output)
	}
	if !strings.Contains(output, "Do the thing") {
		t.Errorf("should include fix from template: %q", output)
	}
}

func TestRegisterTemplateCaseInsensitive(t *testing.T) {
	RegisterTemplate("Test.CASE_Code", MessageTemplate{
		What: "Case insensitive template",
	})
	t.Cleanup(func() { UnregisterTemplate("Test.CASE_Code") })

	var buf bytes.Buffer
	err := NewRejection("test.case_code", "msg")
	HandleErrorWithConfig(err, HandleConfig{Output: &buf})
	if !strings.Contains(buf.String(), "Case insensitive template") {
		t.Errorf("should match case-insensitively: %q", buf.String())
	}
}

func TestDefaultMessagesTable(t *testing.T) {
	tests := []struct {
		code     string
		wantWhat string
	}{
		{"file.not_found", "A required resource was not found."},
		{"permission.denied", "Permission was denied."},
		{"db.timeout", "The database operation timed out."},
		{"db.connection", "Could not establish a database connection."},
		{"db.error", "A database operation failed."},
		{"config.invalid", "There is a configuration issue."},
		{"config.not_found", "A configuration file was not found."},
		{"conflict", "A conflict was detected."},
		{"validation", "Validation failed."},
		{"timeout", "The operation timed out."},
		{"connection.refused", "Could not establish a connection."},
		{"git.error", "A git operation failed."},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			var buf bytes.Buffer
			err := NewRejection(tt.code, "msg")
			HandleErrorWithConfig(err, HandleConfig{Output: &buf})
			output := buf.String()
			if !strings.Contains(output, tt.wantWhat) {
				t.Errorf("code %q: output %q should contain %q", tt.code, output, tt.wantWhat)
			}
		})
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
		Diagnose:       true,
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
		Diagnose:       true,
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
