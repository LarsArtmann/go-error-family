module github.com/larsartmann/go-error-family/diagnose

go 1.26

require github.com/larsartmann/go-error-family v0.10.1

// diagnose/v0.1.0 shipped a local-directory replace directive (=> ..)
// that breaks every consumer building from the module proxy. Superseded
// by v0.2.5.
retract v0.1.0
