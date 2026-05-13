package diagnose

import (
	"context"
	"fmt"
	"os"
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
	return hasContextKey(err, "path", "file", "dir", "directory", "config_path", "output_path") ||
		errorCodeContains(err, "file") ||
		errorCodeContains(err, "dir") ||
		errorCodeContains(err, "path") ||
		errorCodeContains(err, "config") ||
		errorCodeContains(err, "permission")
}

func (r *FilesystemRule) Run(ctx context.Context, err error) (*DiagnosticResult, error) {
	path := r.resolvePath(err)
	if path == "" {
		return &DiagnosticResult{
			Status:     StatusUnknown,
			Summary:    "No file path found in error context",
			Confidence: 0.1,
			Details:    map[string]string{},
		}, nil
	}

	result := &DiagnosticResult{
		Details:    map[string]string{"path": path},
		Confidence: 0.8,
	}

	// Check 1: Does the path exist?
	info, statErr := os.Stat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			result.Status = StatusFailed
			result.Summary = fmt.Sprintf("Path does not exist: %s", path)
			result.Details["exists"] = "false"

			// Check parent directory.
			parent := parentDir(path)
			parentInfo, parentErr := os.Stat(parent)
			if parentErr != nil {
				result.Details["parent_exists"] = "false"
				result.SuggestedFix = fmt.Sprintf("Create parent directory and path:\n  mkdir -p %s", parent)
				result.AutoFixable = true
				result.AutoFix = func(ctx context.Context) (*FixResult, error) {
					if mkErr := os.MkdirAll(parent, 0o755); mkErr != nil {
						return nil, mkErr
					}
					return &FixResult{
						Resolved: true,
						Summary:  fmt.Sprintf("Created directory: %s", parent),
						Actions:  []string{fmt.Sprintf("mkdir -p %s", parent)},
					}, nil
				}
			} else {
				result.Details["parent_exists"] = "true"
				result.Details["parent_permissions"] = parentInfo.Mode().Perm().String()
				if strings.Contains(path, ".") {
					result.SuggestedFix = fmt.Sprintf("Create the file: %s", path)
				} else {
					result.SuggestedFix = fmt.Sprintf("Create directory: mkdir -p %s", path)
					result.AutoFixable = true
					result.AutoFix = func(ctx context.Context) (*FixResult, error) {
						if mkErr := os.MkdirAll(path, 0o755); mkErr != nil {
							return nil, mkErr
						}
						return &FixResult{
							Resolved: true,
							Summary:  fmt.Sprintf("Created directory: %s", path),
							Actions:  []string{fmt.Sprintf("mkdir -p %s", path)},
						}, nil
					}
				}
			}
			return result, nil
		}

		if os.IsPermission(statErr) {
			result.Status = StatusFailed
			result.Summary = fmt.Sprintf("Permission denied: %s", path)
			result.Details["exists"] = "true"
			result.Details["permissions"] = "denied"
			result.SuggestedFix = fmt.Sprintf("Fix permissions:\n  chmod 755 %s\nOr run with appropriate privileges.", path)
			result.AutoFixable = false
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
			result.Summary = fmt.Sprintf("Directory exists but is not writable: %s", path)
			result.SuggestedFix = fmt.Sprintf("Fix write permissions:\n  chmod 755 %s", path)
		} else {
			_ = f.Close()
			_ = os.Remove(testFile)
			result.Details["writable"] = "true"
			result.Status = StatusHealthy
			result.Summary = fmt.Sprintf("Path exists and is writable: %s", path)
			result.Confidence = 0.3 // Path is fine — probably not the root cause
		}
	} else {
		// Check if file is readable.
		if f, err := os.Open(path); err != nil {
			result.Details["readable"] = "false"
			result.Status = StatusDegraded
			result.Summary = fmt.Sprintf("File exists but is not readable: %s", path)
			result.SuggestedFix = fmt.Sprintf("Fix read permissions:\n  chmod 644 %s", path)
		} else {
			_ = f.Close()
			result.Details["readable"] = "true"
			result.Status = StatusHealthy
			result.Summary = fmt.Sprintf("File exists and is readable: %s (%s)", path, result.Details["permissions"])
			result.Confidence = 0.3
		}
	}

	return result, nil
}

func (r *FilesystemRule) resolvePath(err error) string {
	for _, key := range []string{"path", "file", "dir", "directory", "config_path", "output_path"} {
		if v := contextValue(err, key); v != "" {
			return v
		}
	}
	return ""
}

func parentDir(path string) string {
	if strings.Contains(path, "/") {
		parent := path[:strings.LastIndex(path, "/")]
		if parent == "" {
			return "/"
		}
		return parent
	}
	return "."
}
