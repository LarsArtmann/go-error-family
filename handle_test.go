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
				What: "Could not find {path}",
				Fix:  "Create the file at {path}",
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

func assertExitCode(t *testing.T, result *HandleResult, want int) {
	t.Helper()

	if result.ExitCode != want {
		t.Errorf("ExitCode = %d, want %d", result.ExitCode, want)
	}
}

func TestHandleErrorWithConfigDiagnostics(t *testing.T) {
	var buf bytes.Buffer

	called := false
	err := NewTransient("db.timeout", "timed out")

	code := HandleErrorWithConfig(err, HandleConfig{
		Output:         &buf,
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

func TestHandleErrorWithConfigNoDiagnoseWhenFuncNil(t *testing.T) {
	var buf bytes.Buffer

	err := NewTransient("test", "msg")

	code := HandleErrorWithConfig(err, HandleConfig{
		Output: &buf,
	})
	if code != 75 {
		t.Errorf("exit code = %d, want 75", code)
	}
}

func TestHandleErrorPlainError(t *testing.T) {
	code := HandleError(plainError("unknown"))
	if code != 75 {
		t.Errorf("HandleError(plain) = %d, want 75", code)
	}
}
