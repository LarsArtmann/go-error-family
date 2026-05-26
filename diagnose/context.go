package diagnose

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// RunCommand executes a command with timeout and returns its output.
func RunCommand(
	ctx context.Context,
	timeout time.Duration,
	name string,
	args ...string,
) (stdout string, exitCode int, err error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	//nolint:gosec // Intentional: diagnostic rules run user-provided commands from error context.
	cmd := exec.CommandContext(ctx, name, args...)
	var outBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = io.Discard

	err = cmd.Run()
	stdout = strings.TrimSpace(outBuf.String())

	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
			err = nil
		} else {
			exitCode = -1
			err = fmt.Errorf("timeout=%v: %w", timeout, err)
		}
	}

	return stdout, exitCode, err
}

// CommandExists checks if a command is available on the system PATH.
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
