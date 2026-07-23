package bridge

import (
	"errors"
	"fmt"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/samber/oops"
)

func TestWrap_ClassifyReturnsAttachedFamily(t *testing.T) {
	base := oops.Errorf("something broke")
	classified := Wrap(base, errorfamily.Transient)

	family := errorfamily.Classify(classified)
	if family != errorfamily.Transient {
		t.Errorf("Classify(Wrap(err, Transient)) = %v, want Transient", family)
	}
}

func TestWrap_AllFamilies(t *testing.T) {
	cases := []struct {
		family errorfamily.Family
	}{
		{errorfamily.Rejection},
		{errorfamily.Conflict},
		{errorfamily.Transient},
		{errorfamily.Corruption},
		{errorfamily.Infrastructure},
	}

	for _, tc := range cases {
		t.Run(tc.family.String(), func(t *testing.T) {
			base := oops.Errorf("test")
			classified := Wrap(base, tc.family)

			got := errorfamily.Classify(classified)
			if got != tc.family {
				t.Errorf("Classify() = %v, want %v", got, tc.family)
			}
		})
	}
}

func TestWrap_IsRetryable(t *testing.T) {
	base := oops.Errorf("test")

	transient := Wrap(base, errorfamily.Transient)
	if !errorfamily.IsRetryable(transient) {
		t.Error("IsRetryable(Transient) = false, want true")
	}

	rejection := Wrap(base, errorfamily.Rejection)
	if errorfamily.IsRetryable(rejection) {
		t.Error("IsRetryable(Rejection) = true, want false")
	}
}

func TestWrap_ExitCode(t *testing.T) {
	cases := []struct {
		family   errorfamily.Family
		exitCode int
	}{
		{errorfamily.Rejection, 1},
		{errorfamily.Conflict, 1},
		{errorfamily.Transient, 75},
		{errorfamily.Corruption, 65},
		{errorfamily.Infrastructure, 69},
	}

	for _, tc := range cases {
		t.Run(tc.family.String(), func(t *testing.T) {
			base := oops.Errorf("test")
			classified := Wrap(base, tc.family)

			got := errorfamily.ExitCode(classified)
			if got != tc.exitCode {
				t.Errorf("ExitCode() = %d, want %d", got, tc.exitCode)
			}
		})
	}
}

func TestWrap_ErrorContext_BridgesStrings(t *testing.T) {
	base := oops.With("host", "db1.example.com").With("port", 5432).Errorf("connection failed")
	classified := Wrap(base, errorfamily.Transient)

	ctx := classified.ErrorContext()
	if ctx["host"] != "db1.example.com" {
		t.Errorf("ErrorContext()[host] = %q, want %q", ctx["host"], "db1.example.com")
	}

	if ctx["port"] != "5432" {
		t.Errorf("ErrorContext()[port] = %q, want %q (converted from int)", ctx["port"], "5432")
	}
}

func TestWrap_ErrorContext_IncludesDomain(t *testing.T) {
	base := oops.In("database").Errorf("test")
	classified := Wrap(base, errorfamily.Transient)

	ctx := classified.ErrorContext()
	if ctx["domain"] != "database" {
		t.Errorf("ErrorContext()[domain] = %q, want %q", ctx["domain"], "database")
	}
}

func TestWrap_ErrorContext_IncludesTags(t *testing.T) {
	base := oops.Tags("timeout", "connection").Errorf("test")
	classified := Wrap(base, errorfamily.Transient)

	ctx := classified.ErrorContext()
	if ctx["tags"] != "timeout,connection" {
		t.Errorf("ErrorContext()[tags] = %q, want %q", ctx["tags"], "timeout,connection")
	}
}

func TestWrap_ErrorContext_EmptyWhenNoContext(t *testing.T) {
	base := oops.Errorf("no context")
	classified := Wrap(base, errorfamily.Rejection)

	ctx := classified.ErrorContext()
	if len(ctx) != 0 {
		t.Errorf("ErrorContext() = %v, want empty map", ctx)
	}
}

func TestWrap_PreservesOopsMethods(t *testing.T) {
	base := oops.In("database").With("key", "value").Errorf("test")
	classified := Wrap(base, errorfamily.Transient)

	if classified.Domain() != "database" {
		t.Errorf("Domain() = %q, want %q", classified.Domain(), "database")
	}
}

func TestWrap_UnwrapChain(t *testing.T) {
	tests := []struct {
		name   string
		inner  error
		wrap   func(error) error
		family errorfamily.Family
	}{
		{"oops chain", errors.New("root cause"), oops.Wrap, errorfamily.Transient},
		{
			"plain error",
			errors.New("plain error"),
			func(e error) error { return e },
			errorfamily.Rejection,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classified := Wrap(tt.wrap(tt.inner), tt.family)
			if !errors.Is(classified, tt.inner) {
				t.Error("errors.Is should reach inner error through Unwrap chain")
			}
		})
	}
}

