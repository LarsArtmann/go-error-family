package diagnose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GitRule diagnoses git-related errors.
// Checks: repo existence, clean state, merge conflicts, remote reachability.
//
// Matches errors with context containing: git, repository, repo, branch,
// or error codes containing "git".
type GitRule struct{}

func (r *GitRule) Name() string { return "git" }

func (r *GitRule) Applicable(err error) bool {
	return hasContextKey(err, "git", "repository", "repo", "branch", "git_dir") ||
		errorCodeContains(err, "git") ||
		hasContextSubstring(err, "git")
}

func (r *GitRule) Run(ctx context.Context, err error) (*DiagnosticResult, error) {
	repoPath := r.resolveRepoPath(err)

	result := &DiagnosticResult{
		Details:    map[string]string{"repo_path": repoPath},
		Confidence: 0.7,
	}

	// Check 1: Is this a git repo?
	gitDir := filepath.Join(repoPath, ".git")
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		result.Status = StatusFailed
		result.Summary = fmt.Sprintf("Not a git repository: %s", repoPath)
		result.Details["is_repo"] = "false"
		result.SuggestedFix = fmt.Sprintf("Initialize a git repository:\n  cd %s && git init", repoPath)
		return result, nil
	}
	result.Details["is_repo"] = "true"

	if !commandExists("git") {
		result.Status = StatusUnknown
		result.Summary = "git command not found on PATH"
		return result, nil
	}

	// Check 2: Is the working tree clean?
	stdout, _, exitCode, _ := runCommand(ctx, 5*time.Second, "git", "-C", repoPath, "status", "--porcelain")
	if exitCode != 0 {
		result.Status = StatusUnknown
		result.Summary = fmt.Sprintf("git status failed in %s", repoPath)
		return result, nil
	}

	if strings.TrimSpace(stdout) == "" {
		result.Details["clean"] = "true"
	} else {
		result.Details["clean"] = "false"
		lineCount := len(strings.Split(strings.TrimSpace(stdout), "\n"))
		result.Details["dirty_files"] = fmt.Sprintf("%d", lineCount)

		// Check for merge conflicts.
		if strings.Contains(stdout, "UU") || strings.Contains(stdout, "AA") || strings.Contains(stdout, "DU") {
			result.Status = StatusFailed
			result.Summary = fmt.Sprintf("Merge conflicts in %s (%d unmerged files)", repoPath, strings.Count(stdout, "UU")+strings.Count(stdout, "AA"))
			result.Details["merge_conflicts"] = "true"
			result.SuggestedFix = "Resolve merge conflicts:\n  git mergetool\n  git add <resolved files>\n  git commit"
			return result, nil
		}

		result.Status = StatusDegraded
		result.Summary = fmt.Sprintf("Working tree has uncommitted changes (%d files)", lineCount)
		result.SuggestedFix = "Commit or stash changes:\n  git add . && git commit -m \"wip\"\nOr: git stash"
		return result, nil
	}

	// Check 3: Can we reach the remote?
	remotesStdout, _, _, _ := runCommand(ctx, 3*time.Second, "git", "-C", repoPath, "remote")
	if strings.TrimSpace(remotesStdout) == "" {
		result.Status = StatusHealthy
		result.Summary = fmt.Sprintf("Git repo is clean, no remotes configured: %s", repoPath)
		result.Confidence = 0.3
		return result, nil
	}

	_, _, remoteExitCode, _ := runCommand(ctx, 10*time.Second, "git", "-C", repoPath, "ls-remote", "--heads", "origin")
	if remoteExitCode != 0 {
		result.Status = StatusDegraded
		result.Summary = fmt.Sprintf("Git repo is clean but remote is unreachable: %s", repoPath)
		result.Details["remote_reachable"] = "false"
		result.SuggestedFix = "Check network connectivity and remote URL:\n  git remote -v\n  git ls-remote origin"
		return result, nil
	}

	result.Status = StatusHealthy
	result.Summary = fmt.Sprintf("Git repo is clean and remote is reachable: %s", repoPath)
	result.Details["remote_reachable"] = "true"
	result.Confidence = 0.3

	return result, nil
}

func (r *GitRule) resolveRepoPath(err error) string {
	for _, key := range []string{"git_dir", "repository", "repo", "repo_path"} {
		if v := contextValue(err, key); v != "" {
			return v
		}
	}
	// Default to current working directory.
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "."
}
