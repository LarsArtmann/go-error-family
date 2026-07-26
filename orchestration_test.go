package errorfamily

import (
	"errors"
	"testing"
)

// TestOrchestrationIntegration verifies the Orchestration family end-to-end:
// the constructor produces an error that Classify recognizes as Orchestration,
// and every derived property (exit code, HTTP status, retryability, severity,
// audience, tone) agrees with the family metadata in family.go.
//
// This guards against the class of bug where a new family is added to the const
// block but the familyData table, IsValid boundary, or severity order drifts.
func TestOrchestrationIntegration(t *testing.T) {
	t.Parallel()

	err := NewOrchestration("render.failed", "template engine returned no output")

	t.Run("Classify recognizes Orchestration", func(t *testing.T) {
		t.Parallel()

		if got := Classify(err); got != Orchestration {
			t.Fatalf("Classify(NewOrchestration(...)) = %v, want Orchestration", got)
		}
	})

	t.Run("exit code is EX_SOFTWARE (70)", func(t *testing.T) {
		t.Parallel()

		if got := ExitCode(err); got != 70 {
			t.Errorf("ExitCode() = %d, want 70 (EX_SOFTWARE)", got)
		}

		if got := Orchestration.ExitCode(); got != 70 {
			t.Errorf("Family.ExitCode() = %d, want 70", got)
		}
	})

	t.Run("HTTP status is 500", func(t *testing.T) {
		t.Parallel()

		if got := HTTPStatus(err); got != 500 {
			t.Errorf("HTTPStatus() = %d, want 500", got)
		}

		if got := Orchestration.HTTPStatus(); got != 500 {
			t.Errorf("Family.HTTPStatus() = %d, want 500", got)
		}
	})

	t.Run("not retryable", func(t *testing.T) {
		t.Parallel()

		if IsRetryable(err) {
			t.Error("IsRetryable(NewOrchestration(...)) = true, want false")
		}

		if Orchestration.IsRetryable() {
			t.Error("Orchestration.IsRetryable() = true, want false")
		}

		if got := Orchestration.RetryPolicy().MaxAttempts; got != 1 {
			t.Errorf("RetryPolicy().MaxAttempts = %d, want 1 (no retry)", got)
		}
	})

	t.Run("severity sits between Infrastructure and Corruption", func(t *testing.T) {
		t.Parallel()

		severity := Orchestration.Severity()
		if severity != 5 {
			t.Errorf("Severity() = %d, want 5", severity)
		}

		if Infrastructure.Severity() >= severity {
			t.Errorf(
				"Infrastructure severity %d not < Orchestration %d",
				Infrastructure.Severity(),
				severity,
			)
		}

		if severity >= Corruption.Severity() {
			t.Errorf(
				"Orchestration severity %d not < Corruption %d",
				severity,
				Corruption.Severity(),
			)
		}
	})

	t.Run("IsValid boundary includes Orchestration", func(t *testing.T) {
		t.Parallel()

		if !Orchestration.IsValid() {
			t.Error("Orchestration.IsValid() = false, want true")
		}

		if !validFamilyCountIncludesOrchestration() {
			t.Error("Orchestration not counted in the valid family range")
		}
	})

	t.Run("audience and tone", func(t *testing.T) {
		t.Parallel()

		if got := Orchestration.Audience(); got != AudienceOps {
			t.Errorf("Audience() = %v, want AudienceOps", got)
		}

		if got := Orchestration.Tone(); got != ToneApologetic {
			t.Errorf("Tone() = %v, want ToneApologetic", got)
		}
	})
}

// TestOrchestrationMultiErrorOrdering verifies that in a joined error,
// Orchestration (severity 5) dominates Infrastructure (severity 4) but
// Corruption (severity 6) dominates Orchestration — the total order that
// keeps the "worst wins" classification deterministic.
func TestOrchestrationMultiErrorOrdering(t *testing.T) {
	t.Parallel()

	t.Run("Orchestration beats Infrastructure", func(t *testing.T) {
		t.Parallel()

		joined := errors.Join(
			NewInfrastructure("startup", "nil dependency"),
			NewOrchestration("render.failed", "no output"),
		)
		if got := Classify(joined); got != Orchestration {
			t.Fatalf("Classify(Join(infra, orch)) = %v, want Orchestration", got)
		}
	})

	t.Run("Corruption beats Orchestration", func(t *testing.T) {
		t.Parallel()

		joined := errors.Join(
			NewOrchestration("render.failed", "no output"),
			NewCorruption("schema.break", "unparseable payload"),
		)
		if got := Classify(joined); got != Corruption {
			t.Fatalf("Classify(Join(orch, corrupt)) = %v, want Corruption", got)
		}
	})

	t.Run("order independent of argument order", func(t *testing.T) {
		t.Parallel()

		first := errors.Join(NewOrchestration("a", "a"), NewCorruption("b", "b"))

		second := errors.Join(NewCorruption("b", "b"), NewOrchestration("a", "a"))
		if Classify(first) != Classify(second) {
			t.Error("multi-error classification must be independent of argument order")
		}
	})
}

// TestOrchestrationWrapVariants verifies the Wrap and Wrapf constructors
// preserve the Orchestration family through wrapping chains.
func TestOrchestrationWrapVariants(t *testing.T) {
	t.Parallel()

	root := errors.New("disk full")

	wrapped := WrapOrchestration(root, "build.failed", "could not build artifact")
	if got := Classify(wrapped); got != Orchestration {
		t.Fatalf("Classify(WrapOrchestration(...)) = %v, want Orchestration", got)
	}

	if !errors.Is(wrapped, root) {
		t.Error("WrapOrchestration must preserve errors.Is to the wrapped error")
	}

	wrappedf := WrapOrchestrationf(root, "build.failed", "artifact %s", "bin/app")
	if got := Classify(wrappedf); got != Orchestration {
		t.Fatalf("Classify(WrapOrchestrationf(...)) = %v, want Orchestration", got)
	}

	if !errors.Is(wrappedf, root) {
		t.Error("WrapOrchestrationf must preserve errors.Is to the wrapped error")
	}
}

// validFamilyCountIncludesOrchestration is a sanity helper: the contiguous
// valid range Rejection..Orchestration must include Orchestration. If a future
// edit moves the const or breaks the iota sequence, this catches it.
func validFamilyCountIncludesOrchestration() bool {
	count := 0

	for f := Rejection; f <= Orchestration; f++ {
		if f.IsValid() {
			count++
		}
	}

	return count >= 6 // Rejection, Conflict, Transient, Corruption, Infrastructure, Orchestration
}
