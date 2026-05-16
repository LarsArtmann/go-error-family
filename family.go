package errorfamily

import "strings"

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

func (f Family) String() string {
	switch f {
	case Rejection:
		return "rejection"
	case Conflict:
		return "conflict"
	case Transient:
		return "transient"
	case Corruption:
		return "corruption"
	case Infrastructure:
		return "infrastructure"
	default:
		return "unknown"
	}
}

// ParseFamily parses a family string, case-insensitive.
// Returns Transient for unrecognized values (fail-open for retry).
func ParseFamily(s string) Family {
	switch strings.ToLower(s) {
	case "rejection":
		return Rejection
	case "conflict":
		return Conflict
	case "transient":
		return Transient
	case "corruption":
		return Corruption
	case "infrastructure":
		return Infrastructure
	default:
		return Transient
	}
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
// Shell scripts and CI pipelines can use this to make automated decisions.
func (f Family) ExitCode() int {
	switch f {
	case Rejection:
		return 1 // EX_USAGE — user error
	case Conflict:
		return 1 // EX_USAGE — state conflict (user needs to resolve)
	case Transient:
		return 75 // EX_TEMPFAIL — temporary failure, try again
	case Corruption:
		return 65 // EX_DATAERR — data is damaged
	case Infrastructure:
		return 69 // EX_UNAVAILABLE — service unavailable
	default:
		return 70 // EX_SOFTWARE — internal software error
	}
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
	switch f {
	case Rejection:
		return ToneInstructional
	case Conflict:
		return ToneExplanatory
	case Transient:
		return ToneReassuring
	case Corruption:
		return ToneUrgent
	case Infrastructure:
		return ToneApologetic
	default:
		return ToneApologetic
	}
}
