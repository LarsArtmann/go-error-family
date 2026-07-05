// Example: HTTP service returning structured, safe error responses.
//
// This example uses [errorfamily.HTTPHandler] and [errorfamily.HandlerFunc],
// which deliberately NEVER leak err.Error() to the client. The response body
// contains only the behavioral family, the machine-readable code, and an
// optional user-facing message from a registered [errorfamily.MessageTemplate].
//
// Run: go run ./examples/cmd/http
package main

import (
	"fmt"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

func getUser(w http.ResponseWriter, r *http.Request) error {
	id := r.URL.Query().Get("id")
	if id == "" {
		return errorfamily.NewRejection("user.missing_id", "id query parameter is required")
	}
	if id == "notfound" {
		return errorfamily.NewRejection("user.not_found", "user not found").
			WithContext("id", id)
	}
	if id == "dbfail" {
		return errorfamily.NewTransient("db.timeout", "database connection timed out")
	}
	_, _ = fmt.Fprintf(w, `{"user": {"id": %q}}`+"\n", id)
	return nil
}

func main() {
	// Register a user-facing message template for the missing_id code.
	// HTTPHandler surfaces this as the response "message" field — safe to show.
	errorfamily.RegisterTemplate("user.missing_id", errorfamily.MessageTemplate{
		What: "Please provide an id query parameter.",
	})

	mux := http.NewServeMux()
	mux.Handle("/user", errorfamily.HTTPHandler(getUser))

	fmt.Println("Server on :8080")
	fmt.Println("curl 'http://localhost:8080/user'             → 400 (Rejection)")
	fmt.Println("curl 'http://localhost:8080/user?id=notfound' → 400 (Rejection)")
	fmt.Println("curl 'http://localhost:8080/user?id=dbfail'   → 503 (Transient)")
	_ = http.ListenAndServe(":8080", mux)
}
