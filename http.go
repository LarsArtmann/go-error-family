package errorfamily

import (
	"encoding/json/v2"
	"net/http"
)

// HTTPStatus returns the recommended HTTP response status code for an error by
// classifying it. It is a thin convenience wrapper around
// [Classify](err).[Family.HTTPStatus], for use at HTTP/REST boundaries:
//
//	w.WriteHeader(errorfamily.HTTPStatus(err))
//
// Nil errors classify as Rejection → 400; prefer checking for nil before
// reaching the HTTP layer.
func HTTPStatus(err error) int {
	return Classify(err).HTTPStatus()
}

// HandlerFunc is an HTTP handler that returns an error. When the returned error
// is nil, the request is considered fully handled. A non-nil error is classified
// and translated into an HTTP error response by [HTTPHandler].
//
// Use this with [HTTPHandler] to bridge go-error-family classification into the
// net/http stack without per-handler boilerplate.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// HTTPHandler wraps a [HandlerFunc], classifying any returned error and writing a
// JSON error response with the status code from [Family.HTTPStatus].
//
//	mux.Handle("/api/orders", errorfamily.HTTPHandler(createOrder))
//
// The response body is intentionally safe — it never leaks the raw error
// message. It contains:
//
//   - "family": the behavioral family (e.g. "rejection")
//   - "code":   the machine-readable code (omitted when empty)
//   - "message": a user-facing message from a registered [MessageTemplate]
//     (omitted when no template is registered for the code)
//
// For a custom response shape, write your own response inside the handler on the
// success path and use [HTTPStatus] directly for the failure path.
func HTTPHandler(h HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}

		writeHTTPError(w, err)
	})
}

func writeHTTPError(w http.ResponseWriter, err error) {
	family := Classify(err)
	code := Code(err)

	body := map[string]string{
		"family": family.String(),
	}
	if code != "" {
		body["code"] = code
	}
	// Safe user-facing message from a registered template, if one exists.
	// Never include the raw err.Error() — it may leak internals.
	if tmpl, ok := TemplateForCode(code); ok && tmpl.What != "" {
		body["message"] = tmpl.What
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(family.HTTPStatus())

	if err := json.MarshalWrite(w, body); err != nil {
		// Client disconnected or connection broke — nothing actionable.
		return
	}
}
