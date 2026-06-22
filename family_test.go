package errorfamily

import (
	"testing"
)

type familyStringCase = struct {
	family Family
	want   string
}

type familyIntCase = struct {
	family Family
	want   int
}

func TestFamilyString(t *testing.T) {
	testFamilyProperty(t, "String", []familyStringCase{
		{Rejection, "rejection"},
		{Conflict, "conflict"},
		{Transient, "transient"},
		{Corruption, "corruption"},
		{Infrastructure, "infrastructure"},
		{Family(99), "unknown"},
	}, Family.String)
}

func TestParseFamily(t *testing.T) {
	tests := []struct {
		input string
		want  Family
	}{
		{"rejection", Rejection},
		{"REJECTION", Rejection},
		{"conflict", Conflict},
		{"transient", Transient},
		{"corruption", Corruption},
		{"infrastructure", Infrastructure},
		{"unknown", Transient}, // default
		{"garbage", Transient}, // default
	}
	for _, tt := range tests {
		if got := ParseFamily(tt.input); got != tt.want {
			t.Errorf("ParseFamily(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFamilySeverity(t *testing.T) {
	tests := []struct {
		family Family
		want   int
	}{
		{Transient, 1},
		{Rejection, 2},
		{Conflict, 3},
		{Infrastructure, 4},
		{Corruption, 5},
		{Family(99), 0}, // invalid → 0
	}
	for _, tt := range tests {
		if got := tt.family.Severity(); got != tt.want {
			t.Errorf("Family(%d).Severity() = %d, want %d", tt.family, got, tt.want)
		}
	}
}

func TestFamilyIsRetryable(t *testing.T) {
	if Rejection.IsRetryable() {
		t.Error("Rejection should not be retryable")
	}
	if Conflict.IsRetryable() {
		t.Error("Conflict should not be retryable")
	}
	if !Transient.IsRetryable() {
		t.Error("Transient should be retryable")
	}
	if Corruption.IsRetryable() {
		t.Error("Corruption should not be retryable")
	}
	if Infrastructure.IsRetryable() {
		t.Error("Infrastructure should not be retryable")
	}
}

func TestFamilyExitCode(t *testing.T) {
	testFamilyProperty(t, "ExitCode", []familyIntCase{
		{Rejection, 1},
		{Conflict, 1},
		{Transient, 75},
		{Corruption, 65},
		{Infrastructure, 69},
		{Family(99), 70},
	}, Family.ExitCode)
}

func TestFamilyTone(t *testing.T) {
	if Rejection.Tone() != ToneInstructional {
		t.Error("Rejection should have instructional tone")
	}
	if Transient.Tone() != ToneReassuring {
		t.Error("Transient should have reassuring tone")
	}
}

func TestFamilyDefaultMessageAll(t *testing.T) {
	testFamilyProperty(t, "DefaultMessage", []familyStringCase{
		{Rejection, "The request was invalid. Check your input and try again."},
		{Conflict, "A conflict was detected. Refresh and try again."},
		{Transient, "A temporary error occurred. Please try again in a few moments."},
		{Corruption, "Data appears to be corrupted. This requires manual intervention."},
		{Infrastructure, "The service is currently unavailable. Please try again later."},
		{Family(99), "An unexpected error occurred."},
	}, Family.DefaultMessage)
}

func TestFamilyDefaultWhyAll(t *testing.T) {
	testFamilyProperty(t, "DefaultWhy", []familyStringCase{
		{Rejection, ""},
		{Conflict, ""},
		{Transient, "This is a temporary issue. No data was lost."},
		{Corruption, "Some data appears to be damaged. This requires attention."},
		{Infrastructure, "This is a system issue, not something you caused."},
		{Family(99), ""},
	}, Family.DefaultWhy)
}

func TestFamilyDefaultFixAll(t *testing.T) {
	testFamilyProperty(t, "DefaultFix", []familyStringCase{
		{Rejection, "Check your input and try again."},
		{Conflict, "Refresh your data and try the operation again."},
		{Transient, "Wait a moment and try again."},
		{Corruption, "This may require manual intervention. Check the logs for details."},
		{Infrastructure, "The service may be temporarily unavailable. Try again later."},
		{Family(99), "Try again or contact support."},
	}, Family.DefaultFix)
}

func TestFamilyToneAll(t *testing.T) {
	testFamilyProperty(t, "Tone", []struct {
		family Family
		want   Tone
	}{
		{Rejection, ToneInstructional},
		{Conflict, ToneExplanatory},
		{Transient, ToneReassuring},
		{Corruption, ToneUrgent},
		{Infrastructure, ToneApologetic},
		{Family(99), ToneApologetic},
	}, Family.Tone)
}

func TestAudienceString(t *testing.T) {
	tests := []struct {
		a    Audience
		want string
	}{
		{AudienceUser, "user"},
		{AudienceOps, "ops"},
		{AudienceAll, "all"},
		{Audience(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.a.String(); got != tt.want {
			t.Errorf("Audience(%d).String() = %q, want %q", tt.a, got, tt.want)
		}
	}
}

func TestAudienceIsValid(t *testing.T) {
	tests := []struct {
		a    Audience
		want bool
	}{
		{AudienceUser, true},
		{AudienceOps, true},
		{AudienceAll, true},
		{Audience(42), false},
		{Audience(-1), false},
	}
	for _, tt := range tests {
		t.Run(tt.a.String(), func(t *testing.T) {
			if got := tt.a.IsValid(); got != tt.want {
				t.Errorf("Audience(%d).IsValid() = %v, want %v", tt.a, got, tt.want)
			}
		})
	}
}

func TestFamilyAudience(t *testing.T) {
	tests := []struct {
		family Family
		want   Audience
	}{
		{Rejection, AudienceUser},
		{Conflict, AudienceUser},
		{Transient, AudienceAll},
		{Corruption, AudienceOps},
		{Infrastructure, AudienceOps},
		{Family(99), AudienceOps},
	}
	for _, tt := range tests {
		t.Run(tt.family.String(), func(t *testing.T) {
			if got := tt.family.Audience(); got != tt.want {
				t.Errorf("Family(%v).Audience() = %v, want %v", tt.family, got, tt.want)
			}
		})
	}
}

func TestParseAudience(t *testing.T) {
	tests := []struct {
		input string
		want  Audience
	}{
		{"user", AudienceUser},
		{"USER", AudienceUser},
		{"ops", AudienceOps},
		{"all", AudienceAll},
		{"unknown", AudienceUser},
		{"garbage", AudienceUser},
	}
	for _, tt := range tests {
		if got := ParseAudience(tt.input); got != tt.want {
			t.Errorf("ParseAudience(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestFamilyMarshalText(t *testing.T) {
	tests := []struct {
		family Family
		want   string
	}{
		{Rejection, "rejection"},
		{Transient, "transient"},
		{Infrastructure, "infrastructure"},
	}
	for _, tt := range tests {
		got, err := tt.family.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText() error: %v", err)
		}
		if string(got) != tt.want {
			t.Errorf("MarshalText() = %q, want %q", got, tt.want)
		}
	}
}

func TestFamilyUnmarshalText(t *testing.T) {
	tests := []struct {
		input string
		want  Family
	}{
		{"rejection", Rejection},
		{"TRANSIENT", Transient},
		{"unknown", Transient},
		{"garbage", Transient},
	}
	for _, tt := range tests {
		var f Family
		if err := f.UnmarshalText([]byte(tt.input)); err != nil {
			t.Fatalf("UnmarshalText(%q) error: %v", tt.input, err)
		}
		if f != tt.want {
			t.Errorf("UnmarshalText(%q) = %v, want %v", tt.input, f, tt.want)
		}
	}
}

func TestAudienceMarshalText(t *testing.T) {
	got, err := AudienceOps.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error: %v", err)
	}
	if string(got) != "ops" {
		t.Errorf("MarshalText() = %q, want %q", got, "ops")
	}
}

func TestAudienceUnmarshalText(t *testing.T) {
	var a Audience
	if err := a.UnmarshalText([]byte("ALL")); err != nil {
		t.Fatalf("UnmarshalText() error: %v", err)
	}
	if a != AudienceAll {
		t.Errorf("UnmarshalText() = %v, want %v", a, AudienceAll)
	}
}

func TestFamilyHTTPStatus(t *testing.T) {
	testFamilyProperty(t, "HTTPStatus", []familyIntCase{
		{Rejection, 400},
		{Conflict, 409},
		{Transient, 503},
		{Corruption, 500},
		{Infrastructure, 503},
		{Family(99), 500},
	}, Family.HTTPStatus)
}
