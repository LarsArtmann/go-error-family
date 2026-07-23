// Example: CLI error handler with contextual messages and exit codes.
//
// Run: go run ./examples/cmd/cli
package main

import (
	"errors"
	"fmt"
	"os"

	errorfamily "github.com/larsartmann/go-error-family"
)

var errConnectionRefused = errors.New("connection refused")

func fetchConfig(path string) error {
	if path == "/etc/config.yaml" {
		return errorfamily.NewRejection("file.not_found", "configuration file missing").
			WithContext("path", path).
			WithContext("suggestion", "Create /etc/config.yaml from the template")
	}

	return nil
}

func connectDB(host string) error {
	if host == "badhost" {
		return errConnectionRefused
	}

	return nil
}

func run() error {
	if err := fetchConfig("/etc/config.yaml"); err != nil {
		return errorfamily.WrapRejection(err, "startup.failed", "could not initialize")
	}

	if err := connectDB("localhost"); err != nil {
		return errorfamily.WrapTransient(err, "db.connection", "database unreachable")
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		os.Exit(errorfamily.HandleError(err))
	}

	fmt.Println("Success")
}
