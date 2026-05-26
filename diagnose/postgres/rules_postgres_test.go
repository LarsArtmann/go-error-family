package postgres

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestPostgresRuleApplicable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			"postgres context",
			errorfamily.NewTransient("test", "msg").WithContext("db_host", "localhost"),
			true,
		},
		{"db code", errorfamily.NewTransient("db.timeout", "msg"), true},
		{
			"sql substring",
			errorfamily.NewTransient("test", "msg").WithContext("url", "postgres://host"),
			true,
		},
		{"unrelated", errorfamily.NewTransient("test", "msg"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PostgresRule{}
			if got := r.Applicable(tt.err); got != tt.want {
				t.Errorf("Applicable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostgresRuleRun(t *testing.T) {
	r := &PostgresRule{}
	err := errorfamily.NewTransient("db.timeout", "msg").
		WithContext("host", "localhost").
		WithContext("port", "5432")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	// Can't assert specific status (depends on system), but should not panic
	if result.Details["host"] != "localhost" {
		t.Errorf("host detail = %q, want 'localhost'", result.Details["host"])
	}
}

func TestPostgresRuleResolveHost(t *testing.T) {
	r := &PostgresRule{}
	err := errorfamily.NewTransient("test", "msg").WithContext("db_host", "myhost")
	if host := r.resolveHost(err); host != "myhost" {
		t.Errorf("resolveHost = %q, want 'myhost'", host)
	}
}

func TestPostgresRuleResolvePortInvalid(t *testing.T) {
	r := &PostgresRule{}
	err := errorfamily.NewTransient("test", "msg").WithContext("db_port", "not-a-number")
	if port := r.resolvePort(err); port != "5432" {
		t.Errorf("resolvePort with invalid port = %q, want '5432'", port)
	}
}

func TestPostgresRuleResolveDefaults(t *testing.T) {
	r := &PostgresRule{}
	err := errorfamily.NewTransient("test", "msg")
	if host := r.resolveHost(err); host != "localhost" {
		t.Errorf("default host = %q, want 'localhost'", host)
	}
	if port := r.resolvePort(err); port != "5432" {
		t.Errorf("default port = %q, want '5432'", port)
	}
}

func TestIsPostgresRunning(t *testing.T) {
	// Just verify it doesn't panic
	IsPostgresRunning(context.Background(), "", "")
}
