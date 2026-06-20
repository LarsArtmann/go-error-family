package diagnose

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMockCommandRunnerRunExactMatch(t *testing.T) {
	m := NewMockCommandRunner()
	m.Responses["git status"] = MockResponse{Stdout: "clean", ExitCode: 0}

	stdout, code, err := m.Run(context.Background(), time.Second, "git", "status")
	if stdout != "clean" || code != 0 || err != nil {
		t.Errorf("exact match = %q/%d/%v, want clean/0/<nil>", stdout, code, err)
	}
}

func TestMockCommandRunnerRunPrefixMatch(t *testing.T) {
	m := NewMockCommandRunner()
	m.Responses["git"] = MockResponse{Stdout: "git-output"}

	// No exact match, but "git remote -v" has prefix "git".
	stdout, _, _ := m.Run(context.Background(), time.Second, "git", "remote", "-v")
	if stdout != "git-output" {
		t.Errorf("prefix match = %q, want git-output", stdout)
	}
}

func TestMockCommandRunnerRunDefaultEmpty(t *testing.T) {
	m := NewMockCommandRunner()
	stdout, code, err := m.Run(context.Background(), time.Second, "unknown", "cmd")
	if stdout != "" || code != 0 || err != nil {
		t.Errorf("default = %q/%d/%v, want \"\"/0/<nil>", stdout, code, err)
	}
}

func TestMockCommandRunnerRunError(t *testing.T) {
	m := NewMockCommandRunner()
	sentinel := errors.New("boom")
	m.Responses["fail cmd"] = MockResponse{Err: sentinel}

	_, _, err := m.Run(context.Background(), time.Second, "fail", "cmd")
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want sentinel", err)
	}
}

func TestMockCommandRunnerExists(t *testing.T) {
	m := NewMockCommandRunner()
	m.ExistsMap["git"] = true
	m.ExistsMap["missing"] = false

	if !m.Exists("git") {
		t.Error("Exists(git) = false, want true")
	}
	if m.Exists("missing") {
		t.Error("Exists(missing) = true, want false")
	}
}

func TestMockCommandRunnerCallsRecording(t *testing.T) {
	m := NewMockCommandRunner()
	m.ExistsMap["git"] = true

	_, _, _ = m.Run(context.Background(), time.Second, "git", "status")
	_ = m.Exists("git")

	calls := m.Calls()
	want := []string{"git status", "exists:git"}
	if len(calls) != len(want) {
		t.Fatalf("Calls() = %v, want %v", calls, want)
	}
	for i, c := range calls {
		if c != want[i] {
			t.Errorf("Calls()[%d] = %q, want %q", i, c, want[i])
		}
	}

	// Calls() returns a copy — mutating it must not affect the runner.
	calls[0] = "mutated"
	if got := m.Calls()[0]; got == "mutated" {
		t.Error("Calls() returned an internal slice reference")
	}
}
