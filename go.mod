module github.com/larsartmann/go-error-family

go 1.26.3

// Local replace for extracted modules (not yet published).
// go.work handles workspace builds; replace lets go mod tidy resolve.
// Remove these once diagnose and agent get their first published tags.
replace (
	github.com/larsartmann/go-error-family/agent => ./agent
	github.com/larsartmann/go-error-family/diagnose => ./diagnose
)

require github.com/larsartmann/go-error-family/diagnose v0.0.0-20260620144034-cd7cba79b5a0
