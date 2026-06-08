package errorfamily

import (
	"errors"
	"fmt"
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
