package errorfamily

import (
	"errors"
	"maps"
	"strings"
	"sync"
	"sync/atomic"
)

// sentinelMap is the immutable snapshot stored behind an atomic.Pointer.
// Reads load the pointer once (lock-free, allocation-free); writes copy-on-write.
type sentinelMap map[error]Family

// Registry holds classification sentinels and message templates.
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
//
// Sentinels are stored behind an atomic.Pointer to an immutable snapshot: the
// hot read path (every Classify) is lock-free and allocation-free, while rare
// writes serialize under the write lock and publish a new snapshot via copy-on-write.
type Registry struct {
	mu        sync.RWMutex // serializes writers (templates direct + sentinels copy-on-write)
	sentinels atomic.Pointer[sentinelMap]
	templates map[string]MessageTemplate
}

// NewRegistry creates an empty Registry ready for use.
func NewRegistry() *Registry {
	r := &Registry{
		templates: make(map[string]MessageTemplate),
	}
	empty := make(sentinelMap)
	r.sentinels.Store(&empty)
	return r
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
	// Pick the worst (highest-severity) sub-error, independent of join order.
	// Preserves fail-closed retry semantics: any non-Transient sub-error
	// (severity > Transient's) makes the joined result non-Transient.
	if u, ok := err.(interface{ Unwrap() []error }); ok {
		worst := Transient
		for _, sub := range u.Unwrap() {
			if f := r.Classify(sub); f.Severity() > worst.Severity() {
				worst = f
			}
		}
		return worst
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
	r.swapSentinels(func(m sentinelMap) { m[sentinel] = family })
}

// UnregisterClassification removes a previously registered sentinel mapping.
// Thread-safe. No-op if the sentinel has no registered classification.
func (r *Registry) UnregisterClassification(sentinel error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.swapSentinels(func(m sentinelMap) { delete(m, sentinel) })
}

// RegisterClassifications registers multiple sentinel-to-Family mappings at once.
// Thread-safe.
func (r *Registry) RegisterClassifications(classifications map[error]Family) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.swapSentinels(func(m sentinelMap) { maps.Copy(m, classifications) })
}

// swapSentinels performs copy-on-write: clones the current immutable sentinel
// snapshot, applies fn to the clone, and atomically publishes it. Callers must
// hold r.mu (write lock) so concurrent writers never lose updates; readers
// remain fully lock-free.
func (r *Registry) swapSentinels(fn func(m sentinelMap)) {
	old := r.sentinels.Load()
	newMap := make(sentinelMap, 1)
	if old != nil {
		newMap = make(sentinelMap, len(*old)+1)
		maps.Copy(newMap, *old)
	}
	fn(newMap)
	r.sentinels.Store(&newMap)
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

// lookupSentinel loads the immutable sentinel snapshot once (lock-free,
// allocation-free) and walks the error chain against it.
func (r *Registry) lookupSentinel(err error) (Family, bool) {
	m := r.sentinels.Load()
	if m == nil {
		return Rejection, false
	}

	for sentinel, family := range *m {
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
