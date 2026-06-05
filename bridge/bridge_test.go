package bridge

import (
	"errors"
	"fmt"
	"testing"
	errors2 "errors"

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
	if ctx["tags"] == "" {
		t.Error("ErrorContext()[tags] is empty, want tags present")
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

	if classified.OopsError.Domain() != "database" {
		t.Errorf("Domain() = %q, want %q", classified.OopsError.Domain(), "database")
	}
}

func TestWrap_UnwrapChains(t *testing.T) {
	inner := errors.New("root cause")
	wrapped := oops.Wrap(inner)
	classified := Wrap(wrapped, errorfamily.Transient)

	if !errors.Is(classified, inner) {
		t.Error("errors.Is(classified, inner) = false, want true — Unwrap chain broken")
	}
}

func TestWrap_UnwrapPlainError(t *testing.T) {
	inner := errors.New("plain error")
	classified := Wrap(inner, errorfamily.Rejection)

	if !errors.Is(classified, inner) {
		t.Error("errors.Is(classified, inner) = false — plain error not in Unwrap chain")
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

	coded, ok := errors2.AsType[errorfamily.Coded](classified)
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

func TestWrap_ErrorsIs_Delegates(t *testing.T) {
	inner := errors.New("root")
	wrapped := oops.Wrap(inner)
	classified := Wrap(wrapped, errorfamily.Transient)

	if !errors.Is(classified, inner) {
		t.Error("errors.Is should reach inner error through OopsError chain")
	}
}

func TestWrap_ErrorsIs_PlainError(t *testing.T) {
	sentinel := errors.New("sentinel")
	classified := Wrap(sentinel, errorfamily.Rejection)

	if !errors.Is(classified, sentinel) {
		t.Error("errors.Is should match original plain error")
	}
}

func TestWrap_Format(t *testing.T) {
	t.Run("%v oops", func(t *testing.T) {
		base := oops.Errorf("test msg")
		classified := Wrap(base, errorfamily.Transient)
		got := fmt.Sprintf("%v", classified)
		if got == "" {
			t.Errorf("%%v produced empty string")
		}
	})

	t.Run("%v plain", func(t *testing.T) {
		plain := errors.New("plain")
		classified := Wrap(plain, errorfamily.Rejection)
		got := fmt.Sprintf("%v", classified)
		if got != "[rejection] plain" {
			t.Errorf("%%v = %q, want [rejection] plain", got)
		}
	})

	t.Run("%+v oops", func(t *testing.T) {
		base := oops.In("database").With("host", "db1").Errorf("test")
		classified := Wrap(base, errorfamily.Transient)
		got := fmt.Sprintf("%+v", classified)
		if got == "" {
			t.Error("verbose produced empty string")
		}
	})

	t.Run("%s oops", func(t *testing.T) {
		base := oops.Errorf("short")
		classified := Wrap(base, errorfamily.Transient)
		got := fmt.Sprintf("%s", classified)
		if got == "" {
			t.Errorf("%%s produced empty string")
		}
	})

	t.Run("%s plain", func(t *testing.T) {
		plain := errors.New("short")
		classified := Wrap(plain, errorfamily.Rejection)
		got := fmt.Sprintf("%s", classified)
		if got != "short" {
			t.Errorf("%%s = %q, want short", got)
		}
	})
}

func TestInferFamily_NilInput(t *testing.T) {
	family := InferFamily(nil)
	if family != errorfamily.Transient {
		t.Errorf("InferFamily(nil) = %v, want Transient", family)
	}
}

func TestInferFamily_PlainError(t *testing.T) {
	family := InferFamily(errors.New("plain"))
	if family != errorfamily.Transient {
		t.Errorf("InferFamily(plain error) = %v, want Transient", family)
	}
}

func TestInferFamily_ExplicitTagsOverrideDomain(t *testing.T) {
	err := oops.In("database").Tags("rejection").Errorf("test")
	family := InferFamily(err)
	if family != errorfamily.Rejection {
		t.Errorf("tag 'rejection' should override domain 'database' (Transient), got %v", family)
	}
}

func TestInferFamily_TagRetryable(t *testing.T) {
	err := oops.In("validation").Tags("retryable").Errorf("test")
	family := InferFamily(err)
	if family != errorfamily.Transient {
		t.Errorf("tag 'retryable' should force Transient regardless of domain, got %v", family)
	}
}

func TestInferFamily_AllTagOverrides(t *testing.T) {
	cases := []struct {
		tag        string
		wantFamily errorfamily.Family
	}{
		{"retryable", errorfamily.Transient},
		{"transient", errorfamily.Transient},
		{"conflict", errorfamily.Conflict},
		{"corruption", errorfamily.Corruption},
		{"corrupted", errorfamily.Corruption},
		{"rejection", errorfamily.Rejection},
		{"rejected", errorfamily.Rejection},
		{"infrastructure", errorfamily.Infrastructure},
		{"infra", errorfamily.Infrastructure},
	}

	for _, tc := range cases {
		t.Run(tc.tag, func(t *testing.T) {
			err := oops.Tags(tc.tag).Errorf("test")
			got := InferFamily(err)
			if got != tc.wantFamily {
				t.Errorf("tag %q → %v, want %v", tc.tag, got, tc.wantFamily)
			}
		})
	}
}

func TestInferFamily_AllDomainDefaults(t *testing.T) {
	cases := []struct {
		domain     string
		wantFamily errorfamily.Family
	}{
		{"validation", errorfamily.Rejection},
		{"auth", errorfamily.Rejection},
		{"authorization", errorfamily.Rejection},
		{"input", errorfamily.Rejection},
		{"database", errorfamily.Transient},
		{"network", errorfamily.Transient},
		{"cache", errorfamily.Transient},
		{"queue", errorfamily.Transient},
		{"storage", errorfamily.Infrastructure},
		{"infra", errorfamily.Infrastructure},
		{"infrastructure", errorfamily.Infrastructure},
		{"startup", errorfamily.Infrastructure},
		{"data", errorfamily.Corruption},
		{"schema", errorfamily.Corruption},
		{"migration", errorfamily.Corruption},
	}

	for _, tc := range cases {
		t.Run(tc.domain, func(t *testing.T) {
			err := oops.In(tc.domain).Errorf("test")
			got := InferFamily(err)
			if got != tc.wantFamily {
				t.Errorf("domain %q → %v, want %v", tc.domain, got, tc.wantFamily)
			}
		})
	}
}

func TestInferFamily_UnknownDomainFailsOpen(t *testing.T) {
	err := oops.In("unknown_domain").Errorf("test")
	family := InferFamily(err)
	if family != errorfamily.Transient {
		t.Errorf("unknown domain should default to Transient (fail-open), got %v", family)
	}
}

func TestInferFamily_NoDomainNoTags(t *testing.T) {
	err := oops.Errorf("plain error")
	family := InferFamily(err)
	if family != errorfamily.Transient {
		t.Errorf("no domain, no tags should default to Transient, got %v", family)
	}
}

func TestInferFamily_MultipleTagsFirstMatchWins(t *testing.T) {
	err := oops.Tags("rejection", "retryable").Errorf("test")
	family := InferFamily(err)
	if family != errorfamily.Rejection {
		t.Errorf("first matching tag 'rejection' should win, got %v", family)
	}
}

func TestAutoWrap_InfersAndWraps(t *testing.T) {
	err := oops.In("database").With("host", "db1").Errorf("timeout")
	classified := AutoWrap(err)

	family := errorfamily.Classify(classified)
	if family != errorfamily.Transient {
		t.Errorf("AutoWrap(database domain) classify = %v, want Transient", family)
	}
	if classified.Family() != errorfamily.Transient {
		t.Errorf("AutoWrap(database domain).Family() = %v, want Transient", classified.Family())
	}
	if !errorfamily.IsRetryable(classified) {
		t.Error("AutoWrap(database domain) should be retryable")
	}
}

func TestAutoWrap_ValidationDomainIsRejection(t *testing.T) {
	err := oops.In("validation").Errorf("bad input")
	classified := AutoWrap(err)

	if errorfamily.IsRetryable(classified) {
		t.Error("validation domain should not be retryable")
	}
	if classified.Family() != errorfamily.Rejection {
		t.Errorf("validation domain should be Rejection, got %v", classified.Family())
	}
}

func TestAutoWrap_TagOverridesDomain(t *testing.T) {
	err := oops.In("validation").Tags("retryable").Errorf("override")
	classified := AutoWrap(err)

	if !errorfamily.IsRetryable(classified) {
		t.Error("tag 'retryable' should override validation domain to Transient")
	}
}

func TestAutoWrap_PlainError(t *testing.T) {
	err := errors.New("plain stdlib error")
	classified := AutoWrap(err)

	if classified.Family() != errorfamily.Transient {
		t.Errorf("plain error should infer Transient (fail-open), got %v", classified.Family())
	}
	if !errors.Is(classified, err) {
		t.Error("plain error should be reachable via errors.Is")
	}
}

func TestBridge_Integration_FullStack(t *testing.T) {
	err := oops.In("database").
		Tags("timeout").
		Code("db.timeout").
		With("host", "db1").
		With("port", 5432).
		Errorf("connection failed after 5000ms")

	classified := AutoWrap(err)

	if errorfamily.ExitCode(classified) != 75 {
		t.Errorf("ExitCode = %d, want 75 (EX_TEMPFAIL)", errorfamily.ExitCode(classified))
	}

	ctx := classified.ErrorContext()
	if ctx["host"] != "db1" {
		t.Errorf("context host = %q, want %q", ctx["host"], "db1")
	}
	if ctx["domain"] != "database" {
		t.Errorf("context domain = %q, want %q", ctx["domain"], "database")
	}

	if !errorfamily.IsRetryable(classified) {
		t.Error("database timeout should be retryable")
	}

	if classified.OopsError.Domain() != "database" {
		t.Errorf("oops domain = %q, want %q", classified.OopsError.Domain(), "database")
	}

	if classified.ErrorCode() != "db.timeout" {
		t.Errorf("ErrorCode() = %q, want %q", classified.ErrorCode(), "db.timeout")
	}
}

func BenchmarkWrap(b *testing.B) {
	base := oops.With("host", "db1").With("port", "5432").Errorf("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Wrap(base, errorfamily.Transient)
	}
}

func BenchmarkInferFamily(b *testing.B) {
	err := oops.In("database").Tags("timeout").With("host", "db1").Errorf("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InferFamily(err)
	}
}

func BenchmarkAutoWrap(b *testing.B) {
	err := oops.In("database").Tags("timeout").With("host", "db1").Errorf("test")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AutoWrap(err)
	}
}

func BenchmarkErrorContext(b *testing.B) {
	base := oops.
		With("host", "db1").
		With("port", "5432").
		With("user", "admin").
		Errorf("test")
	classified := Wrap(base, errorfamily.Transient)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = classified.ErrorContext()
	}
}

func ExampleAutoWrap() {
	err := oops.In("database").
		Tags("timeout").
		Code("db.timeout").
		With("host", "db1.example.com").
		Errorf("connection failed")

	classified := AutoWrap(err)

	fmt.Println(errorfamily.IsRetryable(classified))
	fmt.Println(errorfamily.ExitCode(classified))
	fmt.Println(classified.Family().String())
	fmt.Println(classified.ErrorCode())
	// Output:
	// true
	// 75
	// transient
	// db.timeout
}

func ExampleInferFamily_tagOverride() {
	err := oops.In("validation").Tags("retryable").Errorf("override domain default")
	classified := AutoWrap(err)

	fmt.Println(classified.Family().String())
	// Output: transient
}

func ExampleWrap_plainError() {
	err := errors.New("file not found")
	classified := Wrap(err, errorfamily.Rejection)

	fmt.Println(classified.Error())
	fmt.Println(classified.Family().String())
	fmt.Println(errors.Is(classified, err))
	// Output:
	// [rejection] file not found
	// rejection
	// true
}
