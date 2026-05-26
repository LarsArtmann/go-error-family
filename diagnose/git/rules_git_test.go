package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

func TestGitRuleName(t *testing.T) {
	r := &GitRule{}
	if got := r.Name(); got != "git" {
		t.Errorf("Name() = %q, want %q", got, "git")
	}
}

func TestGitRuleApplicable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"git context", errorfamily.NewRejection("test", "msg").WithContext("git", "true"), true},
		{"repository context", errorfamily.NewTransient("test", "msg").WithContext("repository", "/repo"), true},
		{"repo context", errorfamily.NewTransient("test", "msg").WithContext("repo", "/repo"), true},
		{"branch context", errorfamily.NewTransient("test", "msg").WithContext("branch", "main"), true},
		{"git_dir context", errorfamily.NewTransient("test", "msg").WithContext("git_dir", "/.git"), true},
		{"git code", errorfamily.NewRejection("git.merge", "msg"), true},
		{"git substring in message", errorfamily.NewTransient("test", "git operation failed"), true},
		{"unrelated code", errorfamily.NewTransient("db.timeout", "msg"), false},
		{"unrelated context", errorfamily.NewTransient("test", "msg").WithContext("host", "db"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GitRule{}
			if got := r.Applicable(tt.err); got != tt.want {
				t.Errorf("Applicable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitRuleRunNotARepo(t *testing.T) {
	r := &GitRule{}
	err := errorfamily.NewRejection("git.error", "msg").WithContext("repo", "/tmp")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["is_repo"] != "false" {
		t.Errorf("Expected is_repo=false, got %v", result.Details)
	}
	if result.Status != diagnose.StatusFailed {
		t.Errorf("Expected StatusFailed, got %v", result.Status)
	}
	if result.SuggestedFix == "" {
		t.Error("Expected non-empty SuggestedFix")
	}
}

func TestGitRuleRunCleanRepo(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["is_repo"] != "true" {
		t.Errorf("Expected is_repo=true, got %v", result.Details)
	}
	if result.Details["clean"] != "true" {
		t.Errorf("Expected clean=true, got %v", result.Details)
	}
}

func TestGitRuleRunDirtyRepo(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	untrackedFile := filepath.Join(tmpDir, "untracked.txt")
	if err := os.WriteFile(untrackedFile, []byte("dirty"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["clean"] != "false" {
		t.Errorf("Expected clean=false, got %v", result.Details)
	}
	if result.Details["dirty_files"] == "" {
		t.Error("Expected dirty_files to be set")
	}
	if result.Status != diagnose.StatusDegraded {
		t.Errorf("Expected StatusDegraded, got %v", result.Status)
	}
}

func TestGitRuleRunRepoPathFromGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("git_dir", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["is_repo"] != "true" {
		t.Errorf("Expected is_repo=true when using git_dir context, got %v", result.Details)
	}
}

func TestGitRuleRunRepoPathFromRepository(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repository", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["is_repo"] != "true" {
		t.Errorf("Expected is_repo=true when using repository context, got %v", result.Details)
	}
}

func TestGitRuleRunCleanRepoNoRemote(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["clean"] != "true" {
		t.Errorf("Expected clean=true, got details=%v", result.Details)
	}
	if result.Status != diagnose.StatusHealthy {
		t.Errorf("Expected StatusHealthy (no remotes), got %v", result.Status)
	}
}

func TestGitRuleRunCurrentDir(t *testing.T) {
	r := &GitRule{}
	err := errorfamily.NewRejection("git.status", "msg").WithContext("git", "true")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status == diagnose.StatusUnknown && result.Details["is_repo"] == "false" {
		t.Error("Expected current dir to be a git repo")
	}
}

func TestGitRuleRunCanceledContext(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(ctx, err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	_ = result
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	commitFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(commitFile, []byte("# test"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	stdout, exitCode, err := diagnose.RunCommand(
		context.Background(),
		testTimeout,
		"git",
		append([]string{"-C", dir}, args...)...,
	)
	if err != nil {
		t.Fatalf("git %v: %v (stdout=%s, exitCode=%d)", args, err, stdout, exitCode)
	}
	if exitCode != 0 {
		t.Fatalf("git %v: exitCode=%d stdout=%s", args, exitCode, stdout)
	}
}

const testTimeout = 5 * time.Second
