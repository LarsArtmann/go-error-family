package diagnose_test

import (
	"context"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

// webhookTimeoutRule is a minimal data-driven rule: a RuleSpec decides which
// errors it applies to, and Run produces one fixed diagnostic. Build real
// rules the same way, injecting a diagnose.CommandRunner for any system
// commands so tests stay deterministic.
type webhookTimeoutRule struct {
	Spec diagnose.RuleSpec
}

func (r *webhookTimeoutRule) Name() string { return "webhook-timeout" }

func (r *webhookTimeoutRule) Applicable(err error) bool { return r.Spec.Matches(err) }

func (r *webhookTimeoutRule) Run(
	ctx context.Context,
	err error,
) (*diagnose.DiagnosticResult, error) {
	result := &diagnose.DiagnosticResult{
		Status:     diagnose.StatusDegraded,
		Summary:    "Webhook endpoint slower than the client timeout",
		Confidence: diagnose.ConfidenceLikely,
		Details: map[string]string{
			"endpoint": diagnose.ContextValue(err, string(diagnose.KeyURL)),
		},
		Context: diagnose.ErrorContext(err),
	}
	diagnose.SetFix(result, "Raise the client timeout", "app config: webhooks.timeout = 10s")
	return result, nil
}

func ExampleRuleSpec() {
	rule := &webhookTimeoutRule{
		Spec: diagnose.RuleSpec{
			ContextKeys:   []diagnose.ContextKey{diagnose.KeyURL},
			ContextSubstr: []string{"webhook"},
		},
	}

	err := errorfamily.NewTransient("webhook.timeout", "delivery timed out").
		WithContext("url", "https://hooks.example.com/orders")

	fmt.Println(rule.Applicable(err))

	runner := diagnose.NewRunner()
	runner.Register(rule)
	results := runner.Run(context.Background(), err)

	for _, result := range results {
		fmt.Println(result.RuleName, result.Status, result.Details["endpoint"])
		fmt.Println(result.Fix.Command)
	}
	fmt.Println(
		diagnose.ResolveContextKey(
			err,
			[]string{string(diagnose.KeyURL), string(diagnose.KeyEndpoint)},
			"unknown",
		),
	)

	// Output:
	// true
	// webhook-timeout degraded https://hooks.example.com/orders
	// app config: webhooks.timeout = 10s
	// https://hooks.example.com/orders
}
