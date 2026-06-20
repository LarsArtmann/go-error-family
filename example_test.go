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
