// Package agent provides an error analysis agent that uses diagnostic context
// to produce root cause analysis and fix suggestions.
//
// Stability: experimental (v0.x). The API may change between minor versions.
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

const (
	defaultAgentTimeout = 60 * time.Second
	defaultConfidence   = 0.5
)

var errAgentDisabled = errors.New("agent is disabled: set agent.Config{Enabled: true}")

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
	Analyze(
		ctx context.Context,
		err error,
		diagnosis []*diagnose.DiagnosticResult,
	) (*AgentResult, error)
}

// New creates a new debug agent with the given configuration.
// The agent is disabled by default — set cfg.Enabled = true to activate.
func New(cfg Config) DebugAgent {
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultAgentTimeout
	}

	return &agent{cfg: cfg}
}

type agent struct {
	cfg Config
}

func (a *agent) Analyze(
	ctx context.Context,
	err error,
	diagnosis []*diagnose.DiagnosticResult,
) (*AgentResult, error) {
	if !a.cfg.Enabled {
		return nil, errAgentDisabled
	}

	ctx, cancel := context.WithTimeout(ctx, a.cfg.Timeout)
	defer cancel()

	return a.deterministicAnalyze(ctx, err, diagnosis)
}

func (a *agent) deterministicAnalyze(
	ctx context.Context,
	err error,
	diagnosis []*diagnose.DiagnosticResult,
) (*AgentResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	result := &AgentResult{
		Confidence: defaultConfidence,
	}

	var parts []string

	for _, diag := range diagnosis {
		if diag.Status == diagnose.StatusFailed {
			result.Confidence = max(result.Confidence, diag.Confidence)
			parts = append(parts, diag.Summary)
			// Structured Fix comes directly from the diagnostic rule — no prose
			// parsing needed. Emit a FixStep whenever the rule has any guidance.
			if diag.Fix.Command != "" || diag.Fix.Summary != "" {
				result.FixSteps = append(result.FixSteps, FixStep{
					Description: diag.Summary,
					Command:     diag.Fix.Command,
					Rationale: fmt.Sprintf(
						"Diagnostic rule '%s' identified this issue",
						diag.RuleName,
					),
				})
			}
		}
	}

	if len(parts) > 0 {
		result.RootCause = parts[0]
		result.Explanation = strings.Join(parts, "; ")
	} else {
		result.RootCause = "no specific root cause identified"
		result.Explanation = fmt.Sprintf(
			"Error classified as %s. No diagnostic failures found.",
			errorfamily.Classify(err),
		)
	}

	return result, nil
}
