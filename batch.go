package errorfamily

import (
	"fmt"
	"sync"
)

// Outcome represents the result of processing a single item in a batch operation.
type Outcome[T any] struct {
	Value T
	Err   error
}

// IsOK reports whether the item was processed successfully.
func (o Outcome[T]) IsOK() bool { return o.Err == nil }

// Outcome creates an Outcome from a value and optional error.
// If err is nil, the outcome is successful.
func NewOutcome[T any](value T, err error) Outcome[T] {
	return Outcome[T]{Value: value, Err: err}
}

// --- ErrorBatch: error-only collector ---

// ErrorBatch collects errors from a batch or multi-step operation.
// Thread-safe for concurrent Add calls.
//
// Use ErrorBatch when you only need to track which operations failed.
// Use BatchResult[T] when you also need to retain successful values.
//
//	batch := errorfamily.NewErrorBatch()
//	for _, item := range items {
//	    if err := process(item); err != nil {
//	        batch.Add(err)
//	    }
//	}
//	if batch.HasFailures() {
//	    os.Exit(batch.ExitCode())
//	}
type ErrorBatch struct {
	mu     sync.Mutex
	errors []error
}

// NewErrorBatch creates an empty error batch.
func NewErrorBatch() *ErrorBatch {
	return &ErrorBatch{}
}

// Add records an error. Nil errors are ignored.
// Thread-safe.
func (b *ErrorBatch) Add(err error) {
	if err == nil {
		return
	}
	b.mu.Lock()
	b.errors = append(b.errors, err)
	b.mu.Unlock()
}

// AddBatch merges another ErrorBatch's errors into this one.
// Thread-safe.
func (b *ErrorBatch) AddBatch(other *ErrorBatch) {
	other.mu.Lock()
	copied := make([]error, len(other.errors))
	copy(copied, other.errors)
	other.mu.Unlock()

	b.mu.Lock()
	b.errors = append(b.errors, copied...)
	b.mu.Unlock()
}

// Len returns the number of recorded errors.
func (b *ErrorBatch) Len() int {
	b.mu.Lock()
	n := len(b.errors)
	b.mu.Unlock()
	return n
}

// HasFailures reports whether any errors were recorded.
func (b *ErrorBatch) HasFailures() bool { return b.Len() > 0 }

// Errors returns a copy of all recorded errors.
func (b *ErrorBatch) Errors() []error {
	b.mu.Lock()
	copied := make([]error, len(b.errors))
	copy(copied, b.errors)
	b.mu.Unlock()
	return copied
}

// Families returns a histogram of error families among recorded errors.
func (b *ErrorBatch) Families() map[Family]int {
	errors := b.Errors()
	m := make(map[Family]int, len(errors))
	for _, err := range errors {
		m[Classify(err)]++
	}
	return m
}

// DominantFamily returns the most severe error family among recorded errors.
// Returns Rejection if no errors are recorded.
//
// Severity order: Corruption > Infrastructure > Conflict > Rejection > Transient.
func (b *ErrorBatch) DominantFamily() Family {
	b.mu.Lock()
	errors := b.errors
	b.mu.Unlock()
	return dominantFamily(errors)
}

// HasRetryable reports whether any recorded error is Transient (retryable).
func (b *ErrorBatch) HasRetryable() bool {
	for _, err := range b.Errors() {
		if IsRetryable(err) {
			return true
		}
	}
	return false
}

// Retryable returns all recorded errors that are Transient (retryable).
func (b *ErrorBatch) Retryable() []error {
	var result []error
	for _, err := range b.Errors() {
		if IsRetryable(err) {
			result = append(result, err)
		}
	}
	return result
}

// Err returns an error representing the batch failure, or nil if no errors were recorded.
// The returned error implements Classified, Coded, Contextual, and Retryable,
// so it integrates with HandleError, Classify, and all existing tools.
func (b *ErrorBatch) Err() error {
	b.mu.Lock()
	errors := make([]error, len(b.errors))
	copy(errors, b.errors)
	b.mu.Unlock()

	if len(errors) == 0 {
		return nil
	}

	return &batchError{
		total:     len(errors),
		succeeded: 0,
		family:    dominantFamily(errors),
		errors:    errors,
	}
}

// ExitCode returns the process exit code for CLI boundary.
// Returns 0 if no failures, otherwise the dominant family's exit code.
func (b *ErrorBatch) ExitCode() int {
	if !b.HasFailures() {
		return 0
	}
	return b.DominantFamily().ExitCode()
}

// --- BatchResult: value + error collector ---

