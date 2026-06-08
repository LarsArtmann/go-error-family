package diagnose

import (
	"context"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestHasContextKey(t *testing.T) {
	err := errorfamily.NewTransient("test", "msg").WithContext("host", "localhost")
	if !HasContextKey(err, "host") {
		t.Error("should find 'host' context key")
	}
	if HasContextKey(err, "port") {
		t.Error("should not find 'port' context key")
	}
}

func TestHasContextKeyPlainError(t *testing.T) {
	err := errors.New("plain error")
	if HasContextKey(err, "anything") {
		t.Error("plain error should not have context keys")
	}
}

func TestContextValue(t *testing.T) {
	err := errorfamily.NewTransient("test", "msg").WithContext("host", "localhost")
	if v := ContextValue(err, "host"); v != "localhost" {
		t.Errorf("ContextValue(host) = %q, want 'localhost'", v)
	}
	if v := ContextValue(err, "missing"); v != "" {
		t.Errorf("ContextValue(missing) = %q, want empty", v)
	}
}

func TestHasContextSubstring(t *testing.T) {
	err := errorfamily.NewTransient("test", "msg").WithContext("path", "/var/data/config.yaml")
	if !HasContextSubstring(err, "config.yaml") {
		t.Error("should find 'config.yaml' in context values")
	}
	if HasContextSubstring(err, "nonexistent_xyz") {
		t.Error("should not find random substring")
	}
}

func TestHasContextSubstringInErrorMessage(t *testing.T) {
	err := errors.New("connection refused")
	if !HasContextSubstring(err, "connection refused") {
		t.Error("should find substring in error message for plain errors")
	}
}

func TestFamilyIs(t *testing.T) {
	err := errorfamily.NewTransient("test", "msg")
	if !FamilyIs(err, errorfamily.Transient) {
		t.Error("Transient error should match Transient family")
	}
	if FamilyIs(err, errorfamily.Rejection) {
		t.Error("Transient error should not match Rejection family")
	}
}

func TestErrorCodeContains(t *testing.T) {
	err := errorfamily.NewTransient("db.timeout", "msg")
	if !ErrorCodeContains(err, "db.") {
		t.Error("should find 'db.' in error code")
	}
	if ErrorCodeContains(err, "network") {
		t.Error("should not find 'network' in error code")
	}
}

func TestErrorCodeContainsPlainError(t *testing.T) {
	err := errors.New("plain error")
	if ErrorCodeContains(err, "anything") {
		t.Error("plain error should not match error code")
	}
}

func TestContextKeyStringValues(t *testing.T) {
	tests := []struct {
		key  ContextKey
		want string
	}{
		{KeyHost, "host"},
		{KeyPort, "port"},
		{KeyPath, "path"},
		{KeyDBHost, "db_host"},
		{KeyDBPort, "db_port"},
		{KeyDBName, "db_name"},
		{KeyDatabaseURL, "database_url"},
		{KeyPostgresHost, "postgres_host"},
		{KeyRepository, "repository"},
		{KeyRepo, "repo"},
		{KeyGitDir, "git_dir"},
	}
	for _, tt := range tests {
		if string(tt.key) != tt.want {
			t.Errorf("ContextKey(%q).String() = %q, want %q", tt.key, string(tt.key), tt.want)
		}
	}
}

func TestContextKeyWithRuleSpec(t *testing.T) {
	spec := RuleSpec{
		ContextKeys: []ContextKey{KeyHost, KeyPort},
	}
	err := errorfamily.NewTransient("test", "msg").
		WithContext("host", "localhost")
	if !spec.Matches(err) {
		t.Error("RuleSpec with typed ContextKey should match error with 'host' context")
	}
}

func TestRunAuto(t *testing.T) {
	err := errorfamily.NewTransient("test", "msg").WithContext("host", "example.com")
	results := RunAuto(context.Background(), err)
	_ = results
}

func TestDefaultRunner(t *testing.T) {
	runner := DefaultRunner()
	if runner == nil {
		t.Fatal("DefaultRunner() returned nil")
	}
}

func TestStatusIsValid(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusHealthy, true},
		{StatusDegraded, true},
		{StatusFailed, true},
		{StatusUnknown, true},
		{Status(42), false},
		{Status(-1), false},
	}
	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("Status(%d).IsValid() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		input string
		want  Status
	}{
		{"healthy", StatusHealthy},
		{"HEALTHY", StatusHealthy},
		{"degraded", StatusDegraded},
		{"failed", StatusFailed},
		{"unknown", StatusUnknown},
		{"garbage", StatusUnknown},
	}
	for _, tt := range tests {
		if got := ParseStatus(tt.input); got != tt.want {
			t.Errorf("ParseStatus(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
