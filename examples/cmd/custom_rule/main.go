// Example: Writing a custom diagnostic rule.
//
// This shows how to create a rule that matches errors mentioning "rate limit"
// and checks if the retry-after header is present.
package main

import (
	"context"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

// RateLimitRule matches rate-limit errors and suggests waiting.
type RateLimitRule struct{}

func (r *RateLimitRule) Name() string { return "rate_limit" }

func (r *RateLimitRule) Applicable(err error) bool {
	// Match by context key or error code.
	return diagnose.HasContextKey(err, "retry_after") ||
		diagnose.ErrorCodeContains(err, "rate.limit")
}

func (r *RateLimitRule) Run(ctx context.Context, err error) (*diagnose.DiagnosticResult, error) {
	retryAfter := diagnose.ResolveContextKey(
		err,
		[]string{"retry_after", "retry-after", "Retry-After"},
		"unknown",
	)

	result := &diagnose.DiagnosticResult{
		Status:     diagnose.StatusDegraded,
		Confidence: diagnose.ConfidenceHigh,
		Details:    map[string]string{"retry_after": retryAfter},
	}

	if retryAfter == "unknown" {
		result.Summary = "Rate limited but no Retry-After header found"
		result.Fix = diagnose.Fix{
			Summary: "Wait 1 second and retry, or implement exponential backoff",
		}
	} else {
		result.Summary = fmt.Sprintf("Rate limited — wait %s before retrying", retryAfter)
		result.Fix = diagnose.Fix{
			Summary: "Wait for the duration specified in the Retry-After header",
		}
		result.Status = diagnose.StatusHealthy
		result.Confidence = diagnose.ConfidenceNotCause
	}

	return result, nil
}

func main() {
	err := errorfamily.NewTransient("rate.limit.exceeded", "too many requests").
		WithContext("retry_after", "12")

	// Create a runner with the custom rule alongside built-in rules.
	runner := diagnose.NewRunner(
		&RateLimitRule{},
		&diagnose.NetworkRule{},
	)

	results := runner.Run(context.Background(), err)
	for _, r := range results {
		fmt.Printf("[%s] %s: %s\n", r.RuleName, r.Status, r.Summary)

		if r.Fix.Summary != "" {
			fmt.Printf("  Fix: %s\n", r.Fix.Summary)
		}
	}
}
