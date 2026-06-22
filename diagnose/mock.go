package diagnose

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MockCommandRunner implements CommandRunner for deterministic testing.
// It records all calls and returns pre-configured responses.
type MockCommandRunner struct {
	mu        sync.Mutex
	Responses map[string]MockResponse
	ExistsMap map[string]bool
	calls     []string
}

// MockResponse is a pre-configured response for a command invocation.
type MockResponse struct {
	Stdout   string
	ExitCode int
	Err      error
}

// NewMockCommandRunner returns a ready-to-use MockCommandRunner.
func NewMockCommandRunner() *MockCommandRunner {
	return &MockCommandRunner{
		Responses: make(map[string]MockResponse),
		ExistsMap: make(map[string]bool),
	}
}

// Set registers a mock response for the given command key.
// Shorthand for assigning a MockResponse{Stdout: stdout, ExitCode: exitCode}
// to m.Responses[key] — collapses a 4-line struct literal into one line.
func (m *MockCommandRunner) Set(key, stdout string, exitCode int) {
	m.Responses[key] = MockResponse{Stdout: stdout, ExitCode: exitCode}
}

// SetError registers a mock response that returns the given error.
func (m *MockCommandRunner) SetError(key string, err error) {
	m.Responses[key] = MockResponse{Err: err}
}

// Run records the call and returns the matching MockResponse.
func (m *MockCommandRunner) Run(
	_ context.Context,
	_ time.Duration,
	name string,
	args ...string,
) (string, int, error) {
	key := name + " " + strings.Join(args, " ")
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, key)
	if resp, ok := m.Responses[key]; ok {
		return resp.Stdout, resp.ExitCode, resp.Err
	}
	for k, resp := range m.Responses {
		if strings.HasPrefix(key, k) {
			return resp.Stdout, resp.ExitCode, resp.Err
		}
	}
	return "", 0, nil
}

// Exists records the call and returns the pre-configured result.
func (m *MockCommandRunner) Exists(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, "exists:"+name)
	return m.ExistsMap[name]
}

// Calls returns a copy of all recorded calls.
func (m *MockCommandRunner) Calls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.calls))
	copy(result, m.calls)
	return result
}
