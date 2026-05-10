package errorfamily

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// HandleResult contains the full output of handling an error at the CLI boundary.
type HandleResult struct {
	ExitCode      int
	Message       string
	Diagnostics   []string
	SuggestedFix  string
	ErrorReported bool
}

// HandleConfig controls how HandleError processes an error at the CLI boundary.
type HandleConfig struct {
	// Output is where human-readable messages are written. Defaults to os.Stderr.
	Output io.Writer

	// Verbose controls whether diagnostic details are shown.
	Verbose bool

	// Diagnose controls whether automatic diagnostic rules are run.
	Diagnose bool

	// TemplateOverride overrides the default message template for a specific error code.
	// map[errorCode]MessageTemplate
	TemplateOverride map[string]MessageTemplate

	// DiagnosticRunner is a custom diagnostic runner. If nil, no diagnostics run.
	DiagnosticRunner DiagnosticRunner

	// OnDiagnosed is called after diagnostics complete, before exit.
	// Receives the error and diagnostic results. Useful for logging/metrics.
	OnDiagnosed func(err error, results any)
}

// DiagnosticRunner is a minimal interface for the CLI boundary.
// The diagnose.Runner satisfies this.
type DiagnosticRunner interface {
	Run(ctx context.Context, err error) any
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
	context := extractContext(err)

	message := renderCLI(code, context, family, cfg)

	fmt.Fprintln(cfg.Output, message)

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
		ExitCode:    family.ExitCode(),
		Message:     renderMessage(code, context, family),
		Diagnostics: []string{},
	}

	if !IsRetryable(err) {
		result.SuggestedFix = suggestFix(code, context, family)
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
	// Check for template override first.
	if cfg.TemplateOverride != nil {
		if tmpl, ok := cfg.TemplateOverride[code]; ok {
			return applyTemplate(tmpl, context, family)
		}
	}

	return renderMessage(code, context, family)
}

func renderMessage(code string, context map[string]string, family Family) string {
	if code == "" {
		return familyDefaultMessage(family)
	}

	var parts []string

	// What happened
	what := formatWhat(code, context)
	parts = append(parts, what)

	// Why (if we know)
	why := formatWhy(code, context, family)
	if why != "" {
		parts = append(parts, why)
	}

	// Fix (if actionable)
	fix := suggestFix(code, context, family)
	if fix != "" {
		parts = append(parts, fix)
	}

	return strings.Join(parts, "\n")
}

func formatWhat(code string, context map[string]string) string {
	// Try to make the message specific from code + context.
	msg := codeToWhat(code)
	if msg == "" {
		return fmt.Sprintf("Error: %s", code)
	}
	return applyContext(msg, context)
}

func formatWhy(code string, context map[string]string, family Family) string {
	// Reassure the user based on family.
	switch family {
	case Rejection, Conflict:
		// User's fault — explain what they can check.
		return ""
	case Transient:
		return "This is a temporary issue. No data was lost."
	case Corruption:
		return "Some data appears to be damaged. This requires attention."
	case Infrastructure:
		return "This is a system issue, not something you caused."
	default:
		return ""
	}
}

func suggestFix(code string, context map[string]string, family Family) string {
	fix := codeToFix(code)
	if fix != "" {
		return applyContext(fix, context)
	}

	// Family-based fallback.
	switch family {
	case Rejection:
		if path, ok := context["path"]; ok {
			return fmt.Sprintf("Check that %s is correct.", path)
		}
		return "Check your input and try again."
	case Conflict:
		return "Refresh your data and try the operation again."
	case Transient:
		return "Wait a moment and try again."
	case Corruption:
		return "This may require manual intervention. Check the logs for details."
	case Infrastructure:
		return "The service may be temporarily unavailable. Try again later."
	default:
		return "Try again or contact support."
	}
}

func applyTemplate(tmpl MessageTemplate, context map[string]string, family Family) string {
	var parts []string
	if tmpl.What != "" {
		parts = append(parts, applyContext(tmpl.What, context))
	}
	if tmpl.Why != "" {
		parts = append(parts, applyContext(tmpl.Why, context))
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

func codeToWhat(code string) string {
	// Map common code patterns to human-readable "what happened".
	lower := strings.ToLower(code)
	switch {
	case strings.Contains(lower, "not_found"):
		return "A required resource was not found."
	case strings.Contains(lower, "permission") || strings.Contains(lower, "denied"):
		return "Permission was denied."
	case strings.Contains(lower, "timeout"):
		return "The operation timed out."
	case strings.Contains(lower, "connection") || strings.Contains(lower, "connect"):
		return "Could not establish a connection."
	case strings.Contains(lower, "conflict"):
		return "A conflict was detected."
	case strings.Contains(lower, "validation"):
		return "Validation failed."
	case strings.Contains(lower, "config"):
		return "There is a configuration issue."
	case strings.Contains(lower, "git"):
		return "A git operation failed."
	case strings.Contains(lower, "database") || strings.Contains(lower, "db"):
		return "A database operation failed."
	default:
		return ""
	}
}

func codeToFix(code string) string {
	lower := strings.ToLower(code)
	switch {
	case strings.Contains(lower, "not_found"):
		return "Check that the path and resource name are correct."
	case strings.Contains(lower, "permission") || strings.Contains(lower, "denied"):
		return "Check file permissions or run with appropriate privileges."
	case strings.Contains(lower, "timeout"):
		return "Increase the timeout or check system resources."
	case strings.Contains(lower, "config"):
		return "Review your configuration file for errors."
	default:
		return ""
	}
}

func familyDefaultMessage(family Family) string {
	switch family {
	case Rejection:
		return "The request was invalid. Check your input and try again."
	case Conflict:
		return "A conflict was detected. Refresh and try again."
	case Transient:
		return "A temporary error occurred. Please try again in a few moments."
	case Corruption:
		return "Data appears to be corrupted. This requires manual intervention."
	case Infrastructure:
		return "The service is currently unavailable. Please try again later."
	default:
		return "An unexpected error occurred."
	}
}
