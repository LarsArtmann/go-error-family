// Package agent provides an error analysis agent that uses diagnostic context
// to produce root cause analysis and fix suggestions.
//
// The agent proposes fixes but does NOT execute them. The consumer decides
// what to do with the suggested FixSteps.
//
// Usage:
//
//	ag := agent.New(agent.Config{Enabled: true})
//	result, err := ag.Analyze(ctx, err, diagnosis)
//	for _, step := range result.FixSteps {
//	    fmt.Printf("  - %s\n", step.Description)
//	}
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

// Config controls the behavior of the debug agent.
type Config struct {
	// Enabled controls whether the agent is active at all.
	Enabled bool

	// Timeout is the maximum time the agent will spend analyzing an error.
	Timeout time.Duration
}

// AgentResult holds the agent's analysis of an error.
type AgentResult struct {
	// RootCause is the agent's assessment of what caused the error.
	RootCause string

	// Confidence is 0.0–1.0 indicating how sure the agent is.
	Confidence float64

	// Explanation is a human-readable explanation of the error chain.
	Explanation string

	// FixSteps are ordered steps to resolve the error.
	// The consumer decides whether to execute them.
	FixSteps []FixStep

	// Prevention describes how to prevent this error in the future.
	Prevention string

	// RelatedErrors lists error codes that commonly co-occur.
	RelatedErrors []string
}

// FixStep describes a single action to resolve an error.
type FixStep struct {
	// Description is what this step does in plain language.
	Description string

	// Command is the shell command to execute, if applicable.
	Command string

	// Rationale explains WHY this step is needed.
	Rationale string
}

// DebugAgent analyzes errors using diagnostic context.
type DebugAgent interface {
	// Analyze examines an error with diagnostic context and produces
	// a root cause analysis with fix suggestions.
	Analyze(ctx context.Context, err error, diagnosis []*diagnose.DiagnosticResult) (*AgentResult, error)
}

// New creates a new debug agent with the given configuration.
// The agent is disabled by default — set cfg.Enabled = true to activate.
func New(cfg Config) DebugAgent {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
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
	// In a real implementation, this would be sent to an AI provider.
	// For now, deterministic analysis from diagnostic results.
	_ = a.buildPrompt(err, diagnosis)

	return a.deterministicAnalyze(err, diagnosis)
}

func (a *agent) deterministicAnalyze(err error, diagnosis []*diagnose.DiagnosticResult) (*AgentResult, error) {
	result := &AgentResult{
		Confidence: 0.5,
	}

	var parts []string
	for _, d := range diagnosis {
		if d.Status == diagnose.StatusFailed {
			result.Confidence = max(result.Confidence, d.Confidence)
			parts = append(parts, d.Summary)
			if d.SuggestedFix != "" {
				result.FixSteps = append(result.FixSteps, FixStep{
					Description: d.Summary,
					Command:     extractCommand(d.SuggestedFix),
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
