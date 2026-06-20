package errorfamily

import (
	"encoding/json"
	"fmt"
	"maps"
	"time"
)

// Error is the reference implementation of a classified, structured error.
//
// This is a convenience type, not the contract. The public protocol is the
// Coded/Classified/Contextual/Retryable interfaces — your own domain error
// types implement only those and need not embed or use this struct.
//
// Projects with simple error needs can use this directly.
// Projects with domain-specific needs (e.g., FindingError with Position/File)
// can build their own struct and implement the Coded/Classified/Contextual interfaces.
//
// Implements: error, Coded, Classified, Contextual, Retryable, fmt.Formatter.
type Error struct {
	code      string            // machine-readable identity (e.g. "db.timeout")
	message   string            // human-readable technical message
	family    Family            // behavioral classification
	context   map[string]string // factual details about the error
	cause     error             // underlying error in the chain
	timestamp time.Time         // when the error occurred
}

// Error implements the error interface.
// Returns a compact format: [family:code] message
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.family, e.code, e.message, e.cause)
	}
	return fmt.Sprintf("[%s:%s] %s", e.family, e.code, e.message)
}

// Unwrap returns the underlying cause for error chain traversal.
func (e *Error) Unwrap() error { return e.cause }

// Is supports errors.Is by matching error code and family.
// Two errors match if they have the same code AND family.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.code == t.code && e.family == t.family
}

// ErrorCode returns the machine-readable error code.
func (e *Error) ErrorCode() string { return e.code }

// ErrorFamily returns the behavioral classification.
func (e *Error) ErrorFamily() Family { return e.family }

// ErrorContext returns a copy of the error's factual context.
func (e *Error) ErrorContext() map[string]string {
	if e.context == nil {
		return map[string]string{}
	}
	return maps.Clone(e.context)
}

// IsRetryable reports whether this error is worth retrying.
func (e *Error) IsRetryable() bool { return e.family.IsRetryable() }

// Timestamp returns when this error was created.
func (e *Error) Timestamp() time.Time { return e.timestamp }

// Family returns the error's behavioral classification.
// Convenience accessor for direct use without interface assertion.
func (e *Error) Family() Family { return e.family }

// Code returns the machine-readable error code.
// Convenience accessor for direct use without interface assertion.
func (e *Error) Code() string { return e.code }

// Message returns the human-readable technical message.
func (e *Error) Message() string { return e.message }

// Cause returns the underlying error in the chain.
func (e *Error) Cause() error { return e.cause }

// Format implements fmt.Formatter.
//
//	%v    → [family:code] message
//	%+v   → verbose multi-line with context and cause chain
//	%s    → message only (no classification prefix)
func (e *Error) Format(f fmt.State, verb rune) {
	switch verb {
	case 's':
		_, _ = fmt.Fprint(f, e.message)
	case 'v':
		if f.Flag('+') {
			e.formatVerbose(f)
			return
		}
		_, _ = fmt.Fprint(f, e.Error())
	default:
		_, _ = fmt.Fprint(f, e.Error())
	}
}

func (e *Error) formatVerbose(f fmt.State) {
	_, _ = fmt.Fprintf(f, "[%s] %s: %s", e.family, e.code, e.message)

	if len(e.context) > 0 {
		_, _ = fmt.Fprint(f, "\n  context:")
		for k, v := range e.context {
			_, _ = fmt.Fprintf(f, "\n    %s: %s", k, v)
		}
	}

	if !e.timestamp.IsZero() {
		_, _ = fmt.Fprintf(f, "\n  at: %s", e.timestamp.Format(time.RFC3339))
	}

	if e.cause != nil {
		_, _ = fmt.Fprintf(f, "\n  caused by: %+v", e.cause)
	}
}

// WithContext adds a key-value pair to the error's context.
// Returns a new Error, leaving the original unchanged — safe for shared/sentinel errors.
func (e *Error) WithContext(key, value string) *Error {
	clone := e.clone()
	clone.context[key] = value
	return clone
}

// WithContextMap merges a map of key-value pairs into the error's context.
// Returns a new Error, leaving the original unchanged. Nil or empty input
// returns a clone with no added context.
func (e *Error) WithContextMap(ctx map[string]string) *Error {
	clone := e.clone()
	maps.Insert(clone.context, maps.All(ctx))
	return clone
}

// WithContextf adds a formatted key-value pair to the error's context.
// The value is produced by fmt.Sprintf(format, args...).
// Returns a new Error, leaving the original unchanged.
func (e *Error) WithContextf(key, format string, args ...any) *Error {
	clone := e.clone()
	clone.context[key] = fmt.Sprintf(format, args...)
	return clone
}

// WithCause sets the underlying cause and returns a new error for chaining.
func (e *Error) WithCause(cause error) *Error {
	clone := e.clone()
	clone.cause = cause
	return clone
}

// WithTimestamp sets the error timestamp and returns a new error for chaining.
// Useful for testing and deterministic construction.
func (e *Error) WithTimestamp(ts time.Time) *Error {
	clone := e.clone()
	clone.timestamp = ts
	return clone
}

// clone returns a shallow copy of the error with a deep-copied context map.
func (e *Error) clone() *Error {
	c := &Error{
		code:      e.code,
		message:   e.message,
		family:    e.family,
		cause:     e.cause,
		timestamp: e.timestamp,
		context:   make(map[string]string, len(e.context)),
	}
	maps.Copy(c.context, e.context)
	return c
}

// Summary returns a one-line human-readable summary suitable for logs and CLI output.
// Format: "code: message" without family prefix.
func (e *Error) Summary() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s — %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.code, e.message)
}

// HasContext reports whether the error has a specific context key.
func (e *Error) HasContext(key string) bool {
	if e.context == nil {
		return false
	}
	_, ok := e.context[key]
	return ok
}

// ContextValue returns a specific context value, or empty string if not present.
func (e *Error) ContextValue(key string) string {
	if e.context == nil {
		return ""
	}
	return e.context[key]
}

// jsonError is the JSON view of an Error for API responses.
type jsonError struct {
	Family    string            `json:"family"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Context   map[string]string `json:"context,omitempty"`
	Retryable bool              `json:"retryable"`
	Timestamp string            `json:"timestamp,omitempty"`
}

// JSON returns a canonical JSON encoding of the error for API boundaries.
// The shape is stable: {family, code, message, context, retryable, timestamp}.
// Use this for HTTP/REST error responses where a structured body is preferable
// to the [transient:code] message format of Error().
func (e *Error) JSON() ([]byte, error) {
	view := jsonError{
		Family:    e.family.String(),
		Code:      e.code,
		Message:   e.message,
		Context:   e.context,
		Retryable: e.IsRetryable(),
	}
	if !e.timestamp.IsZero() {
		view.Timestamp = e.timestamp.Format(time.RFC3339)
	}
	data, err := json.Marshal(view)
	if err != nil {
		return nil, fmt.Errorf("marshal error view: %w", err)
	}
	return data, nil
}
