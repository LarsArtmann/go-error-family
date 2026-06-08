package postgres

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

// Common string constants to satisfy goconst linter.
const (
	strPostgres = "postgres"
	strHost     = "host"
)

// PostgresRule diagnoses PostgreSQL-related errors.
// Checks: pg_isready, TCP connectivity, service status.
//
// Matches errors with context containing: postgres, postgresql, database, db_host, db_port,
// or error codes containing "db" or "database", or Transient family errors with db-related context.
type PostgresRule struct {
	// Runner is the command runner used to execute system commands.
	// Defaults to diagnose.DefaultCommandRunner{}.
	Runner diagnose.CommandRunner
}

// cmdRunner returns the configured command runner or the default.
func (r *PostgresRule) cmdRunner() diagnose.CommandRunner {
	return diagnose.ResolveRunner(r.Runner)
}

func (r *PostgresRule) Name() string { return strPostgres }

func (r *PostgresRule) Applicable(err error) bool {
	return postgresSpec.Matches(err)
}

var postgresSpec = diagnose.RuleSpec{ //nolint:gochecknoglobals // Immutable rule matching spec.
	ContextSubstr: []string{strPostgres, "postgresql", "database", "sql"},
	ContextKeys: []diagnose.ContextKey{
		diagnose.KeyDBHost,
		diagnose.KeyDBPort,
		diagnose.KeyDBName,
		diagnose.KeyDatabaseURL,
		diagnose.KeyPostgresHost,
	},
	CodeContains: []string{"db.", "database"},
	Extra: func(err error) bool {
		return diagnose.FamilyIs(err, errorfamily.Transient) &&
			diagnose.HasContextSubstring(err, "sql")
	},
}

//nolint:funlen // Diagnostic rule: sequential checks with early returns.
func (r *PostgresRule) Run(ctx context.Context, err error) (*diagnose.DiagnosticResult, error) {
	host := r.resolveHost(err)
	port := r.resolvePort(err)

	result := &diagnose.DiagnosticResult{
		Details: map[string]string{
			strHost: host,
			"port":  port,
		},
		Context: diagnose.ErrorContext(err),
	}

	// Check 1: pg_isready
	if r.cmdRunner().Exists("pg_isready") {
		stdout, exitCode, _ := r.cmdRunner().Run(
			ctx,
			5*time.Second,
			"pg_isready",
			"-h",
			host,
			"-p",
			port,
		)
		result.Details["pg_isready"] = stdout
		if exitCode == 0 {
			result.Status = diagnose.StatusHealthy
			result.Summary = fmt.Sprintf("PostgreSQL is running on %s:%s", host, port)
			result.Confidence = diagnose.ConfidenceNotCause // Postgres is fine — probably not the root cause
			return result, nil
		}
		result.Summary = fmt.Sprintf(
			"PostgreSQL is NOT responding on %s:%s: %s",
			host,
			port,
			stdout,
		)
		result.SuggestedFix = r.suggestStartFix()
		result.Confidence = diagnose.ConfidenceCertain
		result.Status = diagnose.StatusFailed
		return result, nil
	}

	// Check 2: TCP connectivity
	result.Details["pg_isready"] = "not available"
	addr := net.JoinHostPort(host, port)
	conn, dialErr := net.DialTimeout("tcp", addr, 3*time.Second)
	if dialErr == nil {
		_ = conn.Close()
		result.Status = diagnose.StatusHealthy
		result.Summary = fmt.Sprintf(
			"TCP connection to %s succeeded — PostgreSQL may be running",
			addr,
		)
		result.Confidence = diagnose.ConfidencePartial
		return result, nil
	}

	result.Status = diagnose.StatusFailed
	result.Summary = fmt.Sprintf("Cannot connect to %s: %v", addr, dialErr)
	result.Details["tcp_error"] = dialErr.Error()
	result.SuggestedFix = fmt.Sprintf(
		"Check if PostgreSQL is running:\n  pg_isready -h %s -p %s\n\nStart if needed:\n  %s",
		host,
		port,
		r.suggestStartFix(),
	)
	result.Confidence = diagnose.ConfidenceVeryHigh

	return result, nil
}

const strLocalhost = "localhost"

func (r *PostgresRule) resolveHost(err error) string {
	return diagnose.ResolveContextKey(
		err,
		[]string{
			string(diagnose.KeyDBHost),
			string(diagnose.KeyPostgresHost),
			string(diagnose.KeyHost),
			string(diagnose.KeyPGHOST),
		},
		strLocalhost,
	)
}

func (r *PostgresRule) resolvePort(err error) string {
	for _, key := range []string{
		string(diagnose.KeyDBPort),
		string(diagnose.KeyPostgresPort),
		string(diagnose.KeyPort),
		string(diagnose.KeyPGPORT),
	} {
		if v := diagnose.ContextValue(err, key); v != "" {
			if _, err := strconv.Atoi(v); err == nil {
				return v
			}
		}
	}
	return "5432"
}

func (r *PostgresRule) suggestStartFix() string {
	runner := r.cmdRunner()
	switch {
	case runner.Exists("brew"):
		return "brew services start postgresql"
	case runner.Exists("systemctl"):
		return "sudo systemctl start postgresql"
	case runner.Exists("service"):
		return "sudo service postgresql start"
	default:
		return "pg_ctl start"
	}
}

// IsPostgresRunning is a standalone helper that checks if PostgreSQL is accessible.
// Useful for health checks and startup validation.
func IsPostgresRunning(ctx context.Context, host, port string) bool {
	if host == "" {
		host = strLocalhost
	}
	if port == "" {
		port = "5432"
	}

	runner := diagnose.DefaultCommandRunner{}
	if runner.Exists("pg_isready") {
		_, exitCode, _ := runner.Run(
			ctx,
			5*time.Second,
			"pg_isready",
			"-h",
			host,
			"-p",
			port,
		)
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
