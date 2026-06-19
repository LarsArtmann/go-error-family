package git

import (
	"context"
	"fmt"
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
			diagnose.AssertApplicable(t, &GitRule{}, tt.err, tt.want)
		})
	}
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

func newMockRunner() *diagnose.MockCommandRunner {
	return diagnose.NewMockCommandRunner()
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
