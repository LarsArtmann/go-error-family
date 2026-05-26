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
