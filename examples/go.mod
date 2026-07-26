module github.com/larsartmann/go-error-family/examples

go 1.26.5

require (
	github.com/larsartmann/go-error-family v0.9.0
	github.com/larsartmann/go-error-family/bridge v0.0.0
	github.com/larsartmann/go-error-family/diagnose v0.2.1
	github.com/samber/oops v1.23.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/oklog/ulid/v2 v2.1.2 // indirect
	github.com/samber/lo v1.53.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)

replace github.com/larsartmann/go-error-family/bridge => ../bridge
