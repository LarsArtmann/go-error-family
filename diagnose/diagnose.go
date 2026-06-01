// Package diagnose provides deterministic diagnostic rules that investigate errors
// by checking the live system state. Rules are matched by error codes, context keys,
// and message patterns, then run concurrent checks to produce actionable findings.
//
// The package ships with zero-dependency rules (FilesystemRule, NetworkRule) in
// DefaultRunner. Additional rules live in submodules:
//   - diagnose/git: GitRule (checks repo state, working tree, remotes)
//   - diagnose/postgres: PostgresRule (checks pg_isready, TCP connectivity)
//
// Usage:
//
//	runner := diagnose.DefaultRunner()
//	results := runner.Run(ctx, err)
//	for _, r := range results {
//	    fmt.Println(r.Summary)
//	    if r.SuggestedFix != "" {
//	        fmt.Println("  Fix:", r.SuggestedFix)
//	    }
//	}
//
// Custom rules implement the DiagnosticRule interface:
//
//	type MyRule struct{}
//	func (r *MyRule) Name() string { return "my_rule" }
//	func (r *MyRule) Applicable(err error) bool { ... }
//	func (r *MyRule) Run(ctx context.Context, err error) (*diagnose.DiagnosticResult, error) { ... }
//
// For testability, rules that shell out to system commands should accept a
// CommandRunner interface instead of calling RunCommand directly:
//
//	type MyRule struct {
//	    Runner diagnose.CommandRunner  // defaults to diagnose.DefaultCommandRunner{}
//	}
package diagnose

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Common string constants for diagnostic result details to satisfy goconst.
const (
	strTrue      = "true"
	strFalse     = "false"
	strHost      = "host"
	strPort      = "port"
	strLocalhost = "localhost"
	strUnknown   = "unknown"
)

// ContextKey is a typed string for error context keys used by diagnostic rules.
// Use these constants instead of raw strings to prevent typos and enable IDE autocompletion.
type ContextKey string

const (
	// KeyHost is the context key for a hostname or IP address.
	KeyHost ContextKey = "host"
	// KeyPort is the context key for a port number.
	KeyPort ContextKey = "port"
	// KeyPath is the context key for a filesystem path.
	KeyPath ContextKey = "path"
	// KeyFile is the context key for a file path.
	KeyFile ContextKey = "file"
	// KeyDir is the context key for a directory path.
	KeyDir ContextKey = "dir"
	// KeyURL is the context key for a URL.
	KeyURL ContextKey = "url"
	// KeyEndpoint is the context key for a service endpoint.
	KeyEndpoint ContextKey = "endpoint"
	// KeyAddress is the context key for a network address.
	KeyAddress ContextKey = "address"
	// KeyRemote is the context key for a remote host.
	KeyRemote ContextKey = "remote"
	// KeyDBHost is the context key for a database host.
	KeyDBHost ContextKey = "db_host"
	// KeyDBPort is the context key for a database port.
	KeyDBPort ContextKey = "db_port"
	// KeyDBName is the context key for a database name.
	KeyDBName ContextKey = "db_name"
	// KeyDatabaseURL is the context key for a database connection URL.
	KeyDatabaseURL ContextKey = "database_url"
	// KeyPostgresHost is the context key for a PostgreSQL host.
	KeyPostgresHost ContextKey = "postgres_host"
	// KeyConfigPath is the context key for a configuration file path.
	KeyConfigPath ContextKey = "config_path"
	// KeyOutputPath is the context key for an output file path.
	KeyOutputPath ContextKey = "output_path"
	// KeyDirectory is an alias for directory paths.
	KeyDirectory ContextKey = "directory"
	// KeyGit is the context key for git-related context.
	KeyGit ContextKey = "git"
	// KeyRepository is the context key for a repository path.
	KeyRepository ContextKey = "repository"
	// KeyRepo is the context key for a repo path (short form).
	KeyRepo ContextKey = "repo"
	// KeyBranch is the context key for a git branch name.
	KeyBranch ContextKey = "branch"
	// KeyGitDir is the context key for a .git directory path.
	KeyGitDir ContextKey = "git_dir"
	// KeyRepoPath is the context key for a full repository path.
	KeyRepoPath ContextKey = "repo_path"
)

// Status represents the result of a diagnostic check.
type Status int

const (
	// StatusHealthy means the checked component is operating normally.
	StatusHealthy Status = iota
	// StatusDegraded means the component is partially functional.
	StatusDegraded
	// StatusFailed means the component is not functional and may be the root cause.
	StatusFailed
	// StatusUnknown means the check could not be completed.
	StatusUnknown
)

