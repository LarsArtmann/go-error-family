package errorfamily

import (
	"errors"
	"fmt"
	"testing"
)

func FuzzParseFamily(f *testing.F) {
	f.Add("transient")
	f.Add("REJECTION")
	f.Add("conflict")
	f.Add("corruption")
	f.Add("infrastructure")
	f.Add("")
	f.Add("garbage_12345")
	f.Add("transient\nwithnewline")
	f.Add("trans\x00ent")

	f.Fuzz(func(t *testing.T, input string) {
		got := ParseFamily(input)
		if !got.IsValid() {
			t.Errorf("ParseFamily(%q) = %v, not a valid Family", input, got)
		}
	})
}

func FuzzParseFamilyRoundTrip(f *testing.F) {
	f.Add("rejection")
	f.Add("conflict")
	f.Add("transient")
	f.Add("corruption")
	f.Add("infrastructure")

	f.Fuzz(func(t *testing.T, input string) {
		first := ParseFamily(input)

		second := ParseFamily(first.String())
		if first != second {
			t.Errorf("ParseFamily(%q) = %v, but ParseFamily(%q) = %v",
				input, first, first.String(), second)
		}
	})
}

func FuzzClassify(f *testing.F) {
	f.Add("some error")
	f.Add("")
	f.Add("db.timeout")
	f.Add("connection.refused")
	f.Add("rate.limit.exceeded")

	f.Fuzz(func(t *testing.T, msg string) {
		err := NewTransient("fuzz.code", msg)

		got := Classify(err)
		if got != Transient {
			t.Errorf("Classify(NewTransient(...)) = %v, want Transient", got)
		}
	})
}

func FuzzClassifyPlainError(f *testing.F) {
	f.Add("generic error")
	f.Add("")
	f.Add("connection refused")
	f.Add("timeout")

	f.Fuzz(func(t *testing.T, msg string) {
		err := errors.New(msg)

		got := Classify(err)
		if got != Transient {
			t.Errorf("Classify(errors.New(%q)) = %v, want Transient (default)", msg, got)
		}
	})
}

func FuzzErrorFormatting(f *testing.F) {
	f.Add("db.timeout", "query took too long")
	f.Add("", "")
	f.Add("code", "msg with\nnewline")
	f.Add("x", "msg\twith\ttabs")

	f.Fuzz(func(t *testing.T, code, message string) {
		err := New(Transient, code, message)

		s := err.Error()
		if s == "" {
			t.Error("Error() returned empty string")
		}

		verbose := fmt.Sprintf("%+v", err)
		if verbose == "" {
			t.Errorf("%%+v formatting returned empty string")
		}

		plain := fmt.Sprintf("%s", err)
		if plain != message {
			t.Errorf("%%s = %q, want %q", plain, message)
		}
	})
}

// FuzzApplyContext verifies the {key} template substitution never panics and
// always terminates for arbitrary input (including nested/malformed braces).
// Replacement is a single non-overlapping pass; nested placeholders like
// "{x{x}}" are an inherent edge case, not a bug, so we only assert crash-safety.
func FuzzApplyContext(f *testing.F) {
	f.Add("hello {name}", "name", "world")
	f.Add("no placeholders", "k", "v")
	f.Add("{a} and {b}", "a", "1")
	f.Add("nested {brace", "x", "y")
	f.Add("value with {x} inside", "x", "replaced")
	f.Add("{x{x}}", "x", "")

	f.Fuzz(func(t *testing.T, template, key, value string) {
		_ = applyContext(template, map[string]string{key: value})
	})
}

// FuzzWrapOnce verifies that WrapOnce never panics and is idempotent:
// wrapping an already-wrapped error returns the same pointer.
func FuzzWrapOnce(f *testing.F) {
	f.Add("db.timeout", "query failed")
	f.Add("", "")
	f.Add("code", "msg\nwith\nnewlines")

	f.Fuzz(func(t *testing.T, code, msg string) {
		base := errors.New(msg)

		wrapped := WrapOnce(base, Transient, code, msg)
		if wrapped == nil {
			t.Fatal("WrapOnce returned nil for non-nil input")
		}

		doubleWrapped := WrapOnce(wrapped, Rejection, "other.code", "other msg")
		if doubleWrapped != wrapped {
			t.Error("WrapOnce is not idempotent: double-wrap produced a different pointer")
		}
	})
}

// FuzzContextValueToString verifies that WithContextAny never panics on
// arbitrary string input and round-trips the value correctly.
func FuzzContextValueToString(f *testing.F) {
	f.Add("count", "42")
	f.Add("host", "localhost:5432")
	f.Add("", "")
	f.Add("path", "/var/log/app\n.log")

	f.Fuzz(func(t *testing.T, key, val string) {
		err := NewTransient("test", "msg").WithContextAny(key, val)
		if err.ContextValue(key) != val {
			t.Errorf("ContextValue(%q) = %q, want %q", key, err.ContextValue(key), val)
		}
	})
}

// FuzzWithExitCode verifies that WithExitCode never panics and that the
// package-level ExitCode function resolves the override correctly.
func FuzzWithExitCode(f *testing.F) {
	f.Add(1)
	f.Add(42)
	f.Add(0)
	f.Add(-1)
	f.Add(255)

	f.Fuzz(func(t *testing.T, code int) {
		err := NewTransient("test", "msg").WithExitCode(code)
		if got := err.ExitCode(); got != code {
			t.Errorf("ExitCode() = %d, want %d", got, code)
		}

		if code != 0 {
			if got := ExitCode(err); got != code {
				t.Errorf("package ExitCode = %d, want %d (override)", got, code)
			}
		}
	})
}
