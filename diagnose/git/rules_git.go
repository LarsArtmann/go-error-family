package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-error-family/diagnose"
)

// GitRule diagnoses git-related errors.
// Checks: repo existence, clean state, merge conflicts, remote reachability.
//
// Matches errors with context containing: git, repository, repo, branch,
// or error codes containing "git".
type GitRule struct {
	// Runner is the command runner used to execute system commands.
	// Defaults to diagnose.DefaultCommandRunner{}.
	Runner diagnose.CommandRunner
}

// cmdRunner returns the configured command runner or the default.
func (r *GitRule) cmdRunner() diagnose.CommandRunner {
	return diagnose.ResolveRunner(r.Runner)
}

func (r *GitRule) Name() string { return "git" } //nolint:goconst // Rule name, not worth extracting

func (r *GitRule) Applicable(err error) bool {
	return gitSpec.Matches(err)
}

var gitSpec = diagnose.RuleSpec{ //nolint:gochecknoglobals // Immutable rule matching spec.
	ContextKeys: []diagnose.ContextKey{
		diagnose.KeyGit,
		diagnose.KeyRepository,
		diagnose.KeyRepo,
		diagnose.KeyBranch,
		diagnose.KeyGitDir,
	},
	CodeContains:  []string{"git"},
	ContextSubstr: []string{"git"},
}

//nolint:nilerr // Diagnostic rules return results, not errors; local stat errors are expected.
func (r *GitRule) Run( //nolint:hierarchical-errors // DiagnosticRule interface
	ctx context.Context,
	err error,
) (*diagnose.DiagnosticResult, error) {
	repoPath := r.resolveRepoPath(err)

	result := &diagnose.DiagnosticResult{
		Details:    map[string]string{"repo_path": repoPath},
		Confidence: diagnose.ConfidenceLikely,
		Context:    diagnose.ErrorContext(err),
	}

	// Check 1: Is this a git repo?
	gitDir := filepath.Join(repoPath, ".git")
	info, gitDirErr := os.Stat(gitDir)
	if gitDirErr != nil || !info.IsDir() {
		result.Status = diagnose.StatusFailed
		result.Summary = "Not a git repository: " + repoPath
		result.Details["is_repo"] = strFalse
		result.Fix = diagnose.Fix{
			Summary: "Initialize a git repository in " + repoPath,
			Command: "git init",
		}
		return result, nil
	}
	result.Details["is_repo"] = strTrue

	if !r.cmdRunner().Exists("git") {
		result.Status = diagnose.StatusUnknown
		result.Summary = "git command not found on PATH"
		return result, nil
	}

	// Check 2: Is the working tree clean?
	if r.checkWorkingTree(ctx, result, repoPath) {
		return result, nil
	}

	// Check 3: Can we reach the remote?
	r.checkRemote(ctx, result, repoPath)

	return result, nil
}

// checkWorkingTree returns true if the result has been set (either dirty or conflicts found).
func (r *GitRule) checkWorkingTree(
	ctx context.Context,
	result *diagnose.DiagnosticResult,
	repoPath string,
) bool {
	stdout, exitCode, _ := r.cmdRunner().Run( //nolint:hierarchical-errors
		ctx,
		5*time.Second,
		"git",
		"-C",
		repoPath,
		"status",
		"--porcelain",
	)
	if exitCode != 0 {
		result.Status = diagnose.StatusUnknown
		result.Summary = "git status failed in " + repoPath
		return true
	}

	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		result.Details["clean"] = strTrue
		return false
	}

	result.Details["clean"] = strFalse
	lineCount := len(strings.Split(trimmed, "\n"))
	result.Details["dirty_files"] = strconv.Itoa(lineCount)

	// Check for merge conflicts.
	if strings.Contains(trimmed, "UU") || strings.Contains(trimmed, "AA") ||
		strings.Contains(trimmed, "DU") {
		result.Status = diagnose.StatusFailed
		result.Summary = fmt.Sprintf(
			"Merge conflicts in %s (%d unmerged files)",
			repoPath,
			strings.Count(trimmed, "UU")+strings.Count(trimmed, "AA"),
		)
		result.Details["merge_conflicts"] = strTrue
		diagnose.SetFix(result, "Resolve merge conflicts", "git mergetool")
		return true
	}

	result.Status = diagnose.StatusDegraded
	result.Summary = fmt.Sprintf("Working tree has uncommitted changes (%d files)", lineCount)
	diagnose.SetFix(
		result,
		"Commit or stash uncommitted changes",
		`git add . && git commit -m "wip"`,
	)
	return true
}

func (r *GitRule) checkRemote(
	ctx context.Context,
	result *diagnose.DiagnosticResult,
	repoPath string,
) {
	remotesStdout, _, _ := r.cmdRunner().Run(ctx, 3*time.Second, "git", "-C", repoPath, "remote") //nolint:hierarchical-errors
	if strings.TrimSpace(remotesStdout) == "" {
		result.Status = diagnose.StatusHealthy
		result.Summary = "Git repo is clean, no remotes configured: " + repoPath
		result.Confidence = diagnose.ConfidenceNotCause
		return
	}

	_, remoteExitCode, _ := r.cmdRunner().Run( //nolint:hierarchical-errors
		ctx,
			10*time.Second,
			"git",
			"-C",
			repoPath,
			"ls-remote",
			"--heads",
			"origin",
		)
	if remoteExitCode != 0 {
		result.Status = diagnose.StatusDegraded
		result.Summary = "Git repo is clean but remote is unreachable: " + repoPath
		result.Details["remote_reachable"] = strFalse
		diagnose.SetFix(result, "Check network connectivity and remote URL", "git ls-remote origin")
		return
	}

	result.Status = diagnose.StatusHealthy
	result.Summary = "Git repo is clean and remote is reachable: " + repoPath
	result.Details["remote_reachable"] = strTrue
	result.Confidence = diagnose.ConfidenceNotCause
}

const (
	strTrue  = "true"
	strFalse = "false"
)

func (r *GitRule) resolveRepoPath(err error) string {
	if v := diagnose.ResolveContextKey(
		err,
		[]string{
			string(diagnose.KeyGitDir),
			string(diagnose.KeyRepository),
			string(diagnose.KeyRepo),
			string(diagnose.KeyRepoPath),
		},
		"",
	); v != "" {
		return v
	}
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "."
}
