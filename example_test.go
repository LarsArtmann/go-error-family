package errorfamily

import (
	"errors"
	"fmt"
	"time"
)

func ExampleNewTransient() {
	err := NewTransient("db.timeout", "query took too long").
		WithContext("host", "localhost").
		WithContext("port", "5432")
	fmt.Println(err)
	// Output: [transient:db.timeout] query took too long
}

func ExampleClassify() {
	err := NewRejection("file.not_found", "config missing")
	family := Classify(err)
	fmt.Println(family.IsRetryable(), family.ExitCode())
	// Output: false 1
}

func ExampleHandleError() {
	err := NewRejection("file.not_found", "config missing").
		WithContext("path", "/etc/app/config.yaml")
	_ = HandleError(err)
}

func ExampleWrapRejection() {
	orig := errors.New("file does not exist")
	err := WrapRejection(orig, "config.invalid", "bad configuration")
	fmt.Println(err)
	// Output: [rejection:config.invalid] bad configuration: file does not exist
}

func ExampleParseFamily() {
	fmt.Println(ParseFamily("transient"))
	fmt.Println(ParseFamily("UNKNOWN"))
	// Output: transient
	// transient
}

func ExampleHandleErrorDetailed() {
	err := NewRejection("file.not_found", "config missing").
		WithContext("path", "/etc/app/config.yaml")
	result := HandleErrorDetailed(err)
	fmt.Println("exit:", result.ExitCode)
	fmt.Println("fix:", result.SuggestedFix)
	// Output:
	// exit: 1
	// fix: Check that the path and resource name are correct.
}

func ExampleRegisterClassification() {
	sentinel := errors.New("connection pool exhausted")
	RegisterClassification(sentinel, Transient)

	family := Classify(sentinel)
	fmt.Println(family)
	// Output: transient
}

func ExampleFamily_MarshalText() {
	f := Transient
	text, _ := f.MarshalText()
	fmt.Println(string(text))
	// Output: transient
}

func ExampleFamily_UnmarshalText() {
	var f Family
	_ = f.UnmarshalText([]byte("rejection"))
	fmt.Println(f)
	// Output: rejection
}

func ExampleNewRegistry() {
	// A custom Registry enables test isolation and scoped handling without
	// touching the package-level DefaultRegistry.
	reg := NewRegistry()
	reg.RegisterClassification(errors.New("sentinel"), Transient)

	fmt.Println(reg.Classify(errors.New("sentinel")))
	// Output: transient
}

func ExampleFamily_HTTPStatus() {
	// Translate a family into an HTTP status code at REST boundaries.
	err := NewConflict("order.duplicate", "order already exists")
	fmt.Println(Classify(err).HTTPStatus())
	// Output: 409
}

func ExampleError_JSON() {
	err := NewTransient("db.timeout", "query timed out").
		WithContext("host", "db1").
		WithTimestamp(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	data, _ := err.JSON()
	fmt.Println(string(data))
	// Output: {"family":"transient","code":"db.timeout","message":"query timed out","context":{"host":"db1"},"retryable":true,"timestamp":"2026-01-01T00:00:00Z"}
}

func ExampleWrapOnce() {
	// Inner layer already classified the error.
	inner := NewTransient("db.timeout", "database timed out")

	// API boundary: WrapOnce returns the existing *Error unchanged
	// instead of creating a double-wrapped chain.
	err := WrapOnce(inner, Infrastructure, "api.error", "API failed")

	fmt.Println(err == inner)
	// Output: true
}

func ExampleError_WithExitCode() {
	// Override the family-based exit code for a specific error.
	// Useful when a transient error should exit with a code that
	// differs from the BSD sysexits default (75).
	err := NewTransient("rate.limited", "too many requests").WithExitCode(88)
	fmt.Println(err.ExitCode())
	// Output: 88
}

func ExampleError_WithContextAny() {
	// Add context values of any type — they are converted to strings.
	err := NewRejection("validation.failed", "invalid input").
		WithContextAny("count", 3).
		WithContextAny("enabled", true)

	fmt.Println(err.ContextValue("count"))
	fmt.Println(err.ContextValue("enabled"))
	// Output: 3
	// true
}

func ExampleExitCode() {
	// ExitCode checks the ExitCoder interface first (per-error override),
	// then falls back to the family's canonical exit code.
	fmt.Println(ExitCode(NewTransient("db.timeout", "msg")))
	fmt.Println(ExitCode(NewRejection("bad.input", "msg")))
	fmt.Println(ExitCode(NewTransient("custom", "msg").WithExitCode(5)))
	// Output: 75
	// 1
	// 5
}
