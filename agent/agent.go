// Package agent provides an AI-powered debug agent that can analyze errors,
// run diagnostics, and suggest or apply fixes with configurable user involvement.
//
// The agent operates at four involvement levels:
//   - Silent:     Analyzes and logs, no user interaction
//   - Suggest:    Suggests fixes, user must approve each step
//   - Assist:     Applies safe fixes automatically, asks for risky ones
//   - Autonomous: Applies all fixes without asking
//
// Usage:
//
//	cfg := agent.Config{
//	    Involvement: agent.InvolvementSuggest,
//	    Model:       "gpt-4",
//	}
//	ag := agent.New(cfg)
//	result, err := ag.Analyze(ctx, err, diagnosis)
package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

// Involvement controls how much the AI agent can do without user approval.
type Involvement int

const (
	// InvolvementSilent means the agent runs analysis but only logs results.
	// No user interaction. Useful for CI/CD pipelines and background services.
	InvolvementSilent Involvement = iota

	// InvolvementSuggest means the agent suggests fixes but the user must approve each step.
	// The agent will NOT execute any commands or modify any state.
	InvolvementSuggest

	// InvolvementAssist means the agent can apply safe (low-risk) fixes automatically
	// but asks for user confirmation before risky operations.
	InvolvementAssist

	// InvolvementAutonomous means the agent applies all fixes without asking.
	// DANGEROUS in production. Use only in development/testing environments.
	InvolvementAutonomous
)

func (i Involvement) String() string {
	switch i {
	case InvolvementSilent:
		return "silent"
	case InvolvementSuggest:
		return "suggest"
	case InvolvementAssist:
		return "assist"
	case InvolvementAutonomous:
		return "autonomous"
	default:
		return "unknown"
	}
}

// RiskLevel classifies how risky a fix step is.
type RiskLevel int

const (
	// RiskSafe means the fix is read-only or trivially reversible.
	// Example: creating a directory, setting a config value.
	RiskSafe RiskLevel = iota

	// RiskMedium means the fix changes state but is reversible.
	// Example: git commit, restarting a service.
	RiskMedium

	// RiskHigh means the fix is destructive or hard to reverse.
	// Example: deleting files, force-pushing, dropping database tables.
	RiskHigh
)

func (r RiskLevel) String() string {
	switch r {
	case RiskSafe:
		return "safe"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	default:
		return "unknown"
	}
}

// Config controls the behavior of the debug agent.
type Config struct {
	// Involvement controls how much the agent can do without user approval.
	Involvement Involvement

	// Enabled controls whether the agent is active at all.
	Enabled bool

	// Model is the AI model to use (e.g., "gpt-4", "claude-3-opus").
	// Empty means use the default model from the provider.
	Model string

	// MaxTokens limits the response size from the AI.
	MaxTokens int

	// Timeout is the maximum time the agent will spend analyzing an error.
	Timeout time.Duration

	// MaxRetries is how many times the agent will retry a failed fix step.
	MaxRetries int

	// AllowedCommands controls what shell commands the agent can run.
	// Glob patterns: "git *", "mkdir *", "chmod *".
	// Empty means no commands are allowed.
	AllowedCommands []string

	// ForbiddenCommands are commands the agent must NEVER run, regardless of involvement level.
	// Default: "rm *", "drop *", "delete *", "format *", "shutdown *", "reboot *"
	ForbiddenCommands []string

	// SystemPrompt overrides the default system prompt for the AI.
	SystemPrompt string

	// ConfirmFunc is called when the agent needs user confirmation.
	// Receives the proposed action. Returns true to proceed, false to skip.
	// If nil and Involvement is Suggest, all actions are skipped.
	ConfirmFunc func(action string) bool
}

// DefaultConfig returns a safe default configuration.
func DefaultConfig() Config {
	return Config{
		Involvement: InvolvementSuggest,
		Enabled:     false,
		MaxTokens:   4096,
		Timeout:     60 * time.Second,
		MaxRetries:  1,
		AllowedCommands: []string{
			"git status*", "git diff*", "git log*",
			"pg_isready*", "ls*", "cat*",
			"mkdir *", "chmod *",
			"nc -zv*", "dig*", "ping*",
		},
		ForbiddenCommands: []string{
			"rm *", "rmdir *", "drop *", "delete *",
			"format *", "shutdown *", "reboot *",
			"dd *", "mkfs *",
		},
	}
}

// AgentResult holds the AI agent's analysis of an error.
type AgentResult struct {
	// RootCause is the agent's assessment of what caused the error.
	RootCause string

	// Confidence is 0.0–1.0 indicating how sure the agent is.
	Confidence float64

	// Explanation is a human-readable explanation of the error chain.
	Explanation string

	// FixSteps are ordered steps to resolve the error.
	FixSteps []FixStep

	// Prevention describes how to prevent this error in the future.
	Prevention string

	// RelatedErrors lists error codes that commonly co-occur.
	RelatedErrors []string

	// AnalysisTime is how long the agent spent analyzing.
	AnalysisTime time.Duration

	// ModelUsed is which AI model produced this result.
	ModelUsed string

	// TokensUsed is the total token count for the analysis.
	TokensUsed int
}

// FixStep describes a single action to resolve an error.
type FixStep struct {
	// Description is what this step does in plain language.
	Description string

	// Command is the shell command to execute, if applicable.
	Command string

	// Risk is the risk level of this step.
	Risk RiskLevel

	// AutoApply is true if the agent determined this step can be applied automatically.
	AutoApply bool

	// Rationale explains WHY this step is needed.
	Rationale string

	// Applied is true if the step was actually executed.
	Applied bool

	// Output is the command output if the step was applied.
	Output string
}

