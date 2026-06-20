// Example: HTTP middleware that returns structured error responses
// with context-aware body JSON and correct HTTP status codes.
//
// Run: go run ./examples/cmd/http
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Retry   bool   `json:"retryable"`
}

func handleHTTPError(w http.ResponseWriter, err error) {
	family := errorfamily.Classify(err)
	status := family.HTTPStatus()

	var code string
	if c, ok := errors.AsType[errorfamily.Coded](err); ok {
		code = c.ErrorCode()
	} else {
		code = family.String()
	}

	var msg string
	e := &errorfamily.Error{}
	if errors.As(err, &e) {
		msg = e.Message()
	} else {
		msg = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorResponse{
		Code:    code,
		Message: msg,
		Retry:   family.IsRetryable(),
	}); err != nil {
		_, _ = fmt.Fprint(w, `{"error":"encoding failed"}`)
	}
}

type appHandler func(w http.ResponseWriter, r *http.Request) error

func (h appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		handleHTTPError(w, err)
	}
}

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
	_, _ = fmt.Fprintf(w, `{"user": {"id": %q}}\n`, id)
	return nil
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/user", appHandler(getUser))
	fmt.Println("Server on :8080")
	fmt.Println("curl http://localhost:8080/user          → 400 Bad Request")
	fmt.Println("curl http://localhost:8080/user?id=notfound → 404 Not Found")
	fmt.Println("curl http://localhost:8080/user?id=dbfail   → 503 Service Unavailable")
	_ = http.ListenAndServe(":8080", mux)
}
