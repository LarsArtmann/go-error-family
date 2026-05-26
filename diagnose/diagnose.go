// Package diagnose provides automatic error context discovery and deterministic
// diagnostic rules. When an error occurs, the diagnostic system can:
//
//  1. Auto-discover system context (OS, disk, memory, network state)
//  2. Run deterministic diagnostic rules matching the error
//  3. Suggest specific fixes or auto-fix when possible
//  4. Feed results to an AI agent for deeper analysis
//
// Usage:
//
//	diagnosis := diagnose.Run(ctx, err)
//	for _, result := range diagnosis {
//	    fmt.Println(result.Summary)
//	    if result.SuggestedFix != "" {
//	        fmt.Println("  Fix:", result.SuggestedFix)
//	    }
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

// Helper functions for rule matching.

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
	ContextKeys   []string
	CodeContains  []string
	ContextSubstr []string
	Extra         func(error) bool
}

// Matches reports whether the error matches this spec's criteria.
func (s RuleSpec) Matches(err error) bool {
	if len(s.ContextKeys) > 0 && HasContextKey(err, s.ContextKeys...) {
		return true
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
