package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		{
			"repository context",
			errorfamily.NewTransient("test", "msg").WithContext("repository", "/repo"),
			true,
		},
		{
			"repo context",
			errorfamily.NewTransient("test", "msg").WithContext("repo", "/repo"),
			true,
		},
		{
			"branch context",
			errorfamily.NewTransient("test", "msg").WithContext("branch", "main"),
			true,
		},
		{
			"git_dir context",
			errorfamily.NewTransient("test", "msg").WithContext("git_dir", "/.git"),
			true,
		},
		{"git code", errorfamily.NewRejection("git.merge", "msg"), true},
		{
			"git substring in message",
			errorfamily.NewTransient("test", "msg").WithContext("msg", "git operation failed"),
			true,
		},
		{"unrelated code", errorfamily.NewTransient("db.timeout", "msg"), false},
		{
			"unrelated context",
			errorfamily.NewTransient("test", "msg").WithContext("host", "db"),
			false,
		},
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

func newMockRunner() *diagnose.MockCommandRunner {
	return diagnose.NewMockCommandRunner()
}

func TestGitRuleMockNotARepo(t *testing.T) {
	mr := newMockRunner()
	r := &GitRule{Runner: mr}
	err := errorfamily.NewRejection("git.error", "msg").WithContext("repo", "/nonexistent")

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
	if !strings.Contains(result.SuggestedFix, "git init") {
		t.Errorf("Expected git init suggestion, got %q", result.SuggestedFix)
	}
}

func TestGitRuleMockNoGitBinary(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.Exists_["git"] = false
	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != diagnose.StatusUnknown {
		t.Errorf("Expected StatusUnknown when git not found, got %v", result.Status)
	}
	if !strings.Contains(result.Summary, "not found") {
		t.Errorf("Expected 'not found' in summary, got %q", result.Summary)
	}
}

func TestGitRuleMockCleanWorkingTree(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.Exists_["git"] = true
	mr.Responses["git -C "+tmpDir+" status --porcelain"] = diagnose.MockResponse{Stdout: "", ExitCode: 0}
	mr.Responses["git -C "+tmpDir+" remote"] = diagnose.MockResponse{Stdout: "", ExitCode: 0}

	r := &GitRule{Runner: mr}
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
	if result.Status != diagnose.StatusHealthy {
		t.Errorf("Expected StatusHealthy, got %v", result.Status)
	}
}

func TestGitRuleMockDirtyWorkingTree(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.Exists_["git"] = true
	mr.Responses["git -C "+tmpDir+" status --porcelain"] = diagnose.MockResponse{
		Stdout:   "?? untracked.txt\n M modified.txt",
		ExitCode: 0,
	}

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["clean"] != "false" {
		t.Errorf("Expected clean=false, got %v", result.Details)
	}
	if result.Details["dirty_files"] != "2" {
		t.Errorf("Expected dirty_files=2, got %v", result.Details["dirty_files"])
	}
	if result.Status != diagnose.StatusDegraded {
		t.Errorf("Expected StatusDegraded, got %v", result.Status)
	}
	if !strings.Contains(result.SuggestedFix, "git add") {
		t.Errorf("Expected 'git add' in fix, got %q", result.SuggestedFix)
	}
}

func TestGitRuleMockMergeConflicts(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.Exists_["git"] = true
	mr.Responses["git -C "+tmpDir+" status --porcelain"] = diagnose.MockResponse{
		Stdout:   "UU file1.txt\nUU file2.txt\nAA file3.txt",
		ExitCode: 0,
	}

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Details["merge_conflicts"] != "true" {
		t.Errorf("Expected merge_conflicts=true, got %v", result.Details)
	}
	if result.Status != diagnose.StatusFailed {
		t.Errorf("Expected StatusFailed, got %v", result.Status)
	}
	if !strings.Contains(result.SuggestedFix, "mergetool") {
		t.Errorf("Expected 'mergetool' in fix, got %q", result.SuggestedFix)
	}
}

