package diagnose

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestRunCommandSuccess(t *testing.T) {
	stdout, exitCode, err := RunCommand(context.Background(), 5*time.Second, "go", "version")
	if err != nil {
		t.Fatalf("RunCommand() error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "go") {
		t.Errorf("stdout = %q, want go version output", stdout)
	}
}

func TestRunCommandNonZeroExit(t *testing.T) {
	stdout, exitCode, err := RunCommand(context.Background(), 5*time.Second, "sh", "-c", "echo out; exit 3")
	if err != nil {
		t.Fatalf("RunCommand() error: %v (exit error must surface as exitCode, not err)", err)
	}
	if exitCode != 3 {
		t.Errorf("exitCode = %d, want 3", exitCode)
	}
	if stdout != "out" {
		t.Errorf("stdout = %q, want %q", stdout, "out")
	}
}

func TestRunCommandMissingBinary(t *testing.T) {
	_, exitCode, err := RunCommand(
		context.Background(),
		5*time.Second,
		"definitely-not-a-real-binary-xyz",
	)
	if err == nil {
		t.Fatal("RunCommand() error = nil, want exec failure")
	}
	if exitCode != -1 {
		t.Errorf("exitCode = %d, want -1 for exec failure", exitCode)
	}
	if !strings.Contains(err.Error(), "timeout=") {
		t.Errorf("err = %v, want wrapped with timeout context", err)
	}
}

func TestRunCommandTimeout(t *testing.T) {
	_, exitCode, err := RunCommand(context.Background(), 50*time.Millisecond, "sleep", "2")
	if err == nil {
		t.Fatal("RunCommand() error = nil, want timeout error")
	}
	if exitCode != -1 {
		t.Errorf("exitCode = %d, want -1 on timeout", exitCode)
	}
}

func TestResolveRunner(t *testing.T) {
	if got := ResolveRunner(nil); got != CommandRunner(DefaultCommandRunner{}) {
		t.Errorf("ResolveRunner(nil) = %v, want DefaultCommandRunner{}", got)
	}
	mock := NewMockCommandRunner()
	if got := ResolveRunner(mock); got != CommandRunner(mock) {
		t.Errorf("ResolveRunner(mock) = %v, want the mock back", got)
	}
}

func TestCommandExists(t *testing.T) {
	if !CommandExists("go") {
		t.Error("CommandExists(go) = false, want true")
	}
	if CommandExists("definitely-not-a-real-binary-xyz") {
		t.Error("CommandExists(fake) = true, want false")
	}
}

func TestDefaultCommandRunner(t *testing.T) {
	var runner CommandRunner = DefaultCommandRunner{}

	stdout, exitCode, err := runner.Run(context.Background(), 5*time.Second, "go", "version")
	if err != nil || exitCode != 0 || !strings.Contains(stdout, "go") {
		t.Errorf("Run(go version) = %q, %d, %v; want version output, 0, nil", stdout, exitCode, err)
	}

	_, exitCode, err = runner.Run(context.Background(), 5*time.Second, "sh", "-c", "exit 7")
	if err != nil || exitCode != 7 {
		t.Errorf("Run(exit 7) = %d, %v; want exitCode 7, nil err", exitCode, err)
	}

	if !runner.Exists("go") {
		t.Error("Exists(go) = false, want true")
	}
	if runner.Exists("definitely-not-a-real-binary-xyz") {
		t.Error("Exists(fake) = true, want false")
	}
}

func TestMockCommandRunnerSet(t *testing.T) {
	mr := NewMockCommandRunner()
	mr.Set("fake --flag", "line1\nline2", 2)

	stdout, exitCode, err := mr.Run(context.Background(), time.Second, "fake", "--flag")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if exitCode != 2 {
		t.Errorf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "line1\nline2" {
		t.Errorf("stdout = %q, want untrimmed mock stdout", stdout)
	}
}

func TestHelpersPlainErrorBranches(t *testing.T) {
	plain := errors.New("connection refused for dbhost:5432")

	if got := ContextValue(plain, "host"); got != "" {
		t.Errorf("ContextValue(plain) = %q, want empty", got)
	}
	gotCtx := ErrorContext(plain)
	if gotCtx == nil || len(gotCtx) != 0 {
		t.Errorf("ErrorContext(plain) = %v, want empty non-nil map", gotCtx)
	}
}

func TestHelpersContextualBranches(t *testing.T) {
	err := errorfamily.NewTransient("db.timeout", "msg").
		WithContext("host", "dbhost").
		WithContext("port", "5432")

	if got := ContextValue(err, "host"); got != "dbhost" {
		t.Errorf("ContextValue(host) = %q, want dbhost", got)
	}
	if got := ContextValue(err, "missing"); got != "" {
		t.Errorf("ContextValue(missing) = %q, want empty", got)
	}
	gotCtx := ErrorContext(err)
	if gotCtx["host"] != "dbhost" || gotCtx["port"] != "5432" {
		t.Errorf("ErrorContext() = %v, want host and port entries", gotCtx)
	}
}

func TestResolveContextKeyDefault(t *testing.T) {
	err := errorfamily.NewTransient("test", "msg")
	if got := ResolveContextKey(err, []string{"a", "b"}, "fallback"); got != "fallback" {
		t.Errorf("ResolveContextKey() = %q, want fallback", got)
	}
}

func TestFilesystemHandleStatErrorBranches(t *testing.T) {
	r := &FilesystemRule{}

	t.Run("permission denied", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("permission checks meaningless as root")
		}
		dir := t.TempDir()
		closed := filepath.Join(dir, "locked")
		if err := os.WriteFile(closed, []byte("x"), 0o000); err != nil {
			t.Fatal(err)
		}
		result := &DiagnosticResult{Details: map[string]string{}}
		out, err := r.handleStatError(result, closed, os.ErrPermission)
		if err != nil {
			t.Fatalf("handleStatError() error: %v", err)
		}
		if out.Status != StatusFailed {
			t.Errorf("Status = %v, want failed", out.Status)
		}
		if out.Details["permissions"] != "denied" {
			t.Errorf("Details[permissions] = %q, want denied", out.Details["permissions"])
		}
	})

	t.Run("other stat error", func(t *testing.T) {
		result := &DiagnosticResult{Details: map[string]string{}}
		synthetic := errors.New("input/output error")
		out, err := r.handleStatError(result, "/some/path", synthetic)
		if err != nil {
			t.Fatalf("handleStatError() error: %v", err)
		}
		if out.Status != StatusUnknown {
			t.Errorf("Status = %v, want unknown", out.Status)
		}
		if !strings.Contains(out.Summary, "Cannot stat path") {
			t.Errorf("Summary = %q, want Cannot stat path", out.Summary)
		}
	})
}

func TestStripHostBranches(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"bare host", "dbhost", "dbhost"},
		{"host with port", "dbhost:5432", "dbhost"},
		{"url with scheme", "postgres://dbhost:5432/db", "dbhost"},
		{"scheme but empty hostname falls through", "://", ""},
		{"unparsable url falls back to port strip", "http://[::1:5432", "http"},
		{"host with path no port", "dbhost/some/path", "dbhost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripHost(tt.raw); got != tt.want {
				t.Errorf("stripHost(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSortByConfidenceDropsNilAndOrders(t *testing.T) {
	low := &DiagnosticResult{Confidence: ConfidenceNone}
	high := &DiagnosticResult{Confidence: ConfidenceCertain}
	sorted := sortByConfidence([]*DiagnosticResult{low, nil, high})
	if len(sorted) != 2 {
		t.Fatalf("len = %d, want 2 (nil dropped)", len(sorted))
	}
	if sorted[0] != high || sorted[1] != low {
		t.Error("sortByConfidence did not order descending by confidence")
	}
}
