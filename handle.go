package errorfamily

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// HandleResult contains the full output of handling an error at the CLI boundary.
type HandleResult struct {
	ExitCode     int
	Message      string
	SuggestedFix string
}

// HandleConfig controls how HandleError processes an error at the CLI boundary.
type HandleConfig struct {
	// Output is where human-readable messages are written. Defaults to os.Stderr.
	Output io.Writer

	// Diagnose controls whether automatic diagnostic rules are run.
	Diagnose bool

	// TemplateOverride overrides the default message template for a specific error code.
	// map[errorCode]MessageTemplate
	TemplateOverride map[string]MessageTemplate

	// DiagnosticFunc runs diagnostics for the error. If nil, no diagnostics run.
	DiagnosticFunc DiagnosticFunc

	// OnDiagnosed is called after diagnostics complete, before exit.
	// Receives the error and diagnostic findings. Useful for logging/metrics.
	OnDiagnosed func(err error, findings []DiagnosticFinding)
}

// DiagnosticFunc runs diagnostics for an error and returns results.
// The diagnose.Runner satisfies this via its Run method.
type DiagnosticFunc func(ctx context.Context, err error) []DiagnosticFinding

// DiagnosticFinding is a minimal result type for the CLI boundary.
// Avoids importing diagnose while preserving type safety.
type DiagnosticFinding struct {
	RuleName     string
	Status       string // "healthy", "degraded", "failed", "unknown"
	Summary      string
	SuggestedFix string
	Confidence   float64
}

// MessageTemplate defines the Wix-style presentation for an error code.
// Based on the Wix UX framework: What / Why / Fix / WayOut.
type MessageTemplate struct {
	What   string // "Could not find {{.path}}"
	Why    string // "The file doesn't exist at the expected location."
	Fix    string // "Check that {{.path}} exists and is readable."
	WayOut string // "Run with --verbose for more details."
}

// HandleError is the CLI boundary handler — the meta service.
//
// Call this exactly once at the top of main():
//
//	func main() {
//	    if err := run(); err != nil {
//	        os.Exit(errorfamily.HandleError(err))
//	    }
//	}
//
// It:
//  1. Classifies the error (Family → exit code)
//  2. Extracts Code and Context for specific messaging
//  3. Optionally runs diagnostic rules
//  4. Formats a Wix-quality message for the user
//  5. Writes to stderr and returns the exit code
func HandleError(err error) int {
	return HandleErrorWithConfig(err, HandleConfig{})
}

// HandleErrorWithConfig is the configurable version of HandleError.
func HandleErrorWithConfig(err error, cfg HandleConfig) int {
	if err == nil {
		return 0
	}

	if cfg.Output == nil {
		cfg.Output = os.Stderr
	}

	family := Classify(err)
	exitCode := family.ExitCode()

	code := extractCode(err)
	ctx := extractContext(err)

	if cfg.Diagnose && cfg.DiagnosticFunc != nil {
		findings := cfg.DiagnosticFunc(context.Background(), err)
		if cfg.OnDiagnosed != nil {
			cfg.OnDiagnosed(err, findings)
		}
	}

	message := renderCLI(code, ctx, family, cfg)

	_, _ = fmt.Fprintln(cfg.Output, message)

	return exitCode
}

// HandleErrorDetailed returns a structured result without writing output.
// Useful for HTTP handlers, gRPC interceptors, and programmatic consumers.
func HandleErrorDetailed(err error) *HandleResult {
	if err == nil {
		return &HandleResult{ExitCode: 0}
	}

	family := Classify(err)
	code := extractCode(err)
	context := extractContext(err)

	result := &HandleResult{
		ExitCode:     family.ExitCode(),
		Message:      renderMessage(code, context, family),
	}

	if !IsRetryable(err) {
		if tmpl, ok := lookupDefault(code); ok && tmpl.Fix != "" {
			result.SuggestedFix = applyContext(tmpl.Fix, context)
		} else {
			result.SuggestedFix = family.DefaultFix()
		}
	}

	return result
}

func extractCode(err error) string {
	if coded, ok := errors.AsType[Coded](err); ok {
		return coded.ErrorCode()
	}
	return ""
}

func extractContext(err error) map[string]string {
	if contextual, ok := errors.AsType[Contextual](err); ok {
		return contextual.ErrorContext()
	}
	return map[string]string{}
}

