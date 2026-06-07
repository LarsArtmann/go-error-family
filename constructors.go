package errorfamily

import (
	"fmt"
	"time"
)

// New creates a new Error with the given family, code, and message.
func New(family Family, code, message string) *Error {
	return &Error{
		code:      code,
		message:   message,
		family:    family,
		cause:     nil,
		context:   make(map[string]string),
		timestamp: time.Now().UTC(),
	}
}

// Newf creates a new Error with a formatted message.
func Newf(family Family, code, format string, args ...any) *Error {
	return New(family, code, fmt.Sprintf(format, args...))
}

// Wrap wraps an existing error with family, code, and message.
// Returns nil if err is nil.
func Wrap(err error, family Family, code, message string) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		code:      code,
		message:   message,
		family:    family,
		cause:     err,
		context:   make(map[string]string),
		timestamp: time.Now().UTC(),
	}
}

// Wrapf wraps an error with a formatted message.
// Returns nil if err is nil.
func Wrapf(err error, family Family, code, format string, args ...any) *Error {
	return Wrap(err, family, code, fmt.Sprintf(format, args...))
}

// Family-specific constructors.
// These make the error's behavioral classification explicit at the call site.

// NewRejection creates a Rejection error (bad input, unauthorized, not found).
func NewRejection(code, message string) *Error {
	return New(Rejection, code, message)
}

// NewConflict creates a Conflict error (version mismatch, duplicate).
func NewConflict(code, message string) *Error {
	return New(Conflict, code, message)
}

// NewTransient creates a Transient error (temporary failure, retryable).
func NewTransient(code, message string) *Error {
	return New(Transient, code, message)
}

// NewCorruption creates a Corruption error (data damaged, not self-healable).
func NewCorruption(code, message string) *Error {
	return New(Corruption, code, message)
}

// NewInfrastructure creates an Infrastructure error (system cannot serve).
func NewInfrastructure(code, message string) *Error {
	return New(Infrastructure, code, message)
}

// Wrap variants for each family.

// WrapRejection wraps an error as Rejection.
func WrapRejection(err error, code, message string) *Error {
	return Wrap(err, Rejection, code, message)
}

// WrapConflict wraps an error as Conflict.
func WrapConflict(err error, code, message string) *Error {
	return Wrap(err, Conflict, code, message)
}

// WrapTransient wraps an error as Transient (retryable).
func WrapTransient(err error, code, message string) *Error {
	return Wrap(err, Transient, code, message)
}

// WrapCorruption wraps an error as Corruption.
func WrapCorruption(err error, code, message string) *Error {
	return Wrap(err, Corruption, code, message)
}

// WrapInfrastructure wraps an error as Infrastructure.
func WrapInfrastructure(err error, code, message string) *Error {
	return Wrap(err, Infrastructure, code, message)
}
