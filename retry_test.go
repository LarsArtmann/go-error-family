package errorfamily

import (
	"testing"
	"time"
)

func TestFamilyRetryPolicy(t *testing.T) {
	tp := Transient.RetryPolicy()
	if tp.MaxAttempts != defaultRetryMaxAttempts {
		t.Errorf("Transient MaxAttempts = %d, want %d", tp.MaxAttempts, defaultRetryMaxAttempts)
	}
	if tp.MinDelay != 100*time.Millisecond {
		t.Errorf("Transient MinDelay = %v, want 100ms", tp.MinDelay)
	}
	if tp.MaxDelay != 5*time.Second {
		t.Errorf("Transient MaxDelay = %v, want 5s", tp.MaxDelay)
	}

	for _, f := range []Family{Rejection, Conflict, Corruption, Infrastructure} {
		rp := f.RetryPolicy()
		if rp.MaxAttempts != 1 {
			t.Errorf("Family(%v) MaxAttempts = %d, want 1 (no retry)", f, rp.MaxAttempts)
		}
	}
}
