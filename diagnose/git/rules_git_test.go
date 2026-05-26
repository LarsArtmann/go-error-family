package git

import (
	"context"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/diagnose"
)

func TestGitRuleApplicable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"git context", errorfamily.NewRejection("test", "msg").WithContext("git", "true"), true},
		{"git code", errorfamily.NewRejection("git.merge", "msg"), true},
		{"git substring", errorfamily.NewTransient("test", "git operation failed"), true},
		{"unrelated", errorfamily.NewTransient("db.timeout", "msg"), false},
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
}

func TestGitRuleRunCurrentDir(t *testing.T) {
	r := &GitRule{}
	err := errorfamily.NewRejection("git.status", "msg").WithContext("git", "true")

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	// Current dir IS a git repo, so it should report healthy or degraded
	if result.Status == diagnose.StatusUnknown && result.Details["is_repo"] == "false" {
		t.Error("Expected current dir to be a git repo")
	}
}
