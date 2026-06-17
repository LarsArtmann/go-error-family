package errorfamily

import (
	"errors"
	"maps"
	"strings"
	"sync"
)

// Registry holds classification sentinel mappings and message templates.
// It is safe for concurrent use. The zero value is not usable — use [NewRegistry].
//
// A Registry solves two problems that package-level globals cannot:
//   - Test isolation: each test can construct its own Registry without polluting
//     global state or needing t.Cleanup(Unregister…) boilerplate.
//   - Scoped overrides: a service can register sentinels or templates for its
//     own error handling without affecting other parts of the same binary.
//
// The package-level [DefaultRegistry] is used by the convenience functions
// ([Classify], [RegisterClassification], [RegisterTemplate], etc.) and by the
// HandleError family when no custom Registry is provided in [HandleConfig].
type Registry struct {
	mu        sync.RWMutex
	sentinels map[error]Family
	templates map[string]MessageTemplate
}

// NewRegistry creates an empty Registry ready for use.
func NewRegistry() *Registry {
	return &Registry{
		sentinels: make(map[error]Family),
		templates: make(map[string]MessageTemplate),
	}
}

// DefaultRegistry is the package-level Registry used by the convenience
// functions (Classify, RegisterClassification, RegisterTemplate, etc.) and by
// HandleError when HandleConfig.Registry is nil.
//
//nolint:gochecknoglobals // Package-level default registry for backward-compatible API.
var DefaultRegistry = NewRegistry()

// Classify returns the Family of any error using this registry's sentinel mappings.
//
// Checks in order, first match wins:
//  1. Multi-error (errors.Join) — first non-Transient sub-error wins
//  2. Classified interface — the error itself declares its family
//  3. Retryable interface — infer from retryability
//  4. Registered sentinels in this registry
//  5. Default — Transient (fail-open for retry)
//
// Returns Rejection for nil errors.
func (r *Registry) Classify(err error) Family {
	if err == nil {
		return Rejection
	}

	// 1. Multi-error support (errors.Join).
	// errors.AsType traverses multi-errors and returns the first match,
	// but we want fail-closed behavior: if any sub-error is not retryable,
	// the whole operation should not be retried.
	if u, ok := err.(interface{ Unwrap() []error }); ok {
		for _, sub := range u.Unwrap() {
			family := r.Classify(sub)
			if family != Transient {
				return family
			}
		}
		return Transient
	}

	// 2. Check for explicit classification.
	if c, ok := errors.AsType[Classified](err); ok {
		return c.ErrorFamily()
	}

	// 3. Check for retryability (infer family).
	if rl, ok := errors.AsType[Retryable](err); ok {
		if rl.IsRetryable() {
			return Transient
		}
		return Rejection
	}

	// 4. Check registered third-party sentinels.
	if family, ok := r.lookupSentinel(err); ok {
		return family
	}

	// 5. Default: Transient (fail-open so unknown errors get retried).
	return Transient
}

// RegisterClassification maps a third-party sentinel error to a Family.
// Thread-safe. Call from init() in external packages:
//
//	func init() {
//	    errorfamily.DefaultRegistry.RegisterClassification(sql.ErrConnDone, errorfamily.Transient)
//	}
//
// This is for errors you don't own (stdlib, libraries).
// For your own errors, implement the Classified interface instead.
func (r *Registry) RegisterClassification(sentinel error, family Family) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sentinels[sentinel] = family
}

// UnregisterClassification removes a previously registered sentinel mapping.
// Thread-safe. No-op if the sentinel has no registered classification.
func (r *Registry) UnregisterClassification(sentinel error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sentinels, sentinel)
}

// RegisterClassifications registers multiple sentinel-to-Family mappings at once.
// Thread-safe.
func (r *Registry) RegisterClassifications(classifications map[error]Family) {
	r.mu.Lock()
	defer r.mu.Unlock()
	maps.Copy(r.sentinels, classifications)
}

// RegisterTemplate adds a MessageTemplate for a specific error code.
// Thread-safe. Overrides any existing template for the same code.
func (r *Registry) RegisterTemplate(code string, tmpl MessageTemplate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[strings.ToLower(code)] = tmpl
}

// UnregisterTemplate removes a previously registered template.
// Thread-safe. No-op if the code has no registered template.
func (r *Registry) UnregisterTemplate(code string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.templates, strings.ToLower(code))
}

// lookupSentinel snapshots the sentinels and walks the error chain.
// The snapshot is taken under RLock, then iterated lock-free — same
// pattern as the original lookupRegistered.
func (r *Registry) lookupSentinel(err error) (Family, bool) {
	r.mu.RLock()
	snapshot := make(map[error]Family, len(r.sentinels))
	maps.Copy(snapshot, r.sentinels)
	r.mu.RUnlock()

	for sentinel, family := range snapshot {
		if errors.Is(err, sentinel) {
			return family, true
		}
	}

	return Rejection, false
}

// lookupTemplate looks up a user-registered template by code (case-insensitive).
func (r *Registry) lookupTemplate(code string) (MessageTemplate, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tmpl, ok := r.templates[strings.ToLower(code)]
	return tmpl, ok
}
