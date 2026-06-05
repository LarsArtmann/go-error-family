package bridge

import (
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/samber/oops"
)

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
