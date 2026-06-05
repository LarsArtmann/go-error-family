package diagnose

import (
	"context"
	"errors"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

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
	rule := &staticRule{
		name:       "test",
		applicable: true,
		result:     &DiagnosticResult{Status: StatusHealthy},
	}
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
	rule := &staticRule{
		name:       "never",
		applicable: false,
		result:     &DiagnosticResult{Status: StatusHealthy},
	}
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

	rule := &staticRule{
		name:       "cancelled",
		applicable: true,
		result:     &DiagnosticResult{Status: StatusHealthy},
	}
	runner := NewRunner(rule)

	results := runner.Run(ctx, errorfamily.NewTransient("test", "msg"))
	_ = results
}

func TestRunnerContextCancelledMidRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	fastRule := &staticRule{
		name:       "fast",
		applicable: true,
		result:     &DiagnosticResult{Status: StatusHealthy, Confidence: 0.5},
	}
	slowRule := &slowRule{
		name:     "slow",
		duration: 100 * time.Millisecond,
	}

	runner := NewRunner(fastRule, slowRule)

	time.AfterFunc(10*time.Millisecond, cancel)

	start := time.Now()
	results := runner.Run(ctx, errorfamily.NewTransient("test", "msg"))
	elapsed := time.Since(start)

	if elapsed > 80*time.Millisecond {
		t.Errorf("Run took %v, should have returned quickly after cancellation", elapsed)
	}

	_ = results
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
