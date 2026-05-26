package diagnose

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusHealthy, "healthy"},
		{StatusDegraded, "degraded"},
		{StatusFailed, "failed"},
		{StatusUnknown, "unknown"},
		{Status(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestRunnerNoRules(t *testing.T) {
	runner := NewRunner()
	err := errorfamily.NewTransient("test", "msg")
	results := runner.Run(context.Background(), err)
	if results != nil {
		t.Error("Expected nil results with no rules")
	}
}

func TestRunnerRegister(t *testing.T) {
	runner := NewRunner()
	rule := &staticRule{name: "test", applicable: true, result: &DiagnosticResult{Status: StatusHealthy}}
	runner.Register(rule)

	err := errorfamily.NewTransient("test", "msg")
	results := runner.Run(context.Background(), err)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].RuleName != "test" {
		t.Errorf("RuleName = %q, want 'test'", results[0].RuleName)
	}
}

func TestRunnerFiltersInapplicable(t *testing.T) {
	rule := &staticRule{name: "never", applicable: false, result: &DiagnosticResult{Status: StatusHealthy}}
	runner := NewRunner(rule)

	err := errorfamily.NewTransient("test", "msg")
	results := runner.Run(context.Background(), err)
	if results != nil {
		t.Error("Expected nil results when rule is not applicable")
	}
}

func TestRunnerSortsByConfidence(t *testing.T) {
	rule1 := &staticRule{
		name:       "low",
		applicable: true,
		result:     &DiagnosticResult{Status: StatusHealthy, Confidence: 0.3},
	}
	rule2 := &staticRule{
		name:       "high",
		applicable: true,
		result:     &DiagnosticResult{Status: StatusFailed, Confidence: 0.9},
	}
	rule3 := &staticRule{
		name:       "mid",
		applicable: true,
		result:     &DiagnosticResult{Status: StatusDegraded, Confidence: 0.6},
	}

	runner := NewRunner(rule1, rule2, rule3)
	err := errorfamily.NewTransient("test", "msg")
	results := runner.Run(context.Background(), err)

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}
	if results[0].Confidence < results[1].Confidence {
		t.Errorf("Results not sorted by confidence: %v >= %v >= %v",
			results[0].Confidence, results[1].Confidence, results[2].Confidence)
	}
}

func TestRunnerHandlesError(t *testing.T) {
	rule := &staticRule{
		name:       "failing",
		applicable: true,
		runErr:     errors.New("something broke"),
	}
	runner := NewRunner(rule)

	err := errorfamily.NewTransient("test", "msg")
	results := runner.Run(context.Background(), err)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusUnknown {
		t.Errorf("Status = %v, want StatusUnknown", results[0].Status)
	}
	if results[0].RuleName != "failing" {
		t.Errorf("RuleName = %q, want 'failing'", results[0].RuleName)
	}
}

func TestRunnerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rule := &staticRule{name: "cancelled", applicable: true, result: &DiagnosticResult{Status: StatusHealthy}}
	runner := NewRunner(rule)

	results := runner.Run(ctx, errorfamily.NewTransient("test", "msg"))
	// Context is cancelled but the rule still runs (it doesn't check ctx).
	// This test verifies no panic or deadlock.
	_ = results
}

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

