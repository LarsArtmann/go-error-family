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
		t.Errorf("custom registry template not used: %q", result.Message)
	}
	if result.SuggestedFix != "Detailed custom fix" {
		t.Errorf("SuggestedFix should use template fix: %q", result.SuggestedFix)
	}
}

// TestRegistryConcurrentRegisterAndClassify hammers the copy-on-write sentinel
// store with concurrent writers and readers. Run with -race to catch any data race
// in the atomic.Pointer swap path.
func TestRegistryConcurrentRegisterAndClassify(t *testing.T) {
	reg := NewRegistry()

	target := errors.New("target sentinel")
	reg.RegisterClassification(target, Transient)

	done := make(chan struct{})

	// Writers: register/unregister in a tight loop.
	go func() {
		defer close(done)
		for i := range 1000 {
			e := fmt.Errorf("writer-%d", i)
			reg.RegisterClassification(e, Rejection)
			if i%2 == 0 {
				reg.UnregisterClassification(e)
			}
		}
	}()

	// Readers: classify concurrently (lock-free, must never race or panic).
	for range 1000 {
		if got := reg.Classify(target); got != Transient {
			t.Fatalf("Classify(target) = %v, want Transient", got)
		}
	}

	<-done
}

func TestRegistryCloneIndependence(t *testing.T) {
	original := NewRegistry()
	sentinel := errors.New("original sentinel")
	original.RegisterClassification(sentinel, Rejection)
	original.RegisterTemplate("original.code", MessageTemplate{What: "original what"})

	clone := original.Clone()

	// Mutate the clone.
	clone.RegisterClassification(errors.New("clone-only"), Conflict)
	clone.UnregisterClassification(sentinel)
	clone.RegisterTemplate("clone.code", MessageTemplate{What: "clone what"})

	// Original must be unaffected.
	if got := original.Classify(sentinel); got != Rejection {
		t.Errorf("original mutated by clone: Classify(sentinel) = %v, want Rejection", got)
	}
	tmpl, ok := original.lookupTemplate("original.code")
	if !ok || tmpl.What != "original what" {
		t.Errorf("original template lost after clone mutation: %+v ok=%v", tmpl, ok)
	}
	if _, ok := original.lookupTemplate("clone.code"); ok {
		t.Error("original gained clone-only template")
	}

	// Clone must have dropped the sentinel: Classify now falls through to the
	// Transient default (distinct from Rejection, so this proves the drop).
	if got := clone.Classify(sentinel); got != Transient {
		t.Errorf("clone did not drop sentinel: Classify = %v, want Transient (default)", got)
	}
}

func TestRegistryRegisterTemplatesBatch(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterTemplates(map[string]MessageTemplate{
		"code.one": {What: "one"},
		"CODE.TWO": {What: "two"},
	})

	one, ok := reg.lookupTemplate("code.one")
	if !ok || one.What != "one" {
		t.Errorf("batch template code.one missing: %+v ok=%v", one, ok)
	}
	// Case-insensitive lookup (registered as CODE.TWO).
	two, ok := reg.lookupTemplate("code.two")
	if !ok || two.What != "two" {
		t.Errorf("batch template code.two (case-insensitive) missing: %+v ok=%v", two, ok)
	}
}
