// Package bridge connects go-error-family's behavioral classification with
// samber/oops's context enrichment. It provides thin adapters that make
// OopsError satisfy error-family's Classified, Coded, Retryable, and Contextual
// interfaces without either library knowing about the other.
//
// Usage:
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

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/samber/oops"
)

// ClassifiedOops wraps an error with a behavioral Family and optional oops context.
// It satisfies error-family's Classified, Coded, Retryable, and Contextual interfaces,
// so Classify() picks it up at step 2 of the classification cascade.
//
// Embeds oops.OopsError to preserve all oops methods (Stacktrace, Trace, Sources, etc.).
// The original error is always preserved in the Unwrap chain, even when it is not
// an OopsError — Wrap never discards the input.
type ClassifiedOops struct {
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
func Wrap(err error, f errorfamily.Family) *ClassifiedOops {
	oopsErr, _ := oops.AsOops(err)
	return &ClassifiedOops{OopsError: oopsErr, original: err, family: f}
}

// Error implements the error interface.
// Delegates to the embedded OopsError when it has content, otherwise returns
// the original error's message with a family prefix.
func (c *ClassifiedOops) Error() string {
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
func (c *ClassifiedOops) Unwrap() error {
	if inner := c.OopsError.Unwrap(); inner != nil {
		return inner
	}
	return c.original
}

// Is supports errors.Is by delegating to OopsError.Is() when the OopsError
// is populated, and falling back to comparing against the original error.
func (c *ClassifiedOops) Is(target error) bool {
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
func (c *ClassifiedOops) ErrorFamily() errorfamily.Family { return c.family }

// ErrorCode returns a machine-readable error code.
// Bridges oops.Code() to error-family's Coded interface.
// Returns empty string when no code is set.
func (c *ClassifiedOops) ErrorCode() string {
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
func (c *ClassifiedOops) IsRetryable() bool { return c.family.IsRetryable() }

// ErrorContext bridges oops context to map[string]string for error-family's
// Contextual interface. Includes oops attributes, domain, and tags.
// Non-string values are converted via fmt.Sprint.
func (c *ClassifiedOops) ErrorContext() map[string]string {
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
		out["tags"] = fmt.Sprint(tags)
	}
	return out
}

// Family returns the attached Family directly.
// Convenience accessor without interface assertion.
func (c *ClassifiedOops) Family() errorfamily.Family { return c.family }

// Format implements fmt.Formatter.
//
//	%v    → OopsError format or [family] original_message
//	%+v   → OopsError verbose (stacktrace) or family + original verbose
//	%s    → message only
func (c *ClassifiedOops) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		if c.OopsError.Error() != "" {
			_, _ = fmt.Fprintf(f, "%s", c.OopsError.Error())
		} else if c.original != nil {
			_, _ = fmt.Fprintf(f, "%s", c.original.Error())
		}
	case 'v':
		if f.Flag('+') {
			if c.OopsError.Error() != "" {
				_, _ = fmt.Fprintf(f, "%+v", &c.OopsError)
			} else if c.original != nil {
				_, _ = fmt.Fprintf(f, "[%s] %+v", c.family, c.original)
			} else {
				_, _ = fmt.Fprintf(f, "[%s]", c.family)
			}
			return
		}
		_, _ = fmt.Fprint(f, c.Error())
	default:
		_, _ = fmt.Fprint(f, c.Error())
	}
}
