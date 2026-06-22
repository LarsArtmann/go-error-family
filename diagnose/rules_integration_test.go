package diagnose

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

func TestFilesystemRuleName(t *testing.T) {
	r := &FilesystemRule{}
	if r.Name() != "filesystem" {
		t.Errorf("Name() = %q, want %q", r.Name(), "filesystem")
	}
}

func TestFilesystemRuleRunDirWritable(t *testing.T) {
	dir := t.TempDir()
	r := &FilesystemRule{}
	err := errorfamily.NewRejection("config.not_found", "msg").
		WithContext("dir", dir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusHealthy)
	AssertDetail(t, result, "writable", "true")
	AssertDetail(t, result, "type", "directory")
}

func TestFilesystemRuleRunDirNotWritable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write to any directory")
	}
	dir := t.TempDir()
	unwritable := filepath.Join(dir, "locked")
	if mkErr := os.MkdirAll(unwritable, 0o440); mkErr != nil {
		t.Fatalf("mkdir: %v", mkErr)
	}
	//nolint:gosec // G302: test cleanup needs execute bit on directory for removal; 0o750 satisfies both G301 and removal.
	t.Cleanup(func() { _ = os.Chmod(unwritable, 0o750) })

	r := &FilesystemRule{}
	err := errorfamily.NewRejection("config.invalid", "msg").
		WithContext("dir", unwritable)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusDegraded)
	AssertDetail(t, result, "writable", "false")
}

func TestFilesystemRuleRunFileReadable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	if wrErr := os.WriteFile(file, []byte("hello"), 0o600); wrErr != nil {
		t.Fatalf("WriteFile: %v", wrErr)
	}

	r := &FilesystemRule{}
	err := errorfamily.NewRejection("file.not_found", "msg").
		WithContext("path", file)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusHealthy)
	AssertDetail(t, result, "readable", "true")
}

func TestFilesystemRuleRunFileNotReadable(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can read any file")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "secret.txt")
	if wrErr := os.WriteFile(file, []byte("secret"), 0o000); wrErr != nil {
		t.Fatalf("WriteFile: %v", wrErr)
	}
	t.Cleanup(func() { _ = os.Chmod(file, 0o600) })

	r := &FilesystemRule{}
	err := errorfamily.NewRejection("permission.denied", "msg").
		WithContext("file", file)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusDegraded)
	AssertDetail(t, result, "readable", "false")
}

func TestFilesystemRuleRunPermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "noperm.txt")
	if wrErr := os.WriteFile(file, []byte("data"), 0o000); wrErr != nil {
		t.Fatalf("WriteFile: %v", wrErr)
	}
	t.Cleanup(func() { _ = os.Chmod(file, 0o600) })

	r := &FilesystemRule{}
	err := errorfamily.NewRejection("file.error", "msg").
		WithContext("path", file)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusDegraded)
	AssertDetail(t, result, "readable", "false")
}

func TestFilesystemRuleRunCreateFileSuggestion(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "newdir", "newfile.txt")

	r := &FilesystemRule{}
	err := errorfamily.NewRejection("file.not_found", "msg").
		WithContext("path", file)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusFailed)
	AssertDetail(t, result, "parent_exists", "false")
	if result.Fix.Command == "" && result.Fix.Summary == "" {
		t.Error("Fix should not be empty for missing parent")
	}
}

func TestFilesystemRuleRunCreateDirSuggestion(t *testing.T) {
	dir := t.TempDir()
	newDir := filepath.Join(dir, "newdir")

	r := &FilesystemRule{}
	err := errorfamily.NewRejection("dir.not_found", "msg").
		WithContext("dir", newDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusFailed)
	AssertDetail(t, result, "parent_exists", "true")
}

func TestFilesystemRuleRunExistingDirWithParentExists(t *testing.T) {
	dir := t.TempDir()
	existingDir := filepath.Join(dir, "existing")
	if mkErr := os.MkdirAll(existingDir, 0o750); mkErr != nil {
		t.Fatalf("MkdirAll: %v", mkErr)
	}

	r := &FilesystemRule{}
	err := errorfamily.NewRejection("file.not_found", "msg").
		WithContext("path", existingDir)

	result, runErr := r.Run(context.Background(), err)
	if runErr != nil {
		t.Fatalf("Run() error: %v", runErr)
	}
	AssertStatus(t, result, StatusHealthy)
}
