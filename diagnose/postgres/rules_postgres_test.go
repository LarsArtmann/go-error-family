package postgres

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

func TestPostgresRuleName(t *testing.T) {
	r := &PostgresRule{}
	if got := r.Name(); got != "postgres" {
		t.Errorf("Name() = %q, want %q", got, "postgres")
	}
}

func TestPostgresRuleApplicable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			"db_host context",
			errorfamily.NewTransient("test", "msg").WithContext("db_host", "localhost"),
			true,
		},
		{
			"db_port context",
			errorfamily.NewTransient("test", "msg").WithContext("db_port", "5432"),
			true,
		},
		{
			"db_name context",
			errorfamily.NewTransient("test", "msg").WithContext("db_name", "mydb"),
			true,
		},
		{
			"database_url context",
			errorfamily.NewTransient("test", "msg").WithContext("database_url", "postgres://..."),
			true,
		},
		{
			"postgres_host context",
			errorfamily.NewTransient("test", "msg").WithContext("postgres_host", "db"),
			true,
		},
		{"db code", errorfamily.NewTransient("db.timeout", "msg"), true},
		{"database code", errorfamily.NewTransient("database.error", "msg"), true},
		{
			"sql substring in context",
			errorfamily.NewTransient("test", "msg").WithContext("url", "postgres://host"),
			true,
		},
		{
			"postgres substring in context",
			errorfamily.NewTransient("test", "msg").
				WithContext("info", "postgresql connection failed"),
			true,
		},
		{
			"transient + sql message",
			errorfamily.NewTransient("test", "sql error during query"),
			true,
		},
		{
			"database substring in context",
			errorfamily.NewTransient("test", "msg").WithContext("info", "database unavailable"),
			true,
		},
		{"unrelated code", errorfamily.NewTransient("file.not_found", "msg"), false},
		{
			"unrelated context",
			errorfamily.NewTransient("test", "msg").WithContext("path", "/tmp"),
			false,
		},
		{
			"rejection + no db context",
			errorfamily.NewRejection("config.invalid", "bad config"),
			false,
		},
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

func TestPostgresRuleRunWithHostPort(t *testing.T) {
	r := &PostgresRule{}
	err := errorfamily.NewTransient("db.timeout", "msg").
		WithContext("host", "localhost").
		WithContext("port", "5432")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["host"] != "localhost" {
		t.Errorf("host detail = %q, want 'localhost'", result.Details["host"])
	}
	if result.Details["port"] != "5432" {
		t.Errorf("port detail = %q, want '5432'", result.Details["port"])
	}
}

func TestPostgresRuleRunDefaultHostPort(t *testing.T) {
	r := &PostgresRule{}
	err := errorfamily.NewTransient("db.error", "msg")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["host"] != "localhost" {
		t.Errorf("default host = %q, want 'localhost'", result.Details["host"])
	}
	if result.Details["port"] != "5432" {
		t.Errorf("default port = %q, want '5432'", result.Details["port"])
	}
}

func TestPostgresRuleRunNonLocalhost(t *testing.T) {
	r := &PostgresRule{}
	err := errorfamily.NewTransient("db.timeout", "msg").
		WithContext("db_host", "192.0.2.1").
		WithContext("db_port", "5433")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["host"] != "192.0.2.1" {
		t.Errorf("host = %q, want '192.0.2.1'", result.Details["host"])
	}
	if result.Details["port"] != "5433" {
		t.Errorf("port = %q, want '5433'", result.Details["port"])
	}
	if result.Status != diagnose.StatusFailed {
		t.Errorf("Expected StatusFailed for unreachable host, got %v", result.Status)
	}
	if result.SuggestedFix == "" {
		t.Error("Expected non-empty SuggestedFix")
	}
}

func TestPostgresRuleResolveHost(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"db_host", errorfamily.NewTransient("test", "msg").WithContext("db_host", "db1"), "db1"},
		{
			"postgres_host",
			errorfamily.NewTransient("test", "msg").WithContext("postgres_host", "db2"),
			"db2",
		},
		{"host", errorfamily.NewTransient("test", "msg").WithContext("host", "db3"), "db3"},
		{"PGHOST", errorfamily.NewTransient("test", "msg").WithContext("PGHOST", "db4"), "db4"},
		{"default", errorfamily.NewTransient("test", "msg"), "localhost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PostgresRule{}
			if got := r.resolveHost(tt.err); got != tt.want {
				t.Errorf("resolveHost() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPostgresRuleResolvePort(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			"db_port valid",
			errorfamily.NewTransient("test", "msg").WithContext("db_port", "5433"),
			"5433",
		},
		{
			"db_port invalid",
			errorfamily.NewTransient("test", "msg").WithContext("db_port", "abc"),
			"5432",
		},
		{
			"postgres_port",
			errorfamily.NewTransient("test", "msg").WithContext("postgres_port", "5434"),
			"5434",
		},
		{"port key", errorfamily.NewTransient("test", "msg").WithContext("port", "5435"), "5435"},
		{"PGPORT", errorfamily.NewTransient("test", "msg").WithContext("PGPORT", "5436"), "5436"},
		{"default", errorfamily.NewTransient("test", "msg"), "5432"},
		{"empty port", errorfamily.NewTransient("test", "msg").WithContext("port", ""), "5432"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PostgresRule{}
			if got := r.resolvePort(tt.err); got != tt.want {
				t.Errorf("resolvePort() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsPostgresRunning(t *testing.T) {
	running := IsPostgresRunning(context.Background(), "", "")
	_ = running

	runningWithParams := IsPostgresRunning(context.Background(), "192.0.2.1", "5433")
	_ = runningWithParams
}

func TestIsPostgresRunningDefaults(t *testing.T) {
	running := IsPostgresRunning(context.Background(), "localhost", "5432")
	_ = running
}
