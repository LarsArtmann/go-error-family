package diagnose

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// runCommand executes a command with timeout and returns its output.
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

	return stdout, stderr, exitCode, err
}

// commandExists checks if a command is available on the system PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
