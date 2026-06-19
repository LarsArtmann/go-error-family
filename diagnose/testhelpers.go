package diagnose

import "testing"

// AssertDetail fails the test if result.Details[key] != want.
// Intended for use by diagnostic rule tests, including external submodules.
func AssertDetail(t *testing.T, result *DiagnosticResult, key, want string) {
	t.Helper()
	if result.Details[key] != want {
		t.Errorf("Expected %s=%s, got %v", key, want, result.Details)
	}
}

// AssertStatus fails the test if result.Status != want.
// Intended for use by diagnostic rule tests, including external submodules.
func AssertStatus(t *testing.T, result *DiagnosticResult, want Status) {
	t.Helper()
	if result.Status != want {
		t.Errorf("Expected %v, got %v", want, result.Status)
	}
}

// AssertApplicable fails the test if rule.Applicable(err) != want.
// Intended for use by diagnostic rule tests, including external submodules.
func AssertApplicable(t *testing.T, rule DiagnosticRule, err error, want bool) {
	t.Helper()
	if got := rule.Applicable(err); got != want {
		t.Errorf("Applicable() = %v, want %v", got, want)
	}
}