func renderCLI(code string, context map[string]string, family Family, cfg HandleConfig) string {
	// 1. Consumer override (per-call).
	if cfg.TemplateOverride != nil {
		if tmpl, ok := cfg.TemplateOverride[code]; ok {
			return applyTemplate(tmpl, context, family)
		}
	}

	// 2. Registered template (global).
	if tmpl, ok := lookupTemplate(code); ok {
		return applyTemplate(tmpl, context, family)
	}

	// 3. Default rendering (exact code match → family fallback).
	return renderMessage(code, context, family)
}

func renderMessage(code string, context map[string]string, family Family) string {
	if code == "" {
		return family.DefaultMessage()
	}

	// Code-specific template from defaults.
	if tmpl, ok := lookupDefault(code); ok {
		return applyTemplate(tmpl, context, family)
	}

	// Family fallback with code as header.
	var parts []string
	parts = append(parts, fmt.Sprintf("Error: %s", code))
	if why := family.DefaultWhy(); why != "" {
		parts = append(parts, why)
	}
	if fix := family.DefaultFix(); fix != "" {
		parts = append(parts, fix)
	}
	return strings.Join(parts, "\n")
}

func applyTemplate(tmpl MessageTemplate, context map[string]string, family Family) string {
	var parts []string
	if tmpl.What != "" {
		parts = append(parts, applyContext(tmpl.What, context))
	}
	if why := tmpl.Why; why != "" {
		parts = append(parts, applyContext(why, context))
	} else if why := family.DefaultWhy(); why != "" {
		parts = append(parts, why)
	}
	if tmpl.Fix != "" {
		parts = append(parts, applyContext(tmpl.Fix, context))
	}
	if tmpl.WayOut != "" {
		parts = append(parts, applyContext(tmpl.WayOut, context))
	}
	return strings.Join(parts, "\n")
}

func applyContext(template string, context map[string]string) string {
	s := template
	for k, v := range context {
		s = strings.ReplaceAll(s, "{{."+k+"}}", v)
	}
	return s
}

func lookupDefault(code string) (MessageTemplate, bool) {
	tmpl, ok := defaultMessages[strings.ToLower(code)]
	return tmpl, ok
}

// defaultMessages maps error codes (lowercase) to human-readable messages.
// Codes are matched exactly — no substring matching.
var defaultMessages = map[string]MessageTemplate{
	"file.not_found":     {What: "A required resource was not found.", Fix: "Check that the path and resource name are correct."},
	"permission.denied":  {What: "Permission was denied.", Fix: "Check file permissions or run with appropriate privileges."},
	"db.timeout":         {What: "The database operation timed out.", Fix: "Increase the timeout or check system resources."},
	"db.connection":      {What: "Could not establish a database connection.", Fix: "Check that the database is running and reachable."},
	"db.error":           {What: "A database operation failed.", Fix: "Check the database logs for details."},
	"config.invalid":     {What: "There is a configuration issue.", Fix: "Review your configuration file for errors."},
	"config.not_found":   {What: "A configuration file was not found.", Fix: "Check that the config file path is correct."},
	"conflict":           {What: "A conflict was detected.", Fix: "Refresh your data and try the operation again."},
	"validation":         {What: "Validation failed.", Fix: "Check your input and try again."},
	"timeout":            {What: "The operation timed out.", Fix: "Increase the timeout or check system resources."},
	"connection.refused": {What: "Could not establish a connection.", Fix: "Check that the target service is running."},
	"git.error":          {What: "A git operation failed.", Fix: "Check the git repository state."},
}

// RegisterTemplate adds a MessageTemplate for a specific error code.
// Thread-safe. Overrides any existing template for the same code.
func RegisterTemplate(code string, tmpl MessageTemplate) {
	templateRegistry.mu.Lock()
	defer templateRegistry.mu.Unlock()
	templateRegistry.entries[strings.ToLower(code)] = tmpl
}

var templateRegistry = struct {
	mu      sync.RWMutex
	entries map[string]MessageTemplate
}{
	entries: make(map[string]MessageTemplate),
}

func lookupTemplate(code string) (MessageTemplate, bool) {
	templateRegistry.mu.RLock()
	defer templateRegistry.mu.RUnlock()
	tmpl, ok := templateRegistry.entries[strings.ToLower(code)]
	return tmpl, ok
}
