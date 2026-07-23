package diagnose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

var filesystemSpec = RuleSpec{ //nolint:gochecknoglobals // Immutable rule matching spec.
	ContextKeys: []ContextKey{
		KeyPath,
		KeyFile,
		KeyDir,
		KeyDirectory,
		KeyConfigPath,
		KeyOutputPath,
	},
	// CodeContains matches on error code substrings (different from context keys).
	CodeContains: []string{"file", "dir", "path", "config", "permission"}, //nolint:goconst
}

func (r *FilesystemRule) Run( //nolint:hierarchical-errors // DiagnosticRule interface
	ctx context.Context,
	err error,
) (*DiagnosticResult, error) {
	path := r.resolvePath(err)
	errCtx := ErrorContext(err)
	if path == "" {
		return &DiagnosticResult{
			Status:     StatusUnknown,
			Summary:    "No file path found in error context",
			Confidence: ConfidenceNone,
			Details:    map[string]string{},
			Context:    errCtx,
		}, nil
	}

	result := &DiagnosticResult{
		Details:    map[string]string{"path": path},
		Confidence: ConfidenceHigh,
		Context:    errCtx,
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

func (r *FilesystemRule) handleStatError( //nolint:hierarchical-errors // returns stat errors for diagnosis
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
		SetFix(result, "Fix permissions on "+path, "chmod 755 "+path)
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
		SetFix(result, "Create parent directory: "+parent, "mkdir -p "+parent)
		return
	}
	result.Details["parent_exists"] = strTrue
	result.Details["parent_permissions"] = parentInfo.Mode().Perm().String()
	if filepath.Ext(path) != "" {
		SetFix(result, "Create the file: "+path, "touch "+path)
	} else {
		SetFix(result, "Create directory: "+path, "mkdir -p "+path)
	}
}

func (r *FilesystemRule) checkDirWritable(result *DiagnosticResult, path string) {
	testFile := fmt.Sprintf("%s/.write_test_%d", path, time.Now().UnixNano())
	f, err := os.Create(testFile)
	if err != nil {
		setAccessFailure(
			result,
			"writable",
			"Directory exists but is not writable: "+path,
			"chmod 755 "+path,
		)
		return
	}
	_ = f.Close()           //nolint:hierarchical-errors // cleanup: close error irrelevant
	_ = os.Remove(testFile) //nolint:hierarchical-errors // cleanup: test file removal
	setAccessSuccess(result, "writable", "Path exists and is writable: "+path)
}

func (r *FilesystemRule) checkFileReadable(result *DiagnosticResult, path string) {
	f, err := os.Open(path)
	if err != nil {
		setAccessFailure(
			result,
			"readable",
			"File exists but is not readable: "+path,
			"chmod 644 "+path,
		)
		return
	}
	_ = f.Close() //nolint:hierarchical-errors // cleanup: close error irrelevant
	setAccessSuccess(result, "readable", fmt.Sprintf(
		"File exists and is readable: %s (%s)",
		path,
		result.Details["permissions"],
	))
}

func setAccessFailure(result *DiagnosticResult, key, summary, command string) {
	result.Details[key] = strFalse
	result.Status = StatusDegraded
	result.Summary = summary
	SetFix(result, summary, command)
}

func setAccessSuccess(result *DiagnosticResult, key, summary string) {
	result.Details[key] = strTrue
	result.Status = StatusHealthy
	result.Summary = summary
	result.Confidence = ConfidenceNotCause
}

// SetFix populates the Fix field from a summary and command. Used everywhere a
// rule constructs a Fix literal — collapses 4 lines into 1.
//
// Exposed as public so external submodules (git, postgres) can reuse the
// same one-line idiom instead of repeating `result.Fix = Fix{...}` blocks.
func SetFix(result *DiagnosticResult, summary, command string) {
	result.Fix = Fix{Summary: summary, Command: command}
}

func (r *FilesystemRule) resolvePath(err error) string {
	keys := []string{
		string(KeyPath),
		string(KeyFile),
		string(KeyDir),
		string(KeyDirectory),
		string(KeyConfigPath),
		string(KeyOutputPath),
	}
	return ResolveContextKey(err, keys, "")
}
