package postgres

import (
	"context"
	"strings"
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

func TestPostgresRuleApplicableTrue(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"db_host", errorfamily.NewTransient("test", "msg").WithContext("db_host", "localhost")},
		{"db_port", errorfamily.NewTransient("test", "msg").WithContext("db_port", "5432")},
		{"db_name", errorfamily.NewTransient("test", "msg").WithContext("db_name", "mydb")},
		{"database_url", errorfamily.NewTransient("test", "msg").WithContext("database_url", "postgres://...")},
		{"postgres_host", errorfamily.NewTransient("test", "msg").WithContext("postgres_host", "db")},
		{"db code", errorfamily.NewTransient("db.timeout", "msg")},
		{"database code", errorfamily.NewTransient("database.error", "msg")},
		{"sql context", errorfamily.NewTransient("test", "msg").WithContext("url", "postgres://host")},
		{"postgres substr", errorfamily.NewTransient("test", "msg").WithContext("info", "postgresql failed")},
		{"transient+sql", errorfamily.NewTransient("test", "sql error during query")},
		{"database substr", errorfamily.NewTransient("test", "msg").WithContext("info", "database unavailable")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PostgresRule{}
			if got := r.Applicable(tt.err); !got {
				t.Errorf("Applicable() = false, want true")
			}
		})
	}
}

func TestPostgresRuleApplicableFalse(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"unrelated code", errorfamily.NewTransient("file.not_found", "msg")},
		{"unrelated context", errorfamily.NewTransient("test", "msg").WithContext("path", "/tmp")},
		{"rejection no db", errorfamily.NewRejection("config.invalid", "bad config")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PostgresRule{}
			if got := r.Applicable(tt.err); got {
				t.Errorf("Applicable() = true, want false")
			}
		})
	}
}

func newPgMockRunner() *diagnose.MockCommandRunner {
	return diagnose.NewMockCommandRunner()
}

func TestPostgresRuleMockPgIsreadyHealthy(t *testing.T) {
	mr := newPgMockRunner()
	mr.Exists_["pg_isready"] = true
	mr.Exists_["brew"] = false
	mr.Exists_["systemctl"] = false
	mr.Exists_["service"] = false
	mr.Responses["pg_isready -h localhost -p 5432"] = diagnose.MockResponse{
		Stdout:   "localhost:5432 - accepting connections",
		ExitCode: 0,
	}

	r := &PostgresRule{Runner: mr}
	err := errorfamily.NewTransient("db.timeout", "msg").
		WithContext("host", "localhost").
		WithContext("port", "5432")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != diagnose.StatusHealthy {
		t.Errorf("Expected StatusHealthy, got %v", result.Status)
	}
	if !strings.Contains(result.Summary, "running") {
		t.Errorf("Expected 'running' in summary, got %q", result.Summary)
	}
	if result.Details["pg_isready"] != "localhost:5432 - accepting connections" {
		t.Errorf("Expected pg_isready output, got %q", result.Details["pg_isready"])
	}
}

func TestPostgresRuleMockPgIsreadyFailed(t *testing.T) {
	mr := newPgMockRunner()
	mr.Exists_["pg_isready"] = true
	mr.Exists_["brew"] = false
	mr.Exists_["systemctl"] = false
	mr.Exists_["service"] = false
	mr.Responses["pg_isready -h localhost -p 5432"] = diagnose.MockResponse{
		Stdout:   "localhost:5432 - no response",
		ExitCode: 2,
	}

	r := &PostgresRule{Runner: mr}
	err := errorfamily.NewTransient("db.connection", "msg")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != diagnose.StatusFailed {
		t.Errorf("Expected StatusFailed, got %v", result.Status)
	}
	if !strings.Contains(result.Summary, "NOT responding") {
		t.Errorf("Expected 'NOT responding' in summary, got %q", result.Summary)
	}
	if result.Confidence != diagnose.ConfidenceCertain {
		t.Errorf("Expected ConfidenceCertain, got %v", result.Confidence)
	}
}

