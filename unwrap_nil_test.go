package errorfamily

import "testing"

// DD22: a nil *Error must not panic on Unwrap - defensive guard for
// callers that forward concrete pointers into error interfaces and
// unwrap them without a nil check (the gotcha-#213 class).
func TestUnwrap_NilReceiverSafe(t *testing.T) {
	var e *Error

	if got := e.Unwrap(); got != nil {
		t.Fatalf("nil receiver Unwrap = %v, want nil", got)
	}
}
