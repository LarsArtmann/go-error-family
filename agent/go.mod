module github.com/larsartmann/go-error-family/agent

go 1.26.3

require (
	github.com/larsartmann/go-error-family v0.5.0
	github.com/larsartmann/go-error-family/diagnose v0.1.0
)

// Local replaces until root/diagnose get published versions that
// no longer contain agent/ and diagnose/ as sub-packages.
replace (
	github.com/larsartmann/go-error-family => ..
	github.com/larsartmann/go-error-family/diagnose => ../diagnose
)