// DebugAgent analyzes errors using AI and diagnostic context.
type DebugAgent interface {
	// Analyze examines an error with diagnostic context and produces
	// a root cause analysis with fix suggestions.
	Analyze(ctx context.Context, err error, diagnosis []*diagnose.DiagnosticResult) (*AgentResult, error)

	// ApplyFixes executes fix steps according to the configured involvement level.
	// Returns the steps that were actually applied.
	ApplyFixes(ctx context.Context, result *AgentResult) []FixStep
}

// New creates a new debug agent with the given configuration.
// The agent is disabled by default — set cfg.Enabled = true to activate.
func New(cfg Config) DebugAgent {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 1
	}
	return &agent{cfg: cfg}
}

type agent struct {
	cfg Config
}

func (a *agent) Analyze(ctx context.Context, err error, diagnosis []*diagnose.DiagnosticResult) (*AgentResult, error) {
	if !a.cfg.Enabled {
		return &AgentResult{
			RootCause:   "agent disabled",
			Confidence:  0,
			Explanation: "AI debug agent is not enabled. Enable via agent.Config{Enabled: true}.",
		}, nil
	}

	// Build the analysis prompt from error and diagnostic context.
	prompt := a.buildPrompt(err, diagnosis)

	// In a real implementation, this would call the AI provider.
	// For now, this is a scaffold that returns deterministic analysis
	// based on the diagnostic results.
	return a.deterministicAnalyze(err, diagnosis, prompt)
}

func (a *agent) ApplyFixes(ctx context.Context, result *AgentResult) []FixStep {
	var applied []FixStep

	for i := range result.FixSteps {
		step := &result.FixSteps[i]

		if !a.shouldApply(step) {
			continue
		}

		// In a real implementation, this would execute the command
		// via a sandboxed command runner with the allowed/forbidden lists.
		step.Applied = true
		applied = append(applied, *step)
	}

	return applied
}

func (a *agent) shouldApply(step *FixStep) bool {
	switch a.cfg.Involvement {
	case InvolvementSilent:
		return false
	case InvolvementSuggest:
		if a.cfg.ConfirmFunc != nil {
			return a.cfg.ConfirmFunc(fmt.Sprintf("%s\n  Command: %s\n  Risk: %s", step.Description, step.Command, step.Risk))
		}
		return false
	case InvolvementAssist:
		if step.Risk == RiskSafe {
			return true
		}
		if a.cfg.ConfirmFunc != nil {
			return a.cfg.ConfirmFunc(fmt.Sprintf("%s\n  Command: %s\n  Risk: %s", step.Description, step.Command, step.Risk))
		}
		return false
	case InvolvementAutonomous:
		return true
	default:
		return false
	}
}

// deterministicAnalyze provides rule-based analysis without an AI provider.
// This runs when no AI provider is configured, using diagnostic results
// to produce structured fix suggestions.
func (a *agent) deterministicAnalyze(err error, diagnosis []*diagnose.DiagnosticResult, _ string) (*AgentResult, error) {
	result := &AgentResult{
		Confidence: 0.5,
	}

	// Build explanation from diagnosis.
	var parts []string
	for _, d := range diagnosis {
		if d.Status == diagnose.StatusFailed {
			result.Confidence = max(result.Confidence, d.Confidence)
			parts = append(parts, d.Summary)
			if d.SuggestedFix != "" {
				result.FixSteps = append(result.FixSteps, FixStep{
					Description: d.Summary,
					Command:     extractCommand(d.SuggestedFix),
					Risk:        RiskSafe,
					Rationale:   fmt.Sprintf("Diagnostic rule '%s' identified this issue", d.RuleName),
				})
			}
		}
	}

	if len(parts) > 0 {
		result.RootCause = parts[0]
		result.Explanation = strings.Join(parts, "; ")
	} else {
		result.RootCause = "no specific root cause identified"
		result.Explanation = fmt.Sprintf("Error classified as %s. No diagnostic failures found.", errorfamily.Classify(err))
	}

	return result, nil
}

func extractCommand(suggest string) string {
	for line := range strings.SplitSeq(suggest, "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "$ "); ok {
			return after
		}
		if after, ok := strings.CutPrefix(line, "Run: "); ok {
			return after
		}
	}
	return ""
}

func (a *agent) buildPrompt(err error, diagnosis []*diagnose.DiagnosticResult) string {
	var b strings.Builder

	b.WriteString("Analyze the following error and provide root cause analysis with fix steps.\n\n")
	fmt.Fprintf(&b, "Error: %s\n", err.Error())
	fmt.Fprintf(&b, "Family: %s\n", errorfamily.Classify(err))

	if ctx, ok := errors.AsType[errorfamily.Contextual](err); ok {
		b.WriteString("Context:\n")
		for k, v := range ctx.ErrorContext() {
			fmt.Fprintf(&b, "  %s: %s\n", k, v)
		}
	}

	if len(diagnosis) > 0 {
		b.WriteString("\nDiagnostic Results:\n")
		for _, d := range diagnosis {
			fmt.Fprintf(&b, "  [%s] %s (confidence: %.1f)\n", d.Status, d.Summary, d.Confidence)
			if d.SuggestedFix != "" {
				fmt.Fprintf(&b, "    Suggested fix: %s\n", d.SuggestedFix)
			}
		}
	}

	return b.String()
}
