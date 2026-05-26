package diagnose

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// PostgresRule diagnoses PostgreSQL-related errors.
// Checks: pg_isready, TCP connectivity, service status.
//
// Matches errors with context containing: postgres, postgresql, database, db_host, db_port,
// or error codes containing "db" or "database", or Transient family errors with db-related context.
type PostgresRule struct{}

func (r *PostgresRule) Name() string { return "postgres" }

func (r *PostgresRule) Applicable(err error) bool {
	return postgresSpec.matches(err)
}

var postgresSpec = ruleSpec{
	ContextSubstr: []string{"postgres", "postgresql", "database", "sql"},
	ContextKeys:   []string{"db_host", "db_port", "db_name", "database_url", "postgres_host"},
	CodeContains:  []string{"db.", "database"},
	Extra:         func(err error) bool { return familyIs(err, errorfamily.Transient) && hasContextSubstring(err, "sql") },
}

func (r *PostgresRule) Run(ctx context.Context, err error) (*DiagnosticResult, error) {
	host := r.resolveHost(err)
	port := r.resolvePort(err)

	result := &DiagnosticResult{
		Details: map[string]string{
			"host": host,
			"port": port,
		},
	}

	// Check 1: pg_isready
	if commandExists("pg_isready") {
		stdout, _, exitCode, _ := runCommand(ctx, 5*time.Second, "pg_isready", "-h", host, "-p", port)
		result.Details["pg_isready"] = stdout
		if exitCode == 0 {
			result.Status = StatusHealthy
			result.Summary = fmt.Sprintf("PostgreSQL is running on %s:%s", host, port)
			result.Confidence = ConfidenceNotCause // Postgres is fine — probably not the root cause
			return result, nil
		}
		result.Summary = fmt.Sprintf("PostgreSQL is NOT responding on %s:%s: %s", host, port, stdout)
		result.SuggestedFix = r.suggestStartFix()
		result.Confidence = ConfidenceCertain
		result.Status = StatusFailed
		return result, nil
	}

	// Check 2: TCP connectivity
	result.Details["pg_isready"] = "not available"
	addr := net.JoinHostPort(host, port)
	conn, dialErr := net.DialTimeout("tcp", addr, 3*time.Second)
	if dialErr == nil {
		_ = conn.Close()
		result.Status = StatusHealthy
		result.Summary = fmt.Sprintf("TCP connection to %s succeeded — PostgreSQL may be running", addr)
		result.Confidence = ConfidencePartial
		return result, nil
	}

	result.Status = StatusFailed
	result.Summary = fmt.Sprintf("Cannot connect to %s: %v", addr, dialErr)
	result.Details["tcp_error"] = dialErr.Error()
	result.SuggestedFix = fmt.Sprintf(
		"Check if PostgreSQL is running:\n  pg_isready -h %s -p %s\n\nStart if needed:\n  %s",
		host,
		port,
		r.suggestStartFix(),
	)
	result.Confidence = ConfidenceVeryHigh

	return result, nil
}

func (r *PostgresRule) resolveHost(err error) string {
	return resolveContextKey(err, []string{"db_host", "postgres_host", "host", "PGHOST"}, "localhost")
}

func (r *PostgresRule) resolvePort(err error) string {
	for _, key := range []string{"db_port", "postgres_port", "port", "PGPORT"} {
		if v := contextValue(err, key); v != "" {
			if _, err := strconv.Atoi(v); err == nil {
				return v
			}
		}
	}
	return "5432"
}

func (r *PostgresRule) suggestStartFix() string {
	switch {
	case commandExists("brew"):
		return "brew services start postgresql"
	case commandExists("systemctl"):
		return "sudo systemctl start postgresql"
	case commandExists("service"):
		return "sudo service postgresql start"
	default:
		return "pg_ctl start"
	}
}

// IsPostgresRunning is a standalone helper that checks if PostgreSQL is accessible.
// Useful for health checks and startup validation.
func IsPostgresRunning(ctx context.Context, host, port string) bool {
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}

	if commandExists("pg_isready") {
		_, _, exitCode, _ := runCommand(ctx, 5*time.Second, "pg_isready", "-h", host, "-p", port)
		return exitCode == 0
	}

	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
