package errorfamily

import "strings"

// Common string constants to satisfy goconst linter.
const (
	strRejection      = "rejection"
	strConflict       = "conflict"
	strTransient      = "transient"
	strCorruption     = "corruption"
	strInfrastructure = "infrastructure"
	strUnknown        = "unknown"
	msgCheckInput     = "Check your input and try again."
	msgRefreshData    = "Refresh your data and try the operation again."
)

// Family classifies an error's behavioral profile for automated handling.
//
// One concept serving three audiences:
//   - Retry loops: "Should I try again?" (Transient = yes)
//   - Exit codes: "Which exit code for the shell?" (maps to BSD sysexits.h)
//   - Presentation: "Whose fault is it?" (determines tone and framing in user messages)
type Family int

const (
	// Rejection indicates bad input, unauthorized access, or resource not found.
	// No state changed. Not retryable. User's fault. Tone: helpful, instructional.
	Rejection Family = iota

	// Conflict indicates version mismatch, duplicate creation, or state machine violation.
	// No state changed for requester. Not retryable. User needs to resolve. Tone: explanatory.
	Conflict

	// Transient indicates a temporary infrastructure failure.
	// State unknown. Retryable with backoff. System's fault. Tone: reassuring.
	Transient

	// Corruption indicates the source of truth is damaged (unparseable payload, schema break).
	// Not self-healable. Not retryable. Serious. Tone: urgent, escalate to ops.
	Corruption

	// Infrastructure indicates the system cannot serve (closed, nil deps, startup failure).
	// Not retryable. System's fault. Tone: apologetic.
	Infrastructure
)

// familyInfo holds all per-family data in one place.
// Adding a new family requires exactly one entry here.
type familyInfo struct {
	Name    string
	Exit    int
	Tone    Tone
	Message string
	Why     string
	Fix     string
}

var familyData = [...]familyInfo{
	Rejection: {
		Name:    strRejection,
		Exit:    1,
		Tone:    ToneInstructional,
		Message: "The request was invalid. Check your input and try again.",
		Fix:     msgCheckInput,
	},
	Conflict: {
		Name:    strConflict,
		Exit:    1,
		Tone:    ToneExplanatory,
		Message: "A conflict was detected. Refresh and try again.",
		Fix:     msgRefreshData,
	},
	Transient: {
		Name:    strTransient,
		Exit:    75,
		Tone:    ToneReassuring,
		Message: "A temporary error occurred. Please try again in a few moments.",
		Why:     "This is a temporary issue. No data was lost.",
		Fix:     "Wait a moment and try again.",
	},
	Corruption: {
		Name:    strCorruption,
		Exit:    65,
		Tone:    ToneUrgent,
		Message: "Data appears to be corrupted. This requires manual intervention.",
		Why:     "Some data appears to be damaged. This requires attention.",
		Fix:     "This may require manual intervention. Check the logs for details.",
	},
	Infrastructure: {
		Name:    strInfrastructure,
		Exit:    69,
		Tone:    ToneApologetic,
		Message: "The service is currently unavailable. Please try again later.",
		Why:     "This is a system issue, not something you caused.",
		Fix:     "The service may be temporarily unavailable. Try again later.",
	},
}

func (f Family) String() string {
	if f.IsValid() {
		return familyData[f].Name
	}
	return strUnknown
}

// ParseFamily parses a family string, case-insensitive.
// Returns Transient for unrecognized values (fail-open for retry).
func ParseFamily(s string) Family {
	lower := strings.ToLower(s)
	for i, info := range familyData {
		if info.Name == lower {
			return Family(i)
		}
	}
	return Transient
}

// IsRetryable reports whether operations that encounter this error should be retried.
func (f Family) IsRetryable() bool {
	return f == Transient
}

// IsValid reports whether the Family value is one of the five defined constants.
func (f Family) IsValid() bool {
	return f >= Rejection && f <= Infrastructure
}

// ExitCode returns the BSD sysexits.h compatible exit code for this family.
func (f Family) ExitCode() int {
	if f.IsValid() {
		return familyData[f].Exit
	}
	return 70 // EX_SOFTWARE — internal software error
}

// DefaultMessage returns the default human-readable message for this family.
func (f Family) DefaultMessage() string {
	if f.IsValid() {
		return familyData[f].Message
	}
	return "An unexpected error occurred."
}

// DefaultWhy returns the default "why" explanation for this family.
func (f Family) DefaultWhy() string {
	if f.IsValid() {
		return familyData[f].Why
	}
	return ""
}

// DefaultFix returns the default fix suggestion for this family.
func (f Family) DefaultFix() string {
	if f.IsValid() {
		return familyData[f].Fix
	}
	return "Try again or contact support."
}

// Audience describes who should be notified about this error.
type Audience int

const (
	// AudienceUser indicates the error is the requester's concern.
	AudienceUser Audience = iota
	// AudienceOps indicates the error needs operational intervention.
	AudienceOps
	// AudienceAll indicates all parties should be notified.
	AudienceAll
)

func (a Audience) String() string {
	switch a {
	case AudienceUser:
		return "user"
	case AudienceOps:
		return "ops"
	case AudienceAll:
		return "all"
	default:
		return strUnknown
	}
}

// Audience returns who should be notified about errors of this family.
func (f Family) Audience() Audience {
	switch f {
	case Rejection, Conflict:
		return AudienceUser
	case Corruption, Infrastructure:
		return AudienceOps
	case Transient:
		return AudienceAll
	default:
		return AudienceOps
	}
}

// Tone returns a presentation tone hint for this family.
// Used by the presentation layer to choose appropriate language.
type Tone string

const (
	ToneInstructional Tone = "instructional"
	ToneExplanatory   Tone = "explanatory"
	ToneReassuring    Tone = "reassuring"
	ToneUrgent        Tone = "urgent"
	ToneApologetic    Tone = "apologetic"
)

func (f Family) Tone() Tone {
	if f.IsValid() {
		return familyData[f].Tone
	}
	return ToneApologetic
}
