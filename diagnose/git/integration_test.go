package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

func TestGitRuleIntegrationNotARepo(t *testing.T) {
	r := &GitRule{}
	err := errorfamily.NewRejection("git.error", "msg").WithContext("repo", "/tmp")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertDetail(t, result, "is_repo", "false")
	diagnose.AssertStatus(t, result, diagnose.StatusFailed)
}

func TestGitRuleIntegrationCleanRepo(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertDetail(t, result, "is_repo", "true")
	diagnose.AssertDetail(t, result, "clean", "true")
}

func TestGitRuleIntegrationDirtyRepo(t *testing.T) {
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
	diagnose.AssertDetail(t, result, "clean", "false")
	if result.Details["dirty_files"] == "" {
		t.Error("Expected dirty_files to be set")
	}
	diagnose.AssertStatus(t, result, diagnose.StatusDegraded)
}

func TestGitRuleIntegrationRepoPathFromGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("git_dir", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertDetail(t, result, "is_repo", "true")
}

func TestGitRuleIntegrationRepoPathFromRepository(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repository", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertDetail(t, result, "is_repo", "true")
}

func TestGitRuleIntegrationCleanRepoNoRemote(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	r := &GitRule{}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	diagnose.AssertDetail(t, result, "clean", "true")
	diagnose.AssertStatus(t, result, diagnose.StatusHealthy)
}

func TestGitRuleIntegrationCurrentDir(t *testing.T) {
	r := &GitRule{}
	err := errorfamily.NewRejection("git.status", "msg").WithContext("git", "true")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	_ = result
}

func TestGitRuleIntegrationCanceledContext(t *testing.T) {
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
