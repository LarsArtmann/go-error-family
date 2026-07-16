package errorfamily

import (
	"testing"
)

func TestWithContextAny(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		key   string
		value any
		want  string
	}{
		{"string", "s", "hello", "hello"},
		{"int", "n", 42, "42"},
		{"int64", "n", int64(99), "99"},
		{"uint", "n", uint(7), "7"},
		{"uint64", "n", uint64(123), "123"},
		{"float64", "f", 3.14, "3.14"},
		{"bool_true", "b", true, "true"},
		{"bool_false", "b", false, "false"},
		{"nil", "x", nil, ""},
		{"struct", "o", struct{ X int }{X: 5}, "{5}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := NewRejection("test", "msg").WithContextAny(tc.key, tc.value)
			if got := err.ContextValue(tc.key); got != tc.want {
				t.Errorf("ContextValue(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

func TestWithContextAnyCopyOnWrite(t *testing.T) {
	t.Parallel()

	original := NewRejection("code", "msg")
	modified := original.WithContextAny("count", 42)

	if modified == original {
		t.Error("WithContextAny should return a new pointer")
	}
	if original.HasContext("count") {
		t.Error("original should not have the new context key")
	}
}
