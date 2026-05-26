package diagnose

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func BenchmarkRunnerRunNoRules(b *testing.B) {
	runner := NewRunner()
	err := errorfamily.NewTransient("test", "msg")
	ctx := context.Background()
	for b.Loop() {
		_ = runner.Run(ctx, err)
	}
}

func BenchmarkRunnerRunOneRule(b *testing.B) {
	rule := &staticRule{
		name:       "test",
		applicable: true,
		result:     &DiagnosticResult{Status: StatusHealthy, Confidence: 0.5},
	}
	runner := NewRunner(rule)
	err := errorfamily.NewTransient("test", "msg")
	ctx := context.Background()
	for b.Loop() {
		_ = runner.Run(ctx, err)
	}
}

func BenchmarkRunnerRunThreeRules(b *testing.B) {
	rules := []DiagnosticRule{
		&staticRule{
			name:       "low",
			applicable: true,
			result:     &DiagnosticResult{Status: StatusHealthy, Confidence: 0.3},
		},
		&staticRule{
			name:       "mid",
			applicable: true,
			result:     &DiagnosticResult{Status: StatusDegraded, Confidence: 0.6},
		},
		&staticRule{
			name:       "high",
			applicable: true,
			result:     &DiagnosticResult{Status: StatusFailed, Confidence: 0.9},
		},
	}
	runner := NewRunner(rules...)
	err := errorfamily.NewTransient("test", "msg")
	ctx := context.Background()
	for b.Loop() {
		_ = runner.Run(ctx, err)
	}
}

func BenchmarkRuleSpecMatches(b *testing.B) {
	spec := RuleSpec{
		ContextKeys:   []string{"host", "port"},
		CodeContains:  []string{"db."},
		ContextSubstr: []string{"postgres"},
	}
	err := errorfamily.NewTransient("db.timeout", "msg").
		WithContext("host", "localhost").
		WithContext("port", "5432")
	for b.Loop() {
		_ = spec.Matches(err)
	}
}

func BenchmarkDefaultRunner(b *testing.B) {
	runner := DefaultRunner()
	err := errorfamily.NewTransient("network.connect", "msg").
		WithContext("host", "example.com").
		WithContext("port", "443")
	ctx := context.Background()
	for b.Loop() {
		_ = runner.Run(ctx, err)
	}
}
