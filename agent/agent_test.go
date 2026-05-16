package agent

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

func TestInvolvementString(t *testing.T) {
	tests := []struct {
		inv  Involvement
		want string
	}{
		{InvolvementSilent, "silent"},
		{InvolvementSuggest, "suggest"},
		{InvolvementAssist, "assist"},
		{InvolvementAutonomous, "autonomous"},
		{Involvement(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.inv.String(); got != tt.want {
			t.Errorf("Involvement(%d).String() = %q, want %q", tt.inv, got, tt.want)
		}
	}
}

func TestRiskLevelString(t *testing.T) {
	tests := []struct {
		risk RiskLevel
		want string
	}{
		{RiskSafe, "safe"},
		{RiskMedium, "medium"},
		{RiskHigh, "high"},
		{RiskLevel(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.risk.String(); got != tt.want {
			t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.risk, got, tt.want)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Error("DefaultConfig should have Enabled=false")
	}
	if cfg.Involvement != InvolvementSuggest {
		t.Error("DefaultConfig should have InvolvementSuggest")
	}
	if cfg.Timeout == 0 {
		t.Error("DefaultConfig should have non-zero Timeout")
	}
	if cfg.MaxTokens == 0 {
		t.Error("DefaultConfig should have non-zero MaxTokens")
	}
	if len(cfg.ForbiddenCommands) == 0 {
		t.Error("DefaultConfig should have ForbiddenCommands")
	}
}

func TestNewAgentDefaults(t *testing.T) {
	cfg := Config{Enabled: true}
	ag := New(cfg)
	if ag == nil {
		t.Fatal("New() returned nil")
	}
}

func TestNewAgentZeroDefaults(t *testing.T) {
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

func TestApplyFixesSilent(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Involvement: InvolvementSilent,
	}
	ag := New(cfg)

	result := &AgentResult{
		FixSteps: []FixStep{
			{Description: "Do something", Risk: RiskSafe},
		},
	}
	applied := ag.ApplyFixes(context.Background(), result)
	if len(applied) != 0 {
		t.Error("Silent mode should apply nothing")
	}
}

func TestApplyFixesSuggestWithApproval(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Involvement: InvolvementSuggest,
		ConfirmFunc: func(action string) bool { return true },
	}
	ag := New(cfg)

	result := &AgentResult{
		FixSteps: []FixStep{
			{Description: "Do something", Risk: RiskSafe},
		},
	}
	applied := ag.ApplyFixes(context.Background(), result)
	if len(applied) != 1 {
		t.Errorf("Expected 1 applied step with approval, got %d", len(applied))
	}
}

func TestApplyFixesSuggestWithoutConfirm(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Involvement: InvolvementSuggest,
	}
	ag := New(cfg)

	result := &AgentResult{
		FixSteps: []FixStep{
			{Description: "Do something", Risk: RiskSafe},
		},
	}
	applied := ag.ApplyFixes(context.Background(), result)
	if len(applied) != 0 {
		t.Error("Suggest without ConfirmFunc should apply nothing")
	}
}

func TestApplyFixesAssistSafe(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Involvement: InvolvementAssist,
	}
	ag := New(cfg)

	result := &AgentResult{
		FixSteps: []FixStep{
			{Description: "Safe fix", Risk: RiskSafe},
		},
	}
	applied := ag.ApplyFixes(context.Background(), result)
	if len(applied) != 1 {
		t.Error("Assist mode should auto-apply safe fixes")
	}
}

func TestApplyFixesAssistRisky(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Involvement: InvolvementAssist,
		ConfirmFunc: func(action string) bool { return false },
	}
	ag := New(cfg)

	result := &AgentResult{
		FixSteps: []FixStep{
			{Description: "Risky fix", Risk: RiskHigh},
		},
	}
	applied := ag.ApplyFixes(context.Background(), result)
	if len(applied) != 0 {
		t.Error("Assist mode should not auto-apply risky fixes when denied")
	}
}

func TestApplyFixesAutonomous(t *testing.T) {
	cfg := Config{
		Enabled:     true,
		Involvement: InvolvementAutonomous,
	}
	ag := New(cfg)

	result := &AgentResult{
		FixSteps: []FixStep{
			{Description: "Safe", Risk: RiskSafe},
			{Description: "Risky", Risk: RiskHigh},
		},
	}
	applied := ag.ApplyFixes(context.Background(), result)
	if len(applied) != 2 {
		t.Errorf("Autonomous mode should apply all fixes, got %d", len(applied))
	}
}

func TestBuildPrompt(t *testing.T) {
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
