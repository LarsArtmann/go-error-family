package errorfamily

import (
	"bytes"
	"strings"
	"testing"
)

func TestMessageTemplateApply(t *testing.T) {
	tmpl := MessageTemplate{
		What:   "Could not connect to {host}:{port}",
		Why:    "The server is not responding.",
		Fix:    "Check {host} is running.",
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

func TestRegisterTemplateAndLookup(t *testing.T) {
	RegisterTemplate("test.registered", MessageTemplate{
		What: "Custom message for {key}",
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
