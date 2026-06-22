package diagnose

import (
	"context"
	"path/filepath"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestNetworkRuleApplicable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			"host context",
			errorfamily.NewTransient("test", "msg").WithContext("host", "example.com"),
			true,
		},
		{"connect code", errorfamily.NewTransient("network.connect", "msg"), true},
		{"timeout code", errorfamily.NewTransient("timeout", "msg"), true},
		{"unrelated", errorfamily.NewRejection("file.not_found", "msg"), false},
		{
			"connection refused substring",
			errorfamily.NewTransient("test", "connection refused"),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertApplicable(t, &NetworkRule{}, tt.err, tt.want)
		})
	}
}

func TestFilesystemRuleApplicable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			"path context",
			errorfamily.NewRejection("test", "msg").WithContext("path", "/etc/config"),
			true,
		},
		{"file code", errorfamily.NewRejection("file.not_found", "msg"), true},
		{"config code", errorfamily.NewRejection("config.invalid", "msg"), true},
		{"unrelated", errorfamily.NewTransient("db.timeout", "msg"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertApplicable(t, &FilesystemRule{}, tt.err, tt.want)
		})
	}
}

func TestFilesystemRuleRunExistingFile(t *testing.T) {
	r := &FilesystemRule{}
	err := errorfamily.NewRejection("file.not_found", "msg").WithContext("path", "/etc/hostname")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["exists"] != "true" {
		t.Errorf("Expected file to exist, got details: %v", result.Details)
	}
}

func TestFilesystemRuleRunNonexistentPath(t *testing.T) {
	r := &FilesystemRule{}
	err := errorfamily.NewRejection("file.not_found", "msg").
		WithContext("path", "/nonexistent/path/that/does/not/exist")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusFailed)
	AssertDetail(t, result, "exists", "false")
}

func TestFilesystemRuleRunNoPath(t *testing.T) {
	r := &FilesystemRule{}
	err := errorfamily.NewRejection("file.error", "msg")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusUnknown)
}

func TestNetworkRuleRunNoHost(t *testing.T) {
	r := &NetworkRule{}
	err := errorfamily.NewTransient("timeout", "msg")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	_ = result
}

func TestNetworkRuleResolveHostWithURL(t *testing.T) {
	r := &NetworkRule{}
	err := errorfamily.NewTransient("test", "msg").
		WithContext("host", "https://example.com:8080/path")
	if host := r.resolveHost(err); host != "example.com" {
		t.Errorf("resolveHost with URL = %q, want 'example.com'", host)
	}
}

func TestParentDir(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/a/b/c", "/a/b"},
		{"/a/b", "/a"},
		{"/a", "/"},
		{"relative/path", "relative"},
		{"nopath", "."},
	}
	for _, tt := range tests {
		if got := filepath.Dir(tt.path); got != tt.want {
			t.Errorf("filepath.Dir(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
