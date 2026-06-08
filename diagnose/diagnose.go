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
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

// ContextKey is a typed string for error context keys used by diagnostic rules.
// Use these constants instead of raw strings to prevent typos and enable IDE autocompletion.
type ContextKey string

const (
	KeyHost         ContextKey = "host"
	KeyPort         ContextKey = "port"
	KeyPath         ContextKey = "path"
	KeyFile         ContextKey = "file"
	KeyDir          ContextKey = "dir"
	KeyURL          ContextKey = "url"
	KeyEndpoint     ContextKey = "endpoint"
	KeyAddress      ContextKey = "address"
	KeyRemote       ContextKey = "remote"
	KeyDBHost       ContextKey = "db_host"
	KeyDBPort       ContextKey = "db_port"
	KeyDBName       ContextKey = "db_name"
	KeyDatabaseURL  ContextKey = "database_url"
	KeyPostgresHost ContextKey = "postgres_host"
	KeyConfigPath   ContextKey = "config_path"
	KeyOutputPath   ContextKey = "output_path"
	KeyDirectory    ContextKey = "directory"
	KeyGit          ContextKey = "git"
	KeyRepository   ContextKey = "repository"
	KeyRepo         ContextKey = "repo"
	KeyBranch       ContextKey = "branch"
	KeyGitDir       ContextKey = "git_dir"
	KeyRepoPath     ContextKey = "repo_path"
	KeyPostgresPort ContextKey = "postgres_port"
	KeyPGHOST       ContextKey = "PGHOST"
	KeyPGPORT       ContextKey = "PGPORT"
)

// Status represents the result of a diagnostic check.
type Status int

const (
	StatusHealthy Status = iota
	StatusDegraded
	StatusFailed
	StatusUnknown
)

func (s Status) String() string {
	if s.IsValid() {
		return statusNames[s]
	}
	return strUnknown
}

// IsValid reports whether the Status value is one of the four defined constants.
func (s Status) IsValid() bool {
	return s >= StatusHealthy && s <= StatusUnknown
}

// ParseStatus parses a status string, case-insensitive.
// Returns StatusUnknown for unrecognized values.
func ParseStatus(s string) Status {
	lower := strings.ToLower(s)
	for st, name := range statusNames {
		if name == lower {
			return st
		}
	}
	return StatusUnknown
}

var statusNames = map[Status]string{ //nolint:gochecknoglobals // Immutable lookup table.
	StatusHealthy:  "healthy",
	StatusDegraded: "degraded",
	StatusFailed:   "failed",
	StatusUnknown:  strUnknown,
}

// DiagnosticResult holds the outcome of a single diagnostic check.
type DiagnosticResult struct {
	RuleName     string
	Status       Status
	Summary      string
	Details      map[string]string
	Context      map[string]string
	SuggestedFix string
	Confidence   float64
	Duration     time.Duration
}

// DiagnosticRule is the interface for deterministic error diagnostic checks.
type DiagnosticRule interface {
	Name() string
	Applicable(err error) bool
	Run(ctx context.Context, err error) (*DiagnosticResult, error)
}

// Runner executes diagnostic rules against errors.
type Runner struct {
	rules []DiagnosticRule
	mu    sync.RWMutex
}

func NewRunner(rules ...DiagnosticRule) *Runner {
	return &Runner{rules: rules}
}

func (r *Runner) Register(rule DiagnosticRule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, rule)
}

// Run executes all applicable diagnostic rules for the given error.
// Rules are run concurrently for speed. Results are ordered by confidence (highest first).
// Context cancellation is respected: if ctx is cancelled, Run returns early with
// whatever results have been collected so far.
func (r *Runner) Run(ctx context.Context, err error) []*DiagnosticResult {
	rules := r.applicableRules(err)
	if len(rules) == 0 {
		return nil
	}

	results := r.runRules(ctx, err, rules)
	return sortByConfidence(results)
}

func (r *Runner) applicableRules(err error) []DiagnosticRule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rules := make([]DiagnosticRule, 0, len(r.rules))
	for _, rule := range r.rules {
		if rule.Applicable(err) {
			rules = append(rules, rule)
		}
	}
	return rules
}

func (r *Runner) runRules(ctx context.Context, err error, rules []DiagnosticRule) []*DiagnosticResult {
	type ruleResult struct {
		idx    int
		result *DiagnosticResult
	}

	ch := make(chan ruleResult, len(rules))
	for i, rule := range rules {
		go func(idx int, rl DiagnosticRule) {
			start := time.Now()
			result, runErr := rl.Run(ctx, err)
			if runErr != nil {
				ch <- ruleResult{idx: idx, result: &DiagnosticResult{
					RuleName: rl.Name(),
					Status:   StatusUnknown,
					Summary:  fmt.Sprintf("diagnostic failed: %v", runErr),
					Details:  map[string]string{"error": runErr.Error()},
				}}
			} else if result != nil {
				result.RuleName = rl.Name()
				result.Duration = time.Since(start)
				ch <- ruleResult{idx: idx, result: result}
			} else {
				ch <- ruleResult{idx: idx}
			}
		}(i, rule)
	}

	results := make([]*DiagnosticResult, len(rules))
	collected := 0
	for collected < len(rules) {
		select {
		case rr := <-ch:
			results[rr.idx] = rr.result
			collected++
		case <-ctx.Done():
			return results
		}
	}
	return results
}

func sortByConfidence(results []*DiagnosticResult) []*DiagnosticResult {
	filtered := make([]*DiagnosticResult, 0, len(results))
	for _, res := range results {
		if res != nil {
			filtered = append(filtered, res)
		}
	}

	slices.SortFunc(filtered, func(a, b *DiagnosticResult) int {
		switch {
		case a.Confidence > b.Confidence:
			return -1
		case a.Confidence < b.Confidence:
			return 1
		default:
			return 0
		}
	})

	return filtered
}

func RunAuto(ctx context.Context, err error) []*DiagnosticResult {
	return DefaultRunner().Run(ctx, err)
}

func DefaultRunner() *Runner {
	return NewRunner(
		&FilesystemRule{},
		&NetworkRule{},
	)
}

const (
	ConfidenceNone     float64 = 0.1
	ConfidenceNotCause float64 = 0.3
	ConfidencePartial  float64 = 0.4
	ConfidenceLikely   float64 = 0.7
	ConfidenceHigh     float64 = 0.8
	ConfidenceVeryHigh float64 = 0.85
	ConfidenceCertain  float64 = 0.9
)

type CommandRunner interface {
	Run(ctx context.Context, timeout time.Duration, name string, args ...string) (string, int, error)
	Exists(name string) bool
}

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
