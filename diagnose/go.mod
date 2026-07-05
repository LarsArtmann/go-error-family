module github.com/larsartmann/go-error-family/diagnose

go 1.26.4

require github.com/larsartmann/go-error-family v0.0.0-00010101000000-000000000000

// Local replace until root gets a published version that no longer
// contains diagnose/ as a sub-package.
replace github.com/larsartmann/go-error-family => ..
