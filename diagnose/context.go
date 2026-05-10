package diagnose

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// SystemSnapshot captures the system state at the time of an error.
// Useful for auto-context discovery — the error may not know the full
// system picture, but the snapshot provides it.
type SystemSnapshot struct {
	Timestamp   time.Time         `json:"timestamp"`
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	Hostname    string            `json:"hostname"`
	GoVersion   string            `json:"go_version"`
	PID         int               `json:"pid"`
	WorkingDir  string            `json:"working_dir"`
	DiskFree    map[string]int64  `json:"disk_free_bytes"`
	Environment map[string]string `json:"environment"`
	Uptime      time.Duration     `json:"uptime"`
}

// GatherSystemSnapshot captures the current system state.
// Environment variables are sanitized to remove secrets (keys containing
// password, secret, token, key, credential, auth).
func GatherSystemSnapshot(ctx context.Context) *SystemSnapshot {
	snapshot := &SystemSnapshot{
		Timestamp:   time.Now().UTC(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		GoVersion:   runtime.Version(),
		PID:         os.Getpid(),
		WorkingDir:  mustGetwd(),
		DiskFree:    map[string]int64{},
		Environment: map[string]string{},
	}

	if hostname, err := os.Hostname(); err == nil {
		snapshot.Hostname = hostname
	}

	// Sanitized environment.
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		if isSecretKey(key) {
			snapshot.Environment[key] = "***REDACTED***"
		} else {
			snapshot.Environment[key] = parts[1]
		}
	}

	return snapshot
}

var secretPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|key|credential|auth|api_key|apikey|private)`)

func isSecretKey(key string) bool {
	return secretPattern.MatchString(key)
}

func mustGetwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "<unknown>"
	}
	return dir
}

// runCommand executes a command with timeout and returns its output.
// This is the safe command runner used by diagnostic rules.
func runCommand(ctx context.Context, timeout time.Duration, name string, args ...string) (stdout, stderr string, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	stdout = strings.TrimSpace(outBuf.String())
	stderr = strings.TrimSpace(errBuf.String())

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			err = nil
		}
	}

	return
}

// commandExists checks if a command is available on the system PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// formatCommandFix returns a platform-appropriate command string.
func formatCommandFix(command string) string {
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("Run: %s", command)
	}
	return fmt.Sprintf("Run: %s", command)
}
