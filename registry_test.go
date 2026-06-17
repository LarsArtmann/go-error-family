package errorfamily

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRegistryClassifyBasic(t *testing.T) {
	reg := NewRegistry()

	plain := errors.New("some error")
	if reg.Classify(plain) != Transient {
		t.Error("Registry.Classify should default unknown errors to Transient")
	}

	classified := NewRejection("test.code", "msg")
	if reg.Classify(classified) != Rejection {
		t.Error("Registry.Classify should use Classified interface")
	}

	if reg.Classify(nil) != Rejection {
		t.Error("Registry.Classify(nil) should return Rejection")
	}
}

func TestRegistrySentinelIsolation(t *testing.T) {
	sentinel := errors.New("isolated.sentinel")

	// Register on a custom registry only.
	reg := NewRegistry()
	reg.RegisterClassification(sentinel, Corruption)

	// The custom registry sees it.
	if reg.Classify(sentinel) != Corruption {
		t.Error("custom registry should classify sentinel as Corruption")
	}

	// DefaultRegistry does NOT see it — no leakage.
	if DefaultRegistry.Classify(sentinel) != Transient {
		t.Error("DefaultRegistry should not see custom registry's sentinel")
	}
}

func TestRegistrySentinelNoCleanupNeeded(t *testing.T) {
	// This test proves the key benefit of injectable registries:
	// no t.Cleanup(Unregister...) needed. The registry simply goes out of scope.
	sentinel := errors.New("disposable.sentinel")

	reg := NewRegistry()
	reg.RegisterClassification(sentinel, Infrastructure)

	wrapped := fmt.Errorf("wrapped: %w", sentinel)
	if reg.Classify(wrapped) != Infrastructure {
		t.Error("Registry should classify wrapped sentinel via errors.Is chain walk")
	}

	// When this function returns, reg is garbage collected.
	// No global state was mutated. No cleanup needed.
}

func TestRegistryTemplateIsolation(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterTemplate("custom.code", MessageTemplate{
		What: "Custom message from isolated registry",
		Fix:  "Custom fix from isolated registry",
	})

	var buf bytes.Buffer
	err := NewRejection("custom.code", "msg")
	code := HandleErrorWithConfig(err, HandleConfig{
		Output:   &buf,
		Registry: reg,
	})
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	output := buf.String()
	if !strings.Contains(output, "Custom message from isolated registry") {
		t.Errorf("should use isolated registry template: %q", output)
	}

	// DefaultRegistry should NOT see this template.
	var defaultBuf bytes.Buffer
	HandleErrorWithConfig(err, HandleConfig{Output: &defaultBuf})
	if strings.Contains(defaultBuf.String(), "Custom message from isolated registry") {
		t.Error("DefaultRegistry should not see isolated registry's template")
	}
}

func TestRegistryTwoRegistriesIndependent(t *testing.T) {
	sentinel := errors.New("shared.sentinel")

	regA := NewRegistry()
	regB := NewRegistry()

	regA.RegisterClassification(sentinel, Corruption)
	regB.RegisterClassification(sentinel, Conflict)

	if regA.Classify(sentinel) != Corruption {
		t.Error("regA should classify as Corruption")
	}
	if regB.Classify(sentinel) != Conflict {
		t.Error("regB should classify as Conflict")
	}
}

func TestRegistryBatchRegistration(t *testing.T) {
	reg := NewRegistry()
	s1 := errors.New("batch.1")
	s2 := errors.New("batch.2")

	reg.RegisterClassifications(map[error]Family{
		s1: Transient,
		s2: Rejection,
	})

	if reg.Classify(s1) != Transient {
		t.Error("batch-registered s1 should be Transient")
	}
	if reg.Classify(s2) != Rejection {
		t.Error("batch-registered s2 should be Rejection")
	}

	reg.UnregisterClassification(s1)
	if reg.Classify(s1) != Transient {
		t.Error("after unregister, s1 should fall back to Transient default")
	}
}

func TestRegistryUnregisterTemplate(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterTemplate("temp.code", MessageTemplate{What: "temporary"})
	reg.UnregisterTemplate("temp.code")

	var buf bytes.Buffer
	err := NewRejection("temp.code", "msg")
	HandleErrorWithConfig(err, HandleConfig{Output: &buf, Registry: reg})

	if strings.Contains(buf.String(), "temporary") {
		t.Error("unregistered template should not appear in output")
	}
}

func TestRegistryNilInConfigFallsBackToDefault(t *testing.T) {
	// When HandleConfig.Registry is nil, DefaultRegistry is used.
	// This proves backward compatibility — existing code that doesn't
	// set Registry still works identically.
	sentinel := errors.New("fallback.sentinel")
	RegisterClassification(sentinel, Corruption)
	t.Cleanup(func() { UnregisterClassification(sentinel) })

	var buf bytes.Buffer
	err := NewRejection("test", "msg")
	code := HandleErrorWithConfig(err, HandleConfig{Output: &buf, Registry: nil})
	if code != 1 {
		t.Errorf("nil registry should fall back to DefaultRegistry: exit code = %d", code)
	}
}

func TestRegistryHandleErrorDetailedWithCustomRegistry(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterTemplate("detail.code", MessageTemplate{
		What: "Detailed custom message",
		Fix:  "Detailed custom fix",
	})

	err := NewRejection("detail.code", "msg")
	result := HandleErrorDetailedWithConfig(err, HandleConfig{Registry: reg})

	if !strings.Contains(result.Message, "Detailed custom message") {
		t.Errorf("HandleErrorDetailedWithConfig should use custom registry template: %q", result.Message)
	}
	if result.SuggestedFix != "Detailed custom fix" {
		t.Errorf("SuggestedFix should use template fix: %q", result.SuggestedFix)
	}
}
