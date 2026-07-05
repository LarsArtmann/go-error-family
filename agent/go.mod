module github.com/larsartmann/go-error-family/agent

go 1.26.4

require (
	github.com/larsartmann/go-error-family v0.0.0-00010101000000-000000000000
	github.com/larsartmann/go-error-family/diagnose v0.0.0-00010101000000-000000000000
)

// Local replaces until root/diagnose get published versions that
// no longer contain agent/ and diagnose/ as sub-packages.
replace (
	github.com/larsartmann/go-error-family => ..
	github.com/larsartmann/go-error-family/diagnose => ../diagnose
)
