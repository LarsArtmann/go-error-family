package diagnose

import (
	"errors"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

func HasContextKey(err error, keys ...string) bool {
	if ctx, ok := errors.AsType[errorfamily.Contextual](err); ok {
		ctxMap := ctx.ErrorContext()
		for _, key := range keys {
			if _, ok := ctxMap[key]; ok {
				return true
			}
		}
	}
	return false
}

func ContextValue(err error, key string) string {
	if ctx, ok := errors.AsType[errorfamily.Contextual](err); ok {
		return ctx.ErrorContext()[key]
	}
	return ""
}

func ResolveContextKey(err error, keys []string, defaultVal string) string {
	for _, key := range keys {
		if v := ContextValue(err, key); v != "" {
			return v
		}
	}
	return defaultVal
}

func HasContextSubstring(err error, substr string) bool {
	lower := strings.ToLower(substr)
	if ctx, ok := errors.AsType[errorfamily.Contextual](err); ok {
		for _, v := range ctx.ErrorContext() {
			if strings.Contains(strings.ToLower(v), lower) {
				return true
			}
		}
	}
	return strings.Contains(strings.ToLower(err.Error()), lower)
}

func FamilyIs(err error, family errorfamily.Family) bool {
	return errorfamily.Classify(err) == family
}

func ErrorCodeContains(err error, substr string) bool {
	if coded, ok := errors.AsType[errorfamily.Coded](err); ok {
		return strings.Contains(strings.ToLower(coded.ErrorCode()), strings.ToLower(substr))
	}
	return false
}

func ErrorContext(err error) map[string]string {
	if ctx, ok := errors.AsType[errorfamily.Contextual](err); ok {
		return ctx.ErrorContext()
	}
	return map[string]string{}
}

type RuleSpec struct {
	ContextKeys   []ContextKey
	CodeContains  []string
	ContextSubstr []string
	Extra         func(error) bool
}

func (s RuleSpec) Matches(err error) bool {
	if len(s.ContextKeys) > 0 {
		keys := make([]string, len(s.ContextKeys))
		for i, k := range s.ContextKeys {
			keys[i] = string(k)
		}
		if HasContextKey(err, keys...) {
			return true
		}
	}
	for _, substr := range s.CodeContains {
		if ErrorCodeContains(err, substr) {
			return true
		}
	}
	for _, substr := range s.ContextSubstr {
		if HasContextSubstring(err, substr) {
			return true
		}
	}
	if s.Extra != nil && s.Extra(err) {
		return true
	}
	return false
}

const (
	strTrue      = "true"
	strFalse     = "false"
	strHost      = "host"
	strPort      = "port"
	strLocalhost = "localhost"
	strUnknown   = "unknown"
)
