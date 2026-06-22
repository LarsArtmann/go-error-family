module github.com/larsartmann/go-error-family/diagnose

go 1.26.3

require github.com/larsartmann/go-error-family v0.5.0

// Local replace until root gets a published version that no longer
// contains diagnose/ as a sub-package.
replace github.com/larsartmann/go-error-family => ..