func TestGitRuleMockGitStatusFails(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.Exists_["git"] = true
	mr.Responses["git -C "+tmpDir+" status --porcelain"] = diagnose.MockResponse{
		Stdout:   "fatal: not a git object",
		ExitCode: 128,
	}

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != diagnose.StatusUnknown {
		t.Errorf("Expected StatusUnknown on git status failure, got %v", result.Status)
	}
}

func TestGitRuleMockUnreachableRemote(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.Exists_["git"] = true
	mr.Responses["git -C "+tmpDir+" status --porcelain"] = diagnose.MockResponse{Stdout: "", ExitCode: 0}
	mr.Responses["git -C "+tmpDir+" remote"] = diagnose.MockResponse{Stdout: "origin", ExitCode: 0}
	mr.Responses["git -C "+tmpDir+" ls-remote --heads origin"] = diagnose.MockResponse{
		Stdout:   "fatal: could not resolve",
		ExitCode: 128,
	}

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != diagnose.StatusDegraded {
		t.Errorf("Expected StatusDegraded, got %v", result.Status)
	}
	if result.Details["remote_reachable"] != "false" {
		t.Errorf("Expected remote_reachable=false, got %v", result.Details)
	}
}

func TestGitRuleMockReachableRemote(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.Exists_["git"] = true
	mr.Responses["git -C "+tmpDir+" status --porcelain"] = diagnose.MockResponse{Stdout: "", ExitCode: 0}
	mr.Responses["git -C "+tmpDir+" remote"] = diagnose.MockResponse{Stdout: "origin", ExitCode: 0}
	mr.Responses["git -C "+tmpDir+" ls-remote --heads origin"] = diagnose.MockResponse{
		Stdout:   "abc123\trefs/heads/main",
		ExitCode: 0,
	}

	r := &GitRule{Runner: mr}
	err := errorfamily.NewTransient("git.error", "msg").WithContext("repo", tmpDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	if result.Status != diagnose.StatusHealthy {
		t.Errorf("Expected StatusHealthy, got %v", result.Status)
	}
	if result.Details["remote_reachable"] != "true" {
		t.Errorf("Expected remote_reachable=true, got %v", result.Details)
	}
}

// Integration tests using real git.

func TestGitRuleIntegrationNotARepo(t *testing.T) {
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
	if result.Details["is_repo"] != "true" {
		t.Errorf("Expected is_repo=true, got %v", result.Details)
	}
	if result.Details["clean"] != "true" {
		t.Errorf("Expected clean=true, got %v", result.Details)
	}
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

func TestGitRuleIntegrationRepoPathFromGitDir(t *testing.T) {
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

func TestGitRuleIntegrationRepoPathFromRepository(t *testing.T) {
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

func TestGitRuleIntegrationCleanRepoNoRemote(t *testing.T) {
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

func TestGitRuleResolveRepoPath(t *testing.T) {
	tests := []struct {
		name    string
		context map[string]string
		want    string
	}{
		{"git_dir", map[string]string{"git_dir": "/git/dir"}, "/git/dir"},
		{"repository", map[string]string{"repository": "/repo/path"}, "/repo/path"},
		{"repo", map[string]string{"repo": "/repo"}, "/repo"},
		{"repo_path first", map[string]string{"repo_path": "/first"}, "/first"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GitRule{}
			err := errorfamily.NewTransient("test", "msg")
			for k, v := range tt.context {
				err = err.WithContext(k, v)
			}
			got := r.resolveRepoPath(err)
			if got != tt.want {
				t.Errorf("resolveRepoPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGitRuleMockCallsCommandRunner(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	mr := newMockRunner()
	mr.Exists_["git"] = true
	mr.Responses["git -C "+tmpDir+" status --porcelain"] = diagnose.MockResponse{Stdout: "", ExitCode: 0}
	mr.Responses["git -C "+tmpDir+" remote"] = diagnose.MockResponse{Stdout: "", ExitCode: 0}

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

var _ = fmt.Sprintf
