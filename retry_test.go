package errorfamily

import (
	"testing"
	"time"
)

func TestFamilyRetryPolicy(t *testing.T) {
	policy := Transient.RetryPolicy()
	if policy.MaxAttempts != defaultRetryMaxAttempts {
		t.Errorf("Transient MaxAttempts = %d, want %d", policy.MaxAttempts, defaultRetryMaxAttempts)
	}

	if policy.MinDelay != 100*time.Millisecond {
		t.Errorf("Transient MinDelay = %v, want 100ms", policy.MinDelay)
	}

	if policy.MaxDelay != 5*time.Second {
		t.Errorf("Transient MaxDelay = %v, want 5s", policy.MaxDelay)
	}

	for _, f := range []Family{Rejection, Conflict, Corruption, Infrastructure} {
		rp := f.RetryPolicy()
		if rp.MaxAttempts != 1 {
			t.Errorf("Family(%v) MaxAttempts = %d, want 1 (no retry)", f, rp.MaxAttempts)
		}
	}
}
