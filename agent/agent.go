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
		cfg.Timeout = 60 * time.Second
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
		return nil, errors.New("agent is disabled: set agent.Config{Enabled: true}")
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
					Rationale: fmt.Sprintf(
						"Diagnostic rule '%s' identified this issue",
						d.RuleName,
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

	// Diagnostic rules produce suggestions like "Description:\n  command args".
	// Look for the first indented line that looks like a shell command.
	for line := range strings.SplitSeq(suggest, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Indented lines (2+ spaces) in the middle of a suggestion are commands.
		if strings.HasPrefix(line, "  ") && len(trimmed) > 0 {
			// Skip lines that are just prose (contain ":" at end or are too wordy).
			if strings.HasSuffix(trimmed, ":") ||
				strings.Contains(trimmed, " ") && !looksLikeCommand(trimmed) {
				continue
			}
			return trimmed
		}
	}
	return ""
}

// looksLikeCommand reports whether a string looks like a shell command rather than prose.
func looksLikeCommand(s string) bool {
	// Commands typically start with a known command name or contain shell operators.
	shellPrefixes := []string{
		"git ", "mkdir ", "chmod ", "nc ", "dig ", "nslookup ",
		"brew ", "systemctl ", "service ", "pg_", "cd ",
		"docker ", "curl ", "ssh ", "cp ", "mv ", "rm ", "cat ",
	}
	lower := strings.ToLower(s)
	for _, prefix := range shellPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	// Lines containing && or | are almost certainly commands.
	if strings.Contains(s, " && ") || strings.Contains(s, " | ") {
		return true
	}
	return false
}
