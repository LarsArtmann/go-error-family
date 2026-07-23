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
	sentinel1 := errors.New("batch.1")
	sentinel2 := errors.New("batch.2")

	reg.RegisterClassifications(map[error]Family{
		sentinel1: Transient,
		sentinel2: Rejection,
	})

	if reg.Classify(sentinel1) != Transient {
		t.Error("batch-registered sentinel1 should be Transient")
	}

	if reg.Classify(sentinel2) != Rejection {
		t.Error("batch-registered sentinel2 should be Rejection")
	}

	reg.UnregisterClassification(sentinel1)

	if reg.Classify(sentinel1) != Transient {
		t.Error("after unregister, sentinel1 should fall back to Transient default")
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

// fakeSQLiteError mimics a dynamic third-party error type (like *sqlite.Error)
// where each instance is distinct — impossible to register as a sentinel.
type fakeSQLiteError struct{ code int }

func (e *fakeSQLiteError) Error() string { return fmt.Sprintf("sqlite: code %d", e.code) }

func TestRegistryRegisterClassifierDynamic(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterClassifier(func(err error) (Family, bool) {
		if sq, ok := errors.AsType[*fakeSQLiteError](err); ok {
			switch sq.code {
			case 5, 6:
				return Transient, true // BUSY, LOCKED
			case 19:
				return Conflict, true // CONSTRAINT
			}
		}

		return Transient, false
	})

	// Dynamic errors are classified by the predicate, not sentinel identity.
	if got := reg.Classify(&fakeSQLiteError{code: 5}); got != Transient {
		t.Errorf("locked = %v, want Transient", got)
	}

	if got := reg.Classify(&fakeSQLiteError{code: 19}); got != Conflict {
		t.Errorf("constraint = %v, want Conflict", got)
	}
	// Unmatched code → predicate returns false → default Transient.
	if got := reg.Classify(&fakeSQLiteError{code: 999}); got != Transient {
		t.Errorf("unknown code = %v, want Transient (default)", got)
	}
}

func TestRegisterClassifiersBatch(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterClassifiers(
		func(err error) (Family, bool) {
			if err.Error() == "conflict-ish" {
				return Conflict, true
			}

			return Transient, false
		},
		func(err error) (Family, bool) {
			if err.Error() == "reject-me" {
				return Rejection, true
			}

			return Transient, false
		},
	)

	if got := reg.Classify(errors.New("conflict-ish")); got != Conflict {
		t.Errorf("first classifier = %v, want Conflict", got)
	}

	if got := reg.Classify(errors.New("reject-me")); got != Rejection {
		t.Errorf("second classifier = %v, want Rejection", got)
	}
}

func TestClassifierFirstMatchWins(t *testing.T) {
	reg := NewRegistry()
	// Two classifiers both claim to match — the earlier-registered one wins.
	reg.RegisterClassifier(func(err error) (Family, bool) { return Corruption, true })
	reg.RegisterClassifier(func(err error) (Family, bool) { return Rejection, true })

	if got := reg.Classify(errors.New("anything")); got != Corruption {
		t.Errorf("first-match = %v, want Corruption", got)
	}
}

func TestClassifierDoesNotShadowExplicitClassification(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterClassifier(func(err error) (Family, bool) { return Corruption, true })

	// An error that already declares its family must bypass classifiers.
	err := NewTransient("db.timeout", "timed out")
	if got := reg.Classify(err); got != Transient {
		t.Errorf("explicit Classified should bypass classifier: got %v, want Transient", got)
	}
}

func TestClassifierDoesNotShadowSentinel(t *testing.T) {
	reg := NewRegistry()
	sentinel := errors.New("known sentinel")
	reg.RegisterClassification(sentinel, Conflict)
	reg.RegisterClassifier(func(err error) (Family, bool) { return Corruption, true })

	if got := reg.Classify(sentinel); got != Conflict {
		t.Errorf("sentinel should beat classifier: got %v, want Conflict", got)
	}
}

func TestClassifierCloneIndependence(t *testing.T) {
	original := NewRegistry()
	original.RegisterClassifier(func(err error) (Family, bool) {
		if err.Error() == "match" {
			return Corruption, true
		}

		return Transient, false
	})

	clone := original.Clone()
	// Add a classifier only to the clone.
	clone.RegisterClassifier(func(err error) (Family, bool) { return Infrastructure, true })

	// Clone classifies "match" via the inherited classifier.
	if got := clone.Classify(errors.New("match")); got != Corruption {
		t.Errorf("clone lost inherited classifier: got %v, want Corruption", got)
	}
	// Original does not gain the clone-only classifier.
	if got := original.Classify(errors.New("nope")); got != Transient {
		t.Errorf("original gained clone-only classifier: got %v, want Transient", got)
	}
}

func TestClassifierConcurrentRegisterAndClassify(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterClassifier(func(err error) (Family, bool) {
		if err.Error() == "stable" {
			return Rejection, true
		}

		return Transient, false
	})

	done := make(chan struct{})
	go func() {
		defer close(done)

		for i := range 500 {
			reg.RegisterClassifier(func(err error) (Family, bool) {
				if err.Error() == fmt.Sprintf("w-%d", i) {
					return Conflict, true
				}

				return Transient, false
			})
		}
	}()

	for range 500 {
		if got := reg.Classify(errors.New("stable")); got != Rejection {
			t.Fatalf("concurrent classify unstable: got %v, want Rejection", got)
		}
	}

	<-done
}

// pkgLevelClassifierError is a unique type used only by the package-level
// RegisterClassifier test so it cannot interfere with other tests' expectations.
type pkgLevelClassifierError struct{}

func (pkgLevelClassifierError) Error() string { return "package-level-classifier-marker" }

func TestPackageLevelRegisterClassifier(t *testing.T) {
	RegisterClassifiers(
		// Highly specific — only matches our marker type. Stays registered but
		// cannot affect other tests because nothing else returns this type.
		func(err error) (Family, bool) {
			if _, ok := errors.AsType[pkgLevelClassifierError](err); ok {
				return Corruption, true
			}

			return Transient, false
		},
	)
	// Plain errors unrelated to the marker must still classify as the default.
	if got := Classify(errors.New("unrelated")); got != Transient {
		t.Errorf("unrelated error = %v, want Transient", got)
	}

	if got := Classify(pkgLevelClassifierError{}); got != Corruption {
		t.Errorf("marker error = %v, want Corruption", got)
	}
}

// pkgLevelSingularError is a distinct marker type for the singular
// RegisterClassifier test — kept separate so the batch test above doesn't
// already satisfy coverage for the singular variant.
type pkgLevelSingularError struct{}

func (pkgLevelSingularError) Error() string { return "singular marker" }

func TestPackageLevelRegisterClassifierSingular(t *testing.T) {
	RegisterClassifier(func(err error) (Family, bool) {
		if _, ok := errors.AsType[pkgLevelSingularError](err); ok {
			return Infrastructure, true
		}

		return Transient, false
	})

	if got := Classify(pkgLevelSingularError{}); got != Infrastructure {
		t.Errorf("singular marker = %v, want Infrastructure", got)
	}
}
