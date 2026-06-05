package bridge

import (
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/samber/oops"
)

// Common string constants to satisfy goconst linter.
const (
	domainDatabase       = "database"
	domainInfra          = "infra"
	domainInfrastructure = "infrastructure"
)

// domainDefaults maps oops domain strings to error-family Families.
// Domains are structural signals — less intentional than tags, more reliable
// than guessing. Used when no explicit tag override is present.
var domainDefaults = map[string]errorfamily.Family{
	"validation":         errorfamily.Rejection,
	"auth":               errorfamily.Rejection,
	"authorization":      errorfamily.Rejection,
	"input":              errorfamily.Rejection,
	domainDatabase:       errorfamily.Transient,
	"network":            errorfamily.Transient,
	"cache":              errorfamily.Transient,
	"queue":              errorfamily.Transient,
	"storage":            errorfamily.Infrastructure,
	domainInfra:          errorfamily.Infrastructure,
	domainInfrastructure: errorfamily.Infrastructure,
	"startup":            errorfamily.Infrastructure,
	"data":               errorfamily.Corruption,
	"schema":             errorfamily.Corruption,
	"migration":          errorfamily.Corruption,
}

// tagOverrides maps oops tag strings to error-family Families.
// Tags are developer-intentional signals — checked before domain.
// Any oops tag matching a key here overrides the domain default.
var tagOverrides = map[string]errorfamily.Family{
	"retryable":          errorfamily.Transient,
	"transient":          errorfamily.Transient,
	"conflict":           errorfamily.Conflict,
	"corruption":         errorfamily.Corruption,
	"corrupted":          errorfamily.Corruption,
	"rejection":          errorfamily.Rejection,
	"rejected":           errorfamily.Rejection,
	domainInfrastructure: errorfamily.Infrastructure,
	domainInfra:          errorfamily.Infrastructure,
}

// InferFamily derives a Family from oops metadata.
// Classification cascade (most intentional first):
//
//  1. Tags — developer chose them explicitly, strongest signal
//  2. Domain — structural categorization, good default
//  3. Fallback — Transient (fail-open for retry, consistent with error-family)
//
// Returns Transient for non-oops errors or OopsErrors with no matching tags or domains.
func InferFamily(err error) errorfamily.Family {
	oopsErr, ok := oops.AsOops(err)
	if !ok {
		return errorfamily.Transient
	}

	// 1. Explicit tags win — developer-intentional signal.
	for _, tag := range oopsErr.Tags() {
		if family, ok := tagOverrides[tag]; ok {
			return family
		}
	}

	// 2. Domain-based defaults — structural signal.
	if family, ok := domainDefaults[oopsErr.Domain()]; ok {
		return family
	}

	// 3. Fail-open: unknown → retryable.
	return errorfamily.Transient
}

// AutoWrap infers the Family from oops metadata and creates a ClassifiedError
// in one step. This is the primary entry point when using oops-first error
// construction with error-family classification.
//
//	err := oops.In("database").Tags("timeout").With("host", "db1").Wrap(dbErr)
//	classified := bridge.AutoWrap(err)
//	errorfamily.IsRetryable(classified) // → true (domain "database" → Transient)
func AutoWrap(err error) *ClassifiedError {
	return Wrap(err, InferFamily(err))
}
