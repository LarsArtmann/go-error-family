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
	return filesystemSpec.matches(err)
}

var filesystemSpec = ruleSpec{
	ContextKeys:  []string{"path", "file", "dir", "directory", "config_path", "output_path"},
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

	// Check 1: Does the path exist?
	info, statErr := os.Stat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			result.Status = StatusFailed
			result.Summary = "Path does not exist: " + path
			result.Details["exists"] = "false"

			// Check parent directory.
			parent := filepath.Dir(path)
			parentInfo, parentErr := os.Stat(parent)
			if parentErr != nil {
				result.Details["parent_exists"] = "false"
				result.SuggestedFix = "Create parent directory and path:\n  mkdir -p " + parent
			} else {
				result.Details["parent_exists"] = "true"
				result.Details["parent_permissions"] = parentInfo.Mode().Perm().String()
				if strings.Contains(path, ".") {
					result.SuggestedFix = "Create the file: " + path
				} else {
					result.SuggestedFix = "Create directory: mkdir -p " + path
				}
			}
			return result, nil
		}

		if os.IsPermission(statErr) {
			result.Status = StatusFailed
			result.Summary = "Permission denied: " + path
			result.Details["exists"] = "true"
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

	// Path exists.
	result.Details["exists"] = "true"
	result.Details["type"] = "file"
	if info.IsDir() {
		result.Details["type"] = "directory"
	}
	result.Details["permissions"] = info.Mode().Perm().String()
	result.Details["size"] = fmt.Sprintf("%d bytes", info.Size())

	// Check 2: Is it writable?
	if info.IsDir() {
		testFile := fmt.Sprintf("%s/.write_test_%d", path, time.Now().UnixNano())
		if f, err := os.Create(testFile); err != nil {
			result.Details["writable"] = "false"
			result.Status = StatusDegraded
			result.Summary = "Directory exists but is not writable: " + path
			result.SuggestedFix = "Fix write permissions:\n  chmod 755 " + path
		} else {
			_ = f.Close()
			_ = os.Remove(testFile)
			result.Details["writable"] = "true"
			result.Status = StatusHealthy
			result.Summary = "Path exists and is writable: " + path
			result.Confidence = ConfidenceNotCause // Path is fine — probably not the root cause
		}
	} else {
		// Check if file is readable.
		if f, err := os.Open(path); err != nil {
			result.Details["readable"] = "false"
			result.Status = StatusDegraded
			result.Summary = "File exists but is not readable: " + path
			result.SuggestedFix = "Fix read permissions:\n  chmod 644 " + path
		} else {
			_ = f.Close()
			result.Details["readable"] = "true"
			result.Status = StatusHealthy
			result.Summary = fmt.Sprintf("File exists and is readable: %s (%s)", path, result.Details["permissions"])
			result.Confidence = ConfidenceNotCause
		}
	}

	return result, nil
}

func (r *FilesystemRule) resolvePath(err error) string {
	return resolveContextKey(err, []string{"path", "file", "dir", "directory", "config_path", "output_path"}, "")
}
