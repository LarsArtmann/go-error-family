module github.com/larsartmann/go-error-family/diagnose

go 1.26.4

require github.com/larsartmann/go-error-family v0.5.1

// Local replace until root gets a published version that no longer
// contains diagnose/ as a sub-package.
replace github.com/larsartmann/go-error-family => ..