func TestPostgresRuleMockNoPgIsreadyTCPSuccess(t *testing.T) {
	mr := newPgMockRunner()
	mr.Exists_["pg_isready"] = false
	r := &PostgresRule{Runner: mr}
	err := errorfamily.NewTransient("db.error", "msg")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["pg_isready"] != "not available" {
		t.Errorf("Expected pg_isready=not available, got %q", result.Details["pg_isready"])
	}
}

func TestPostgresRuleMockSuggestStartFix(t *testing.T) {
	tests := []struct {
		name        string
		exists      map[string]bool
		wantCommand string
	}{
		{"brew", map[string]bool{"brew": true, "pg_isready": true}, "brew services start postgresql"},
		{"systemctl", map[string]bool{"systemctl": true, "pg_isready": true}, "sudo systemctl start postgresql"},
		{"service", map[string]bool{"service": true, "pg_isready": true}, "sudo service postgresql start"},
		{"default", map[string]bool{"pg_isready": true}, "pg_ctl start"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := newPgMockRunner()
			mr.Exists_ = tt.exists
			r := &PostgresRule{Runner: mr}
			got := r.suggestStartFix()
			if got != tt.wantCommand {
				t.Errorf("suggestStartFix() = %q, want %q", got, tt.wantCommand)
			}
		})
	}
}

func TestPostgresRuleMockCustomHostPort(t *testing.T) {
	mr := newPgMockRunner()
	mr.Exists_["pg_isready"] = true
	mr.Exists_["brew"] = false
	mr.Exists_["systemctl"] = false
	mr.Exists_["service"] = false
	mr.Responses["pg_isready -h db.example.com -p 5433"] = diagnose.MockResponse{
		Stdout:   "db.example.com:5433 - accepting connections",
		ExitCode: 0,
	}

	r := &PostgresRule{Runner: mr}
	err := errorfamily.NewTransient("db.timeout", "msg").
		WithContext("db_host", "db.example.com").
		WithContext("db_port", "5433")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details[strHost] != "db.example.com" {
		t.Errorf("host = %q, want 'db.example.com'", result.Details[strHost])
	}
	if result.Details["port"] != "5433" {
		t.Errorf("port = %q, want '5433'", result.Details["port"])
	}
}

func TestPostgresRuleMockUsesCommandRunner(t *testing.T) {
	mr := newPgMockRunner()
	mr.Exists_["pg_isready"] = true
	mr.Exists_["brew"] = false
	mr.Exists_["systemctl"] = false
	mr.Exists_["service"] = false
	mr.Responses["pg_isready -h localhost -p 5432"] = diagnose.MockResponse{
		Stdout:   "accepting",
		ExitCode: 0,
	}

	r := &PostgresRule{Runner: mr}
	err := errorfamily.NewTransient("db.error", "msg")
	_, _ = r.Run(context.Background(), err)

	calls := mr.Calls()
	hasExists := false
	hasRun := false
	for _, c := range calls {
		if strings.HasPrefix(c, "exists:") {
			hasExists = true
		}
		if strings.Contains(c, "pg_isready") {
			hasRun = true
		}
	}
	if !hasExists {
		t.Error("Expected Exists() call")
	}
	if !hasRun {
		t.Error("Expected Run() call with pg_isready")
	}
}

// Unit tests for resolveHost/resolvePort (no network needed).

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

func TestPostgresRuleDefaultHostPort(t *testing.T) {
	r := &PostgresRule{}
	err := errorfamily.NewTransient("db.error", "msg")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details[strHost] != "localhost" {
		t.Errorf("default host = %q, want 'localhost'", result.Details[strHost])
	}
	if result.Details["port"] != "5432" {
		t.Errorf("default port = %q, want '5432'", result.Details["port"])
	}
}

func TestPostgresRuleRunWithNonLocalhost(t *testing.T) {
	r := &PostgresRule{}
	err := errorfamily.NewTransient("db.timeout", "msg").
		WithContext("db_host", "192.0.2.1").
		WithContext("db_port", "5433")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details[strHost] != "192.0.2.1" {
		t.Errorf("host = %q, want '192.0.2.1'", result.Details[strHost])
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