// BatchResult collects outcomes from a batch or multi-step operation.
// Thread-safe for concurrent Add calls.
//
// Use BatchResult when you need to retain both successful values and errors.
// Use ErrorBatch when you only need to track errors.
//
//	result := errorfamily.NewBatchResult[ProcessedItem]()
//	for _, item := range items {
//	    processed, err := process(item)
//	    result.Add(processed, err)
//	}
//
//	for _, item := range result.Successes() {
//	    fmt.Println("OK:", item)
//	}
//	for _, failure := range result.Failures() {
//	    fmt.Printf("FAIL: %v\n", failure.Err)
//	}
type BatchResult[T any] struct {
	mu       sync.Mutex
	outcomes []Outcome[T]
}

// NewBatchResult creates an empty batch result.
func NewBatchResult[T any]() *BatchResult[T] {
	return &BatchResult[T]{}
}

// Add records an outcome with a value and optional error.
// Thread-safe.
func (br *BatchResult[T]) Add(value T, err error) {
	br.mu.Lock()
	br.outcomes = append(br.outcomes, Outcome[T]{Value: value, Err: err})
	br.mu.Unlock()
}

// AddOutcome records a pre-built Outcome.
// Thread-safe.
func (br *BatchResult[T]) AddOutcome(o Outcome[T]) {
	br.mu.Lock()
	br.outcomes = append(br.outcomes, o)
	br.mu.Unlock()
}

// AddResult merges another BatchResult's outcomes into this one.
// Thread-safe.
func (br *BatchResult[T]) AddResult(other *BatchResult[T]) {
	other.mu.Lock()
	copied := make([]Outcome[T], len(other.outcomes))
	copy(copied, other.outcomes)
	other.mu.Unlock()

	br.mu.Lock()
	br.outcomes = append(br.outcomes, copied...)
	br.mu.Unlock()
}

// Len returns the total number of outcomes.
func (br *BatchResult[T]) Len() int {
	br.mu.Lock()
	n := len(br.outcomes)
	br.mu.Unlock()
	return n
}

// Successes returns all values from successful outcomes.
func (br *BatchResult[T]) Successes() []T {
	br.mu.Lock()
	outcomes := br.outcomes
	br.mu.Unlock()

	var result []T
	for _, o := range outcomes {
		if o.Err == nil {
			result = append(result, o.Value)
		}
	}
	return result
}

// Failures returns all outcomes that had errors.
func (br *BatchResult[T]) Failures() []Outcome[T] {
	br.mu.Lock()
	outcomes := br.outcomes
	br.mu.Unlock()

	var result []Outcome[T]
	for _, o := range outcomes {
		if o.Err != nil {
			result = append(result, o)
		}
	}
	return result
}

// HasFailures reports whether any outcome failed.
func (br *BatchResult[T]) HasFailures() bool {
	br.mu.Lock()
	defer br.mu.Unlock()
	for _, o := range br.outcomes {
		if o.Err != nil {
			return true
		}
	}
	return false
}

// AllSucceeded reports whether every outcome succeeded.
func (br *BatchResult[T]) AllSucceeded() bool {
	br.mu.Lock()
	defer br.mu.Unlock()
	for _, o := range br.outcomes {
		if o.Err != nil {
			return false
		}
	}
	return len(br.outcomes) > 0
}

// AllFailed reports whether every outcome failed.
func (br *BatchResult[T]) AllFailed() bool {
	br.mu.Lock()
	defer br.mu.Unlock()
	if len(br.outcomes) == 0 {
		return false
	}
	for _, o := range br.outcomes {
		if o.Err == nil {
			return false
		}
	}
	return true
}

// IsPartial reports whether some outcomes succeeded and some failed.
// Returns false if there are no outcomes, all succeeded, or all failed.
func (br *BatchResult[T]) IsPartial() bool {
	br.mu.Lock()
	defer br.mu.Unlock()
	if len(br.outcomes) == 0 {
		return false
	}
	hasOK, hasFail := false, false
	for _, o := range br.outcomes {
		if o.Err == nil {
			hasOK = true
		} else {
			hasFail = true
		}
	}
	return hasOK && hasFail
}

// Families returns a histogram of error families among failed outcomes.
func (br *BatchResult[T]) Families() map[Family]int {
	failures := br.Failures()
	m := make(map[Family]int, len(failures))
	for _, o := range failures {
		m[Classify(o.Err)]++
	}
	return m
}

// DominantFamily returns the most severe error family among failed outcomes.
// Returns Rejection if no failures.
//
// Severity order: Corruption > Infrastructure > Conflict > Rejection > Transient.
func (br *BatchResult[T]) DominantFamily() Family {
	failures := br.Failures()
	if len(failures) == 0 {
		return Rejection
	}
	errs := make([]error, len(failures))
	for i, o := range failures {
		errs[i] = o.Err
	}
	return dominantFamily(errs)
}

