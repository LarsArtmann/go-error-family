module github.com/larsartmann/go-error-family/agent

go 1.26

require (
	github.com/larsartmann/go-error-family v0.10.1
	github.com/larsartmann/go-error-family/diagnose v0.2.5
)

// agent/v0.1.0 shipped local-directory replace directives (=> .. and
// => ../diagnose) that break every consumer building from the module
// proxy. Superseded by v0.2.5.
retract v0.1.0
