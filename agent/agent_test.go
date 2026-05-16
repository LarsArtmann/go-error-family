package agent

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

func TestNewAgentDefaults(t *testing.T) {
	cfg := Config{Enabled: true}
	ag := New(cfg)
	if ag == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNewAgentZeroTimeout(t *testing.T) {
	cfg := Config{Enabled: true}
	ag := New(cfg)
	result, err := ag.Analyze(context.Background(), errorfamily.NewTransient("test", "msg"), nil)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}
	if result.RootCause == "" {
		t.Error("Expected non-empty RootCause")
	}
}

func TestAnalyzeDisabled(t *testing.T) {
	cfg := Config{Enabled: false}
	ag := New(cfg)

	result, err := ag.Analyze(context.Background(), errorfamily.NewTransient("test", "msg"), nil)
	if err != nil {
		t.Fatalf("Analyze() error: %v", err)
	}
	if result.Confidence != 0 {
		t.Errorf("Confidence = %v, want 0 for disabled agent", result.Confidence)
	}
	if result.RootCause != "agent disabled" {
		t.Errorf("RootCause = %q, want 'agent disabled'", result.RootCause)
	}
}

func TestAnalyzeWithDiagnosis(t *testing.T) {
	cfg := Config{Enabled: true}
	ag := New(cfg)

	err := errorfamily.NewTransient("db.timeout", "connection refused")
	diagnosis := []*diagnose.DiagnosticResult{
		{
			Status:       diagnose.StatusFailed,
			Summary:      "PostgreSQL is NOT responding",
			SuggestedFix: "Start PostgreSQL: brew services start postgresql",
			Confidence:   0.9,
		},
	}

	result, analyzeErr := ag.Analyze(context.Background(), err, diagnosis)
	if analyzeErr != nil {
		t.Fatalf("Analyze() error: %v", analyzeErr)
	}
	if result.Confidence < 0.9 {
		t.Errorf("Confidence = %v, expected at least 0.9", result.Confidence)
	}
	if len(result.FixSteps) == 0 {
		t.Error("Expected at least one FixStep from failed diagnosis")
	}
	if result.RootCause == "" {
		t.Error("Expected non-empty RootCause")
	}
}

func TestAnalyzeNoFailures(t *testing.T) {
	cfg := Config{Enabled: true}
	ag := New(cfg)

	err := errorfamily.NewTransient("test", "msg")
	diagnosis := []*diagnose.DiagnosticResult{
		{Status: diagnose.StatusHealthy, Summary: "All good", Confidence: 0.3},
	}

	result, analyzeErr := ag.Analyze(context.Background(), err, diagnosis)
	if analyzeErr != nil {
		t.Fatalf("Analyze() error: %v", analyzeErr)
	}
	if result.RootCause != "no specific root cause identified" {
		t.Errorf("RootCause = %q, want 'no specific root cause identified'", result.RootCause)
	}
}

func TestAnalyzeEmptyDiagnosis(t *testing.T) {
	cfg := Config{Enabled: true}
	ag := New(cfg)

	err := errorfamily.NewTransient("test", "msg")
	result, analyzeErr := ag.Analyze(context.Background(), err, nil)
	if analyzeErr != nil {
		t.Fatalf("Analyze() error: %v", analyzeErr)
	}
	if result.Confidence != 0.5 {
		t.Errorf("Confidence = %v, want 0.5 for empty diagnosis", result.Confidence)
	}
}

func TestAnalyzeWithContext(t *testing.T) {
	cfg := Config{Enabled: true}
	ag := New(cfg)

	err := errorfamily.NewTransient("db.timeout", "connection refused").
		WithContext("host", "localhost").
		WithContext("port", "5432")

	diagnosis := []*diagnose.DiagnosticResult{
		{Status: diagnose.StatusFailed, Summary: "Cannot connect", Confidence: 0.9, SuggestedFix: "Check connection"},
	}

	result, analyzeErr := ag.Analyze(context.Background(), err, diagnosis)
	if analyzeErr != nil {
		t.Fatalf("Analyze() error: %v", analyzeErr)
	}
	_ = result
}

func TestExtractCommand(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"$ brew services start postgresql", "brew services start postgresql"},
		{"Run: git status", "git status"},
		{"Some text\n$ actual command", "actual command"},
		{"no command here", ""},
	}
	for _, tt := range tests {
		if got := extractCommand(tt.input); got != tt.want {
			t.Errorf("extractCommand(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