// HasRetryable reports whether any failed outcome has a Transient error.
func (br *BatchResult[T]) HasRetryable() bool {
	for _, o := range br.Failures() {
		if IsRetryable(o.Err) {
			return true
		}
	}
	return false
}

// RetryableFailures returns failed outcomes with Transient errors.
func (br *BatchResult[T]) RetryableFailures() []Outcome[T] {
	var result []Outcome[T]
	for _, o := range br.Failures() {
		if IsRetryable(o.Err) {
			result = append(result, o)
		}
	}
	return result
}

// Err returns an error representing the batch failure, or nil if all outcomes succeeded.
// The returned error implements Classified, Coded, Contextual, and Retryable,
// so it integrates with HandleError, Classify, and all existing tools.
func (br *BatchResult[T]) Err() error {
	br.mu.Lock()
	outcomes := make([]Outcome[T], len(br.outcomes))
	copy(outcomes, br.outcomes)
	br.mu.Unlock()

	if len(outcomes) == 0 {
		return nil
	}

	var errs []error
	succeeded := 0
	for _, o := range outcomes {
		if o.Err != nil {
			errs = append(errs, o.Err)
		} else {
			succeeded++
		}
	}
	if len(errs) == 0 {
		return nil
	}

	return &batchError{
		total:     len(outcomes),
		succeeded: succeeded,
		family:    dominantFamily(errs),
		errors:    errs,
	}
}

// ExitCode returns the process exit code for CLI boundary.
// Returns 0 if all succeeded, otherwise the dominant family's exit code.
func (br *BatchResult[T]) ExitCode() int {
	br.mu.Lock()
	outcomes := br.outcomes
	br.mu.Unlock()

	if len(outcomes) == 0 {
		return 0
	}
	for _, o := range outcomes {
		if o.Err != nil {
			return br.DominantFamily().ExitCode()
		}
	}
	return 0
}

// --- internal: batchError ---

// batchError is the aggregate error returned by ErrorBatch.Err() and BatchResult.Err().
// Implements Classified, Coded, Contextual, and Retryable for seamless integration.
type batchError struct {
	total     int
	succeeded int
	family    Family
	errors    []error
}

func (e *batchError) Error() string {
	failed := e.total - e.succeeded
	if e.succeeded > 0 {
		return fmt.Sprintf("[%s:batch] %d of %d items failed", e.family, failed, e.total)
	}
	return fmt.Sprintf("[%s:batch] %d items failed", e.family, failed)
}

func (e *batchError) ErrorCode() string { return "batch" }

func (e *batchError) ErrorFamily() Family { return e.family }

func (e *batchError) ErrorContext() map[string]string {
	ctx := map[string]string{
		"total":  fmt.Sprintf("%d", e.total),
		"failed": fmt.Sprintf("%d", e.total-e.succeeded),
	}
	if e.succeeded > 0 {
		ctx["succeeded"] = fmt.Sprintf("%d", e.succeeded)
	}
	return ctx
}

func (e *batchError) IsRetryable() bool { return e.family.IsRetryable() }

// Format implements fmt.Formatter.
//
//	%v    → [family:batch] K of N items failed
//	%+v   → verbose: each failure on its own line
func (e *batchError) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			failed := e.total - e.succeeded
			if e.succeeded > 0 {
				_, _ = fmt.Fprintf(f, "[%s:batch] %d of %d items failed", e.family, failed, e.total)
			} else {
				_, _ = fmt.Fprintf(f, "[%s:batch] %d items failed", e.family, failed)
			}
			for i, err := range e.errors {
				_, _ = fmt.Fprintf(f, "\n  [%d] %+v", i+1, err)
			}
			return
		}
		_, _ = fmt.Fprint(f, e.Error())
	default:
		_, _ = fmt.Fprint(f, e.Error())
	}
}

// --- severity ---

// familySeverity orders families by severity for determining the dominant family.
// Higher value = more severe.
// Order: Corruption(4) > Infrastructure(3) > Conflict(2) > Rejection(1) > Transient(0).
func familySeverity(f Family) int {
	switch f {
	case Transient:
		return 0
	case Rejection:
		return 1
	case Conflict:
		return 2
	case Infrastructure:
		return 3
	case Corruption:
		return 4
	default:
		return 5
	}
}

// dominantFamily returns the most severe family among the given errors.
// Returns Transient for empty input (matches Classify default).
func dominantFamily(errs []error) Family {
	if len(errs) == 0 {
		return Transient
	}
	worst := Transient
	worstSev := familySeverity(worst)
	for _, err := range errs {
		f := Classify(err)
		if s := familySeverity(f); s > worstSev {
			worst = f
			worstSev = s
		}
	}
	return worst
}