func TestNetworkRuleApplicable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"host context", errorfamily.NewTransient("test", "msg").WithContext("host", "example.com"), true},
		{"connect code", errorfamily.NewTransient("network.connect", "msg"), true},
		{"timeout code", errorfamily.NewTransient("timeout", "msg"), true},
		{"unrelated", errorfamily.NewRejection("file.not_found", "msg"), false},
		{"connection refused substring", errorfamily.NewTransient("test", "connection refused"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &NetworkRule{}
			if got := r.Applicable(tt.err); got != tt.want {
				t.Errorf("Applicable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilesystemRuleApplicable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"path context", errorfamily.NewRejection("test", "msg").WithContext("path", "/etc/config"), true},
		{"file code", errorfamily.NewRejection("file.not_found", "msg"), true},
		{"config code", errorfamily.NewRejection("config.invalid", "msg"), true},
		{"unrelated", errorfamily.NewTransient("db.timeout", "msg"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &FilesystemRule{}
			if got := r.Applicable(tt.err); got != tt.want {
				t.Errorf("Applicable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunAuto(t *testing.T) {
	err := errorfamily.NewTransient("test", "msg").WithContext("host", "example.com")
	results := RunAuto(context.Background(), err)
	// NetworkRule should match because of host context
	// but actual run depends on system state — just verify no panic.
	_ = results
}

func TestDefaultRunner(t *testing.T) {
	runner := DefaultRunner()
	if runner == nil {
		t.Fatal("DefaultRunner() returned nil")
	}
}

func TestDiagnosticResultDuration(t *testing.T) {
	runner := NewRunner(&slowRule{name: "slow", duration: 50 * time.Millisecond})
	err := errorfamily.NewTransient("test", "msg")
	results := runner.Run(context.Background(), err)
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Duration < 40*time.Millisecond {
		t.Errorf("Duration = %v, expected at least 40ms", results[0].Duration)
	}
}

func TestFilesystemRuleRunExistingFile(t *testing.T) {
	r := &FilesystemRule{}
	err := errorfamily.NewRejection("file.not_found", "msg").WithContext("path", "/etc/hostname")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["exists"] != "true" {
		t.Errorf("Expected file to exist, got details: %v", result.Details)
	}
}

func TestFilesystemRuleRunNonexistentPath(t *testing.T) {
	r := &FilesystemRule{}
	err := errorfamily.NewRejection("file.not_found", "msg").
		WithContext("path", "/nonexistent/path/that/does/not/exist")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != StatusFailed {
		t.Errorf("Status = %v, want StatusFailed", result.Status)
	}
	if result.Details["exists"] != "false" {
		t.Errorf("Expected exists=false, got %v", result.Details)
	}
}

func TestFilesystemRuleRunNoPath(t *testing.T) {
	r := &FilesystemRule{}
	// Force Applicable to return true but with no path in context
	err := errorfamily.NewRejection("file.error", "msg")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != StatusUnknown {
		t.Errorf("Status = %v, want StatusUnknown", result.Status)
	}
}

func TestNetworkRuleRunNoHost(t *testing.T) {
	r := &NetworkRule{}
	// Applicable returns true for timeout code, but no host in context
	err := errorfamily.NewTransient("timeout", "msg")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	// No host means DNS check on empty string — should handle gracefully
	_ = result
}

func TestNetworkRuleResolveHostWithURL(t *testing.T) {
	r := &NetworkRule{}
	err := errorfamily.NewTransient("test", "msg").WithContext("host", "https://example.com:8080/path")
	if host := r.resolveHost(err); host != "example.com" {
		t.Errorf("resolveHost with URL = %q, want 'example.com'", host)
	}
}

func TestParentDir(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/a/b/c", "/a/b"},
		{"/a/b", "/a"},
		{"/a", "/"},
		{"relative/path", "relative"},
		{"nopath", "."},
	}
	for _, tt := range tests {
		if got := filepath.Dir(tt.path); got != tt.want {
			t.Errorf("filepath.Dir(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

type staticRule struct {
	name       string
	applicable bool
	result     *DiagnosticResult
	runErr     error
}

func (r *staticRule) Name() string            { return r.name }
func (r *staticRule) Applicable(_ error) bool { return r.applicable }
func (r *staticRule) Run(_ context.Context, _ error) (*DiagnosticResult, error) {
	return r.result, r.runErr
}

type slowRule struct {
	name     string
	duration time.Duration
}

func (r *slowRule) Name() string            { return r.name }
func (r *slowRule) Applicable(_ error) bool { return true }
func (r *slowRule) Run(ctx context.Context, _ error) (*DiagnosticResult, error) {
	select {
	case <-time.After(r.duration):
	case <-ctx.Done():
	}
	return &DiagnosticResult{Status: StatusHealthy, Confidence: 0.5}, nil
}