func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusDegraded:
		return "degraded"
	case StatusFailed:
		return "failed"
	case StatusUnknown:
		return strUnknown
	default:
		return strUnknown
	}
}

// DiagnosticResult holds the outcome of a single diagnostic check.
type DiagnosticResult struct {
	// RuleName identifies which rule produced this result.
	RuleName string

	// Status is the outcome of the diagnostic check.
	Status Status

	// Summary is a human-readable explanation of the finding.
	// e.g., "PostgreSQL is not running on localhost:5432"
	Summary string

	// Details contains structured data about the finding.
	// e.g., {"host": "localhost", "port": "5432", "pg_isready": "no response"}
	Details map[string]string

	// Context holds the error context that triggered this rule.
	// Populated from the error's ErrorContext() for programmatic consumers.
	Context map[string]string

	// SuggestedFix is a specific, actionable fix the user can apply.
	// e.g., "Start PostgreSQL: brew services start postgresql"
	SuggestedFix string

	// Confidence is 0.0–1.0 indicating how likely this diagnosis explains the error.
	Confidence float64

	// Duration is how long the diagnostic check took.
	Duration time.Duration
}

// DiagnosticRule is the interface for deterministic error diagnostic checks.
//
// Rules match specific error patterns and run targeted checks.
// For example, a PostgresRule matches errors with database-related context
// and checks if PostgreSQL is running, reachable, and healthy.
type DiagnosticRule interface {
	// Name identifies this rule in logs and results.
	Name() string

	// Applicable returns true if this rule should run for the given error.
	// Rules match on error codes, families, context keys, or message patterns.
	Applicable(err error) bool

	// Run executes the diagnostic check.
	// Must respect context cancellation for long-running checks.
	Run(ctx context.Context, err error) (*DiagnosticResult, error)
}

// Runner executes diagnostic rules against errors.
type Runner struct {
	rules []DiagnosticRule
	mu    sync.RWMutex
}

// NewRunner creates a new diagnostic runner with the given rules.
func NewRunner(rules ...DiagnosticRule) *Runner {
	return &Runner{rules: rules}
}

// Register adds a diagnostic rule to the runner.
func (r *Runner) Register(rule DiagnosticRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, rule)
}

// Run executes all applicable diagnostic rules for the given error.
// Rules are run concurrently for speed. Results are ordered by confidence (highest first).
// Context cancellation is respected.
func (r *Runner) Run(ctx context.Context, err error) []*DiagnosticResult {
	r.mu.RLock()
	rules := make([]DiagnosticRule, 0, len(r.rules))
	for _, rule := range r.rules {
		if rule.Applicable(err) {
			rules = append(rules, rule)
		}
	}
	r.mu.RUnlock()

	if len(rules) == 0 {
		return nil
	}

	results := make([]*DiagnosticResult, len(rules))
	var wg sync.WaitGroup

	for i, rule := range rules {
		wg.Add(1)
		go func(idx int, rl DiagnosticRule) {
			defer wg.Done()
			start := time.Now()
			result, runErr := rl.Run(ctx, err)
			if runErr != nil {
				results[idx] = &DiagnosticResult{
					RuleName: rl.Name(),
					Status:   StatusUnknown,
					Summary:  fmt.Sprintf("diagnostic failed: %v", runErr),
					Details:  map[string]string{"error": runErr.Error()},
				}
			} else if result != nil {
				result.RuleName = rl.Name()
				result.Duration = time.Since(start)
				results[idx] = result
			}
		}(i, rule)
	}

	wg.Wait()

	// Filter nils and sort by confidence descending.
	filtered := make([]*DiagnosticResult, 0, len(results))
	for _, res := range results {
		if res != nil {
			filtered = append(filtered, res)
		}
	}

	// Simple insertion sort by confidence (small N).
	for i := 1; i < len(filtered); i++ {
		for j := i; j > 0 && filtered[j].Confidence > filtered[j-1].Confidence; j-- {
			filtered[j], filtered[j-1] = filtered[j-1], filtered[j]
		}
	}

	return filtered
}

// RunAuto is a convenience function that creates a default Runner with
// all built-in rules and runs diagnostics for the given error.
func RunAuto(ctx context.Context, err error) []*DiagnosticResult {
	return DefaultRunner().Run(ctx, err)
}

// DefaultRunner returns a runner with all built-in diagnostic rules registered.
func DefaultRunner() *Runner {
	return NewRunner(
		&FilesystemRule{},
		&NetworkRule{},
	)
}

