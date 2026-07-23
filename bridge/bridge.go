// Package bridge connects go-error-family's behavioral classification with
// samber/oops's context enrichment. It provides thin adapters that make
// OopsError satisfy error-family's Classified, Coded, Retryable, and Contextual
// interfaces without either library knowing about the other.
//
// # Architectural rule: libraries classify, applications enrich
//
// go-error-family and samber/oops are complementary, not competing, and this
// bridge is the seam where they meet:
//
//   - LIBRARY code (clients, SDKs, domain packages) imports go-error-family
//     only and returns classified errors. A library knows its own domain
//     contract (a 404 is a Rejection, a timeout is Transient) but must NOT
//     presume the application's observability stack, so it never imports oops.
//   - APPLICATION code (HTTP handlers, CLI mains, jobs) imports oops for
//     enrichment (stack traces, trace IDs, request context) and, if it also
//     needs behavioral decisions, wraps library errors via this bridge.
//
// This keeps the classification protocol (four small interfaces) as the only
// thing libraries force on consumers, while the heavier enrichment choice stays
// opt-in at the application boundary.
//
// Stability: experimental (v0.x). The API may change between minor versions.
//
// # Usage
//
//	rich := oops.In("database").Tags("timeout").With("host", "db1").Wrap(dbErr)
//	classified := bridge.Wrap(rich, errorfamily.Transient)
//
//	errorfamily.IsRetryable(classified) // → true
//	errorfamily.ExitCode(classified)    // → 75
//	classified.OopsError.Stacktrace()   // → full stack trace
//
// For automatic family inference from oops metadata:
//
//	classified := bridge.AutoWrap(rich) // infers Transient from domain "database"
package bridge

import (
	"errors"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/samber/oops"
)

// ClassifiedError wraps an error with a behavioral Family and optional oops context.
// It satisfies error-family's Classified, Coded, Retryable, and Contextual interfaces,
// so Classify() picks it up at step 2 of the classification cascade.
//
// Embeds oops.OopsError to preserve all oops methods (Stacktrace, Trace, Sources, etc.).
// The original error is always preserved in the Unwrap chain, even when it is not
// an OopsError — Wrap never discards the input.
type ClassifiedError struct {
	oops.OopsError

	original error
	family   errorfamily.Family
}

// Wrap attaches a behavioral Family to an error.
// If the error wraps an OopsError, its context and methods are preserved.
// If not, the original error is still accessible via Unwrap() and errors.Is.
//
// The returned value satisfies Classified, Coded, Retryable, and Contextual
// from error-family, and retains all oops methods when the input is an OopsError.
func Wrap(err error, f errorfamily.Family) *ClassifiedError {
	oopsErr, _ := oops.AsOops(err) //nolint:hierarchical-errors // non-OopsError input is expected, produces zero OopsError

	return &ClassifiedError{OopsError: oopsErr, original: err, family: f}
}

// Error implements the error interface.
// Delegates to the embedded OopsError when it has content, otherwise returns
// the original error's message with a family prefix.
func (c *ClassifiedError) Error() string {
	if msg := c.OopsError.Error(); msg != "" {
		return msg
	}

	if c.original != nil {
		return fmt.Sprintf("[%s] %s", c.family, c.original.Error())
	}

	return fmt.Sprintf("[%s]", c.family)
}

// Unwrap returns the underlying error for chain traversal.
// Returns the OopsError's wrapped error when present, otherwise returns
// the original error. This ensures errors.Is always reaches the root cause.
func (c *ClassifiedError) Unwrap() error { //nolint:hierarchical-errors // wrapper interface — signature must return error
	if inner := c.OopsError.Unwrap(); inner != nil {
		return inner
	}

	return c.original
}

// Is supports errors.Is by delegating to OopsError.Is() when the OopsError
// is populated, and falling back to comparing against the original error.
func (c *ClassifiedError) Is(target error) bool {
	if c.OopsError.Error() != "" {
		if c.OopsError.Is(target) {
			return true
		}
	}

	if c.original != nil && errors.Is(c.original, target) {
		return true
	}

	return false
}

// ErrorFamily returns the behavioral classification.
// This is step 2 in error-family's Classify() cascade.
func (c *ClassifiedError) ErrorFamily() errorfamily.Family { return c.family }

// ErrorCode returns a machine-readable error code.
// Bridges oops.Code() to error-family's Coded interface.
// Returns empty string when no code is set.
func (c *ClassifiedError) ErrorCode() string {
	code := c.Code()
	if code == nil {
		return ""
	}

	if s, ok := code.(string); ok {
		return s
	}

	return fmt.Sprint(code)
}

// IsRetryable reports whether the error is worth retrying.
// Derived from the Family: only Transient is retryable.
func (c *ClassifiedError) IsRetryable() bool { return c.family.IsRetryable() }

// ErrorContext bridges oops context to map[string]string for error-family's
// Contextual interface. Includes oops attributes, domain, and tags.
// Non-string values are converted via fmt.Sprint.
func (c *ClassifiedError) ErrorContext() map[string]string {
	raw := c.Context()

	entries := len(raw)
	if domain := c.Domain(); domain != "" {
		entries++
	}

	if tags := c.Tags(); len(tags) > 0 {
		entries++
	}

	if entries == 0 {
		return map[string]string{}
	}

	out := make(map[string]string, entries)

	for k, v := range raw {
		if s, ok := v.(string); ok {
			out[k] = s
		} else {
			out[k] = fmt.Sprint(v)
		}
	}

	if domain := c.Domain(); domain != "" {
		out["domain"] = domain
	}

	if tags := c.Tags(); len(tags) > 0 {
		out["tags"] = strings.Join(tags, ",")
	}

	return out
}

// Family returns the attached Family directly.
// Convenience accessor without interface assertion.
func (c *ClassifiedError) Family() errorfamily.Family { return c.family }

// Format implements fmt.Formatter.
//
//	%v    → OopsError format or [family] original_message
//	%+v   → OopsError verbose (stacktrace) or family + original verbose
//	%s    → message only
func (c *ClassifiedError) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		if c.OopsError.Error() != "" {
			_, _ = fmt.Fprintf(f, "%s", c.OopsError.Error()) //nolint:hierarchical-errors // fmt.Formatter
		} else if c.original != nil && c.original.Error() != "" {
			_, _ = fmt.Fprintf(f, "%s", c.original.Error()) //nolint:hierarchical-errors // fmt.Formatter
		} else {
			_, _ = fmt.Fprintf(f, "[%s]", c.family) //nolint:hierarchical-errors // fmt.Formatter
		}
	case 'v':
		if f.Flag('+') {
			if c.OopsError.Error() != "" {
				_, _ = fmt.Fprintf(f, "%+v", &c.OopsError) //nolint:hierarchical-errors // fmt.Formatter
			} else if c.original != nil {
				_, _ = fmt.Fprintf(f, "[%s] %+v", c.family, c.original) //nolint:hierarchical-errors // fmt.Formatter
			} else {
				_, _ = fmt.Fprintf(f, "[%s]", c.family) //nolint:hierarchical-errors // fmt.Formatter
			}

			return
		}

		_, _ = fmt.Fprint(f, c.Error()) //nolint:hierarchical-errors // fmt.Formatter
	default:
		_, _ = fmt.Fprint(f, c.Error()) //nolint:hierarchical-errors // fmt.Formatter
	}
}
