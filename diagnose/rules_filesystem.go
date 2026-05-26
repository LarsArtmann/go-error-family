package diagnose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FilesystemRule diagnoses file and directory related errors.
// Checks: path existence, readability, writability, parent directory, permissions.
//
// Matches errors with context containing: path, file, dir, directory,
// or error codes containing "file", "dir", "path", "config", "permission".
type FilesystemRule struct{}

func (r *FilesystemRule) Name() string { return "filesystem" }

func (r *FilesystemRule) Applicable(err error) bool {
	return filesystemSpec.Matches(err)
}

//nolint:goconst // Spec keys are descriptive literals, not worth extracting.
var filesystemSpec = RuleSpec{
	ContextKeys: []string{
		"path",
		"file",
		"dir",
		"directory",
		"config_path",
		"output_path",
	},
	CodeContains: []string{"file", "dir", "path", "config", "permission"},
}

func (r *FilesystemRule) Run(ctx context.Context, err error) (*DiagnosticResult, error) {
	path := r.resolvePath(err)
	if path == "" {
		return &DiagnosticResult{
			Status:     StatusUnknown,
			Summary:    "No file path found in error context",
			Confidence: ConfidenceNone,
			Details:    map[string]string{},
		}, nil
	}

	result := &DiagnosticResult{
		Details:    map[string]string{"path": path},
		Confidence: ConfidenceHigh,
	}

	info, statErr := os.Stat(path)
	if statErr != nil {
		return r.handleStatError(result, path, statErr)
	}

	// Path exists.
	result.Details["exists"] = strTrue
	result.Details["type"] = "file"
	if info.IsDir() {
		result.Details["type"] = "directory"
	}
	result.Details["permissions"] = info.Mode().Perm().String()
	result.Details["size"] = fmt.Sprintf("%d bytes", info.Size())

	if info.IsDir() {
		r.checkDirWritable(result, path)
	} else {
		r.checkFileReadable(result, path)
	}

	return result, nil
}

func (r *FilesystemRule) handleStatError(
	result *DiagnosticResult,
	path string,
	statErr error,
) (*DiagnosticResult, error) {
	if os.IsNotExist(statErr) {
		result.Status = StatusFailed
		result.Summary = "Path does not exist: " + path
		result.Details["exists"] = strFalse
		r.suggestCreate(result, path)
		return result, nil
	}

	if os.IsPermission(statErr) {
		result.Status = StatusFailed
		result.Summary = "Permission denied: " + path
		result.Details["exists"] = strTrue
		result.Details["permissions"] = "denied"
		result.SuggestedFix = fmt.Sprintf(
			"Fix permissions:\n  chmod 755 %s\nOr run with appropriate privileges.",
			path,
		)
		return result, nil
	}

	result.Status = StatusUnknown
	result.Summary = fmt.Sprintf("Cannot stat path: %s: %v", path, statErr)
	return result, nil
}

func (r *FilesystemRule) suggestCreate(result *DiagnosticResult, path string) {
	parent := filepath.Dir(path)
	parentInfo, parentErr := os.Stat(parent)
	if parentErr != nil {
		result.Details["parent_exists"] = strFalse
		result.SuggestedFix = "Create parent directory and path:\n  mkdir -p " + parent
		return
	}
	result.Details["parent_exists"] = strTrue
	result.Details["parent_permissions"] = parentInfo.Mode().Perm().String()
	if strings.Contains(path, ".") {
		result.SuggestedFix = "Create the file: " + path
	} else {
		result.SuggestedFix = "Create directory: mkdir -p " + path
	}
}

func (r *FilesystemRule) checkDirWritable(result *DiagnosticResult, path string) {
	testFile := fmt.Sprintf("%s/.write_test_%d", path, time.Now().UnixNano())
	//nolint:gosec // Intentional: write-test in user-provided directory.
	f, err := os.Create(testFile)
	if err != nil {
		result.Details["writable"] = strFalse
		result.Status = StatusDegraded
		result.Summary = "Directory exists but is not writable: " + path
		result.SuggestedFix = "Fix write permissions:\n  chmod 755 " + path
		return
	}
	_ = f.Close()
	_ = os.Remove(testFile)
	result.Details["writable"] = strTrue
	result.Status = StatusHealthy
	result.Summary = "Path exists and is writable: " + path
	result.Confidence = ConfidenceNotCause
}

func (r *FilesystemRule) checkFileReadable(result *DiagnosticResult, path string) {
	//nolint:gosec // Intentional: read-test of user-provided path.
	f, err := os.Open(path)
	if err != nil {
		result.Details["readable"] = strFalse
		result.Status = StatusDegraded
		result.Summary = "File exists but is not readable: " + path
		result.SuggestedFix = "Fix read permissions:\n  chmod 644 " + path
		return
	}
	_ = f.Close()
	result.Details["readable"] = strTrue
	result.Status = StatusHealthy
	result.Summary = fmt.Sprintf(
		"File exists and is readable: %s (%s)",
		path,
		result.Details["permissions"],
	)
	result.Confidence = ConfidenceNotCause
}

func (r *FilesystemRule) resolvePath(err error) string {
	return ResolveContextKey(
		err,
		[]string{"path", "file", "dir", "directory", "config_path", "output_path"},
		"",
	)
}