func TestWrap_ErrorDelegatesToOops(t *testing.T) {
	base := oops.Errorf("test message %d", 42)
	classified := Wrap(base, errorfamily.Transient)

	msg := classified.Error()
	if msg == "" {
		t.Error("Error() returned empty string")
	}
}

func TestWrap_ErrorForPlainError(t *testing.T) {
	plain := errors.New("plain msg")
	classified := Wrap(plain, errorfamily.Rejection)

	msg := classified.Error()
	if msg != "[rejection] plain msg" {
		t.Errorf("Error() = %q, want %q", msg, "[rejection] plain msg")
	}
}

func TestWrap_NilError(t *testing.T) {
	classified := Wrap(nil, errorfamily.Transient)

	if classified.Error() != "[transient]" {
		t.Errorf("Error() on nil = %q, want [transient]", classified.Error())
	}

	if classified.Unwrap() != nil {
		t.Error("Unwrap() on nil = non-nil, want nil")
	}

	ctx := classified.ErrorContext()
	if len(ctx) != 0 {
		t.Errorf("ErrorContext() on nil = %v, want empty", ctx)
	}
}

func TestWrap_PlainError_PreservesEverything(t *testing.T) {
	plain := errors.New("plain stdlib error")
	classified := Wrap(plain, errorfamily.Rejection)

	if classified.Family() != errorfamily.Rejection {
		t.Errorf("Family() = %v, want Rejection", classified.Family())
	}

	if errorfamily.IsRetryable(classified) {
		t.Error("Rejection should not be retryable")
	}

	if !errors.Is(classified, plain) {
		t.Error("original plain error not reachable via errors.Is")
	}

	if classified.ErrorCode() != "" {
		t.Errorf("ErrorCode() = %q, want empty for plain error", classified.ErrorCode())
	}
}

func TestWrap_FamilyAccessor(t *testing.T) {
	base := oops.Errorf("test")
	classified := Wrap(base, errorfamily.Corruption)

	if classified.Family() != errorfamily.Corruption {
		t.Errorf("Family() = %v, want Corruption", classified.Family())
	}
}

func TestWrap_CodedInterface(t *testing.T) {
	base := oops.Code("db.timeout").Errorf("test")
	classified := Wrap(base, errorfamily.Transient)

	if classified.ErrorCode() != "db.timeout" {
		t.Errorf("ErrorCode() = %q, want %q", classified.ErrorCode(), "db.timeout")
	}

	coded, ok := errors.AsType[errorfamily.Coded](classified)
	if !ok {
		t.Error("AsType[Coded](classified) = false, want true")
	}

	if coded.ErrorCode() != "db.timeout" {
		t.Errorf("Coded.ErrorCode() = %q, want %q", coded.ErrorCode(), "db.timeout")
	}
}

func TestWrap_CodedInterface_NonStringCode(t *testing.T) {
	base := oops.Code(418).Errorf("test")
	classified := Wrap(base, errorfamily.Rejection)

	if classified.ErrorCode() != "418" {
		t.Errorf("ErrorCode() = %q, want %q", classified.ErrorCode(), "418")
	}
}

func TestWrap_CodedInterface_NoCode(t *testing.T) {
	base := oops.Errorf("no code")
	classified := Wrap(base, errorfamily.Transient)

	if classified.ErrorCode() != "" {
		t.Errorf("ErrorCode() = %q, want empty", classified.ErrorCode())
	}
}

func TestWrap_Format(t *testing.T) {
	tests := []struct {
		name         string
		verb         string
		base         error
		family       errorfamily.Family
		wantNonEmpty bool
		want         string
	}{
		{"%v oops", "%v", oops.Errorf("test msg"), errorfamily.Transient, true, ""},
		{"%v plain", "%v", errors.New("plain"), errorfamily.Rejection, false, "[rejection] plain"},
		{
			"+v oops",
			"%+v",
			oops.In("database").With("host", "db1").Errorf("test"),
			errorfamily.Transient,
			true,
			"",
		},
		{"%s oops", "%s", oops.Errorf("short"), errorfamily.Transient, true, ""},
		{"%s plain", "%s", errors.New("short"), errorfamily.Rejection, false, "short"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classified := Wrap(tt.base, tt.family)

			got := fmt.Sprintf(tt.verb, classified)
			if tt.wantNonEmpty && got == "" {
				t.Errorf("%s produced empty string", tt.verb)
			}

			if tt.want != "" && got != tt.want {
				t.Errorf("%s = %q, want %q", tt.verb, got, tt.want)
			}
		})
	}
}
