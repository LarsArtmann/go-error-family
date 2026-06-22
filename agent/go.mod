module github.com/larsartmann/go-error-family/agent

go 1.26.3

require (
	github.com/larsartmann/go-error-family v0.4.0
	github.com/larsartmann/go-error-family/diagnose v0.0.0-20260622173950-5ede599ce76d
)

// Local replaces until root/diagnose get published versions that
// no longer contain agent/ and diagnose/ as sub-packages.
replace (
	github.com/larsartmann/go-error-family => ..
	github.com/larsartmann/go-error-family/diagnose => ../diagnose
)