// Confidence levels for diagnostic results.
const (
	// ConfidenceNone indicates the check provided almost no useful signal.
	ConfidenceNone float64 = 0.1
	// ConfidenceNotCause indicates the checked component is healthy — probably not the root cause.
	ConfidenceNotCause float64 = 0.3
	// ConfidencePartial indicates the check succeeded but with caveats (e.g., TCP ok but no pg_isready).
	ConfidencePartial float64 = 0.4
	// ConfidenceLikely indicates the check found a likely root cause.
	ConfidenceLikely float64 = 0.7
	// ConfidenceHigh indicates the check found a probable root cause.
	ConfidenceHigh float64 = 0.8
	// ConfidenceVeryHigh indicates strong evidence the checked component is the root cause.
	ConfidenceVeryHigh float64 = 0.85
	// ConfidenceCertain indicates the check conclusively identified the root cause.
	ConfidenceCertain float64 = 0.9
)

// CommandRunner abstracts command execution for diagnostic rules.
// Rules that shell out to system commands should use this interface
// instead of calling RunCommand/CommandExists directly, enabling
// mock-based testing without system dependencies.
type CommandRunner interface {
	// Run executes a command with timeout and returns stdout, exit code, and error.
	Run(ctx context.Context, timeout time.Duration, name string, args ...string) (string, int, error)
	// Exists checks if a command is available on the system PATH.
	Exists(name string) bool
}

// DefaultCommandRunner uses the package-level RunCommand and CommandExists functions.
// This is the zero-value-safe default for all rules.
type DefaultCommandRunner struct{}

func (DefaultCommandRunner) Run(
	ctx context.Context,
	timeout time.Duration,
	name string,
	args ...string,
) (string, int, error) {
	return RunCommand(ctx, timeout, name, args...)
}

func (DefaultCommandRunner) Exists(name string) bool {
	return CommandExists(name)
}

// Helper functions for rule matching.

// ErrorContext extracts the context map from an error, if it implements Contextual.
// Returns an empty map for errors without context.
func ErrorContext(err error) map[string]string {
	if ctx, ok := errors.AsType[errorfamily.Contextual](err); ok {
		return ctx.ErrorContext()
	}
	return map[string]string{}
}

func HasContextKey(err error, keys ...string) bool {
	if ctx, ok := errors.AsType[errorfamily.Contextual](err); ok {
		ctxMap := ctx.ErrorContext()
		for _, key := range keys {
			if _, ok := ctxMap[key]; ok {
				return true
			}
		}
	}
	return false
}

func ContextValue(err error, key string) string {
	if ctx, ok := errors.AsType[errorfamily.Contextual](err); ok {
		return ctx.ErrorContext()[key]
	}
	return ""
}

// ResolveContextKey searches the error's context for the first non-empty value
// among the given keys, returning defaultVal if none match.
func ResolveContextKey(err error, keys []string, defaultVal string) string {
	for _, key := range keys {
		if v := ContextValue(err, key); v != "" {
			return v
		}
	}
	return defaultVal
}

func HasContextSubstring(err error, substr string) bool {
	lower := strings.ToLower(substr)
	if ctx, ok := errors.AsType[errorfamily.Contextual](err); ok {
		for _, v := range ctx.ErrorContext() {
			if strings.Contains(strings.ToLower(v), lower) {
				return true
			}
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), lower)
}

func FamilyIs(err error, family errorfamily.Family) bool {
	return errorfamily.Classify(err) == family
}

func ErrorCodeContains(err error, substr string) bool {
	if coded, ok := errors.AsType[errorfamily.Coded](err); ok {
		return strings.Contains(strings.ToLower(coded.ErrorCode()), strings.ToLower(substr))
	}
	return false
}

// RuleSpec declares the matching criteria for a diagnostic rule.
type RuleSpec struct {
	ContextKeys   []ContextKey
	CodeContains  []string
	ContextSubstr []string
	Extra         func(error) bool
}

// Matches reports whether the error matches this spec's criteria.
func (s RuleSpec) Matches(err error) bool {
	if len(s.ContextKeys) > 0 {
		keys := make([]string, len(s.ContextKeys))
		for i, k := range s.ContextKeys {
			keys[i] = string(k)
		}
		if HasContextKey(err, keys...) {
			return true
		}
	}
	for _, substr := range s.CodeContains {
		if ErrorCodeContains(err, substr) {
			return true
		}
	}
	for _, substr := range s.ContextSubstr {
		if HasContextSubstring(err, substr) {
			return true
		}
	}
	if s.Extra != nil && s.Extra(err) {
		return true
	}
	return false
}
