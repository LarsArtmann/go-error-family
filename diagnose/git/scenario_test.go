package git

import (
	"context"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

func TestGitRuleMockNotARepo(t *testing.T) {
	mr := newMockRunner()
	r := &GitRule{Runner: mr}
	err := errorfamily.NewRejection("git.error", "msg").WithContext("repo", "/nonexistent")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertDetail(t, result, "is_repo", "false")
	diagnose.AssertStatus(t, result, diagnose.StatusFailed)
	if !strings.Contains(result.Fix.Command, "git init") {
		t.Errorf("Expected git init suggestion, got %q", result.Fix.Command)
	}
}

func TestGitRuleMockNoGitBinary(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.ExistsMap["git"] = false
	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertStatus(t, result, diagnose.StatusUnknown)
	if !strings.Contains(result.Summary, "not found") {
		t.Errorf("Expected 'not found' in summary, got %q", result.Summary)
	}
}

func TestGitRuleMockCleanWorkingTree(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.ExistsMap["git"] = true
	mr.Set("git -C "+tmpDir+" status --porcelain", "", 0)
	mr.Set("git -C "+tmpDir+" remote", "", 0)

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertDetail(t, result, "is_repo", "true")
	diagnose.AssertDetail(t, result, "clean", "true")
	diagnose.AssertStatus(t, result, diagnose.StatusHealthy)
}

func TestGitRuleMockDirtyWorkingTree(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.ExistsMap["git"] = true
	mr.Set("git -C "+tmpDir+" status --porcelain", "?? untracked.txt\n M modified.txt", 0)

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertDetail(t, result, "clean", "false")
	diagnose.AssertDetail(t, result, "dirty_files", "2")
	diagnose.AssertStatus(t, result, diagnose.StatusDegraded)
	if !strings.Contains(result.Fix.Command, "git add") {
		t.Errorf("Expected 'git add' in fix, got %q", result.Fix.Command)
	}
}

func TestGitRuleMockMergeConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.ExistsMap["git"] = true
	mr.Set("git -C "+tmpDir+" status --porcelain", "UU file1.txt\nUU file2.txt\nAA file3.txt", 0)

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertDetail(t, result, "merge_conflicts", "true")
	diagnose.AssertStatus(t, result, diagnose.StatusFailed)
	if !strings.Contains(result.Fix.Command, "mergetool") {
		t.Errorf("Expected 'mergetool' in fix, got %q", result.Fix.Command)
	}
}

func TestGitRuleMockGitStatusFails(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.ExistsMap["git"] = true
	mr.Set("git -C "+tmpDir+" status --porcelain", "fatal: not a git object", 128)

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertStatus(t, result, diagnose.StatusUnknown)
}

func TestGitRuleMockUnreachableRemote(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.ExistsMap["git"] = true
	mr.Set("git -C "+tmpDir+" status --porcelain", "", 0)
	mr.Set("git -C "+tmpDir+" remote", "origin", 0)
	mr.Set("git -C "+tmpDir+" ls-remote --heads origin", "fatal: could not resolve", 128)

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertStatus(t, result, diagnose.StatusDegraded)
	diagnose.AssertDetail(t, result, "remote_reachable", "false")
}

func TestGitRuleMockReachableRemote(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.ExistsMap["git"] = true
	mr.Set("git -C "+tmpDir+" status --porcelain", "", 0)
	mr.Set("git -C "+tmpDir+" remote", "origin", 0)
	mr.Set("git -C "+tmpDir+" ls-remote --heads origin", "abc123\trefs/heads/main", 0)

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertStatus(t, result, diagnose.StatusHealthy)
	diagnose.AssertDetail(t, result, "remote_reachable", "true")
}

func TestGitRuleMockCallsCommandRunner(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.ExistsMap["git"] = true
	mr.Set("git -C "+tmpDir+" status --porcelain", "", 0)
	mr.Set("git -C "+tmpDir+" remote", "", 0)

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	_, _ = r.Run(context.Background(), err)

	calls := mr.Calls()
	if len(calls) == 0 {
		t.Error("Expected command runner calls, got none")
	}

	hasExists := false
	for _, c := range calls {
		if strings.HasPrefix(c, "exists:") {
			hasExists = true
		}
	}
	if !hasExists {
		t.Error("Expected Exists() call, not found")
	}
}
