package bridge

import (
	"errors"
	"fmt"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/samber/oops"
)

func FuzzInferFamily(f *testing.F) {
	f.Add("database", "timeout", "host")
	f.Add("validation", "", "")
	f.Add("", "retryable", "")
	f.Add("network", "conflict", "")
	f.Add("unknown", "", "")

	f.Fuzz(func(t *testing.T, domain, tag, contextKey string) {
		var err error
		builder := oops.Errorf("test error")
		if domain != "" {
			builder = oops.In(domain).Errorf("test error")
		}
		if tag != "" {
			if domain != "" {
				err = oops.In(domain).Tags(tag).Errorf("test error")
			} else {
				err = oops.Tags(tag).Errorf("test error")
			}
		} else {
			err = builder
		}

		family := InferFamily(err)
		if !family.IsValid() {
			t.Errorf(
				"InferFamily returned invalid family %v for domain=%q tag=%q",
				family,
				domain,
				tag,
			)
		}
	})
}

func FuzzAutoWrap(f *testing.F) {
	f.Add("database", "timeout", "db.timeout")
	f.Add("validation", "", "")
	f.Add("network", "retryable", "net.error")
	f.Add("", "", "")
	f.Add("data", "corruption", "schema.invalid")

	f.Fuzz(func(t *testing.T, domain, tag, code string) {
		var err error
		switch {
		case domain != "" && tag != "":
			err = oops.In(domain).Tags(tag).Code(code).Errorf("test: %s", code)
		case domain != "":
			err = oops.In(domain).Code(code).Errorf("test: %s", code)
		case tag != "":
			err = oops.Tags(tag).Code(code).Errorf("test: %s", code)
		default:
			err = oops.Code(code).Errorf("test: %s", code)
		}

		classified := AutoWrap(err)
		if !classified.Family().IsValid() {
			t.Errorf("AutoWrap produced invalid family %v", classified.Family())
		}
		if classified.Error() == "" {
			t.Error("AutoWrap produced empty Error()")
		}
	})
}

func FuzzWrapRoundTrip(f *testing.F) {
	f.Add("rejection", "file.not_found", "config missing")
	f.Add("transient", "db.timeout", "connection failed")
	f.Add("conflict", "version.mismatch", "stale data")
	f.Add("corruption", "schema.invalid", "bad schema")
	f.Add("infrastructure", "service.down", "unavailable")

	f.Fuzz(func(t *testing.T, familyStr, code, message string) {
		family := errorfamily.ParseFamily(familyStr)

		inner := errors.New(message)
		classified := Wrap(inner, family)

		if classified.Error() == "" {
			t.Error("Wrap produced empty Error()")
		}
		if classified.Family() != family {
			t.Errorf("Family() = %v, want %v", classified.Family(), family)
		}
		if !errors.Is(classified, inner) {
			t.Error("original error lost in Unwrap chain")
		}
		if classified.IsRetryable() != family.IsRetryable() {
			t.Errorf("IsRetryable() = %v, want %v", classified.IsRetryable(), family.IsRetryable())
		}

		classifiedFamily := errorfamily.Classify(classified)
		if classifiedFamily != family {
			t.Errorf("Classify() = %v, want %v", classifiedFamily, family)
		}
	})
}

func FuzzWrapOopsRoundTrip(f *testing.F) {
	f.Add("database", "db.timeout", "connection failed")
	f.Add("validation", "input.invalid", "bad input")
	f.Add("network", "net.timeout", "timeout")

	f.Fuzz(func(t *testing.T, domain, code, message string) {
		oopsErr := oops.In(domain).Code(code).With("key", "value").Errorf("%s", message)
		classified := Wrap(oopsErr, errorfamily.Transient)

		if classified.Error() == "" {
			t.Error("Wrap produced empty Error()")
		}
		if !errors.Is(classified, oopsErr) {
			t.Error("original OopsError lost in Unwrap chain")
		}
		if classified.ErrorCode() != code {
			t.Errorf("ErrorCode() = %q, want %q", classified.ErrorCode(), code)
		}

		ctx := classified.ErrorContext()
		if domain != "" && ctx["domain"] != domain {
			t.Errorf("ErrorContext()[domain] = %q, want %q", ctx["domain"], domain)
		}
	})
}

func FuzzFormat(f *testing.F) {
	f.Add("transient", "test message")
	f.Add("rejection", "bad input")

	f.Fuzz(func(t *testing.T, familyStr, message string) {
		family := errorfamily.ParseFamily(familyStr)

		plainErr := errors.New(message)
		classified := Wrap(plainErr, family)

		v := fmt.Sprintf("%v", classified)
		if v == "" {
			t.Error("v produced empty string")
		}

		s := fmt.Sprintf("%s", classified)
		if s == "" {
			t.Error("s produced empty string")
		}

		verbose := fmt.Sprintf("%+v", classified)
		if verbose == "" {
			t.Error("verbose produced empty string")
		}
	})
}
