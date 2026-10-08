module github.com/larsartmann/go-error-family

go 1.27

// v0.5.0 through v0.6.0 shipped local-directory replace directives
// (./agent, ./diagnose) that break every consumer building from the
// module proxy; v0.6.0 additionally pinned a phantom pseudo-version
// require. Superseded by the extracted multi-module layout; current
// releases carry no replace directives.
retract [v0.5.0, v0.6.0]
