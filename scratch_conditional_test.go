package errorfamily

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Scratch verification for issue #5 (deleted after verification).

func TestScratchWithHTTPStatusPattern(t *testing.T) {
	err := NewConflict("etag.precondition_failed", "If-Match precondition failed").
		WithHTTPStatus(http.StatusPreconditionFailed)

	if got := HTTPStatus(err); got != 412 {
		t.Errorf("HTTPStatus = %d, want 412", got)
	}
	if got := Classify(err); got != Conflict {
		t.Errorf("Classify = %v, want Conflict", got)
	}
	if IsRetryable(err) {
		t.Error("412 error must not be retryable")
	}
	if got := ExitCode(err); got != 1 {
		t.Errorf("ExitCode = %d, want 1 (Conflict family default)", got)
	}

	// 428 as Rejection + override
	err428 := NewRejection("etag.precondition_required", "server requires conditional requests").
		WithHTTPStatus(http.StatusPreconditionRequired)
	if got := HTTPStatus(err428); got != 428 {
		t.Errorf("HTTPStatus(428 pattern) = %d, want 428", got)
	}
	if got := Classify(err428); got != Rejection {
		t.Errorf("Classify(428 pattern) = %v, want Rejection", got)
	}
}

// The wrapper-struct pattern proposed verbatim in the issue.
type scratchPreconditionFailed struct{ *Error }

func (e *scratchPreconditionFailed) HTTPStatus() int { return http.StatusPreconditionFailed }

func TestScratchWrapperStructPattern(t *testing.T) {
	err := &scratchPreconditionFailed{NewConflict(
		"etag.precondition_failed",
		"If-Match precondition failed",
	)}

	if got := HTTPStatus(err); got != 412 {
		t.Errorf("HTTPStatus = %d, want 412", got)
	}
	if got := Classify(err); got != Conflict {
		t.Errorf("Classify = %v, want Conflict", got)
	}
	if got := Code(err); got != "etag.precondition_failed" {
		t.Errorf("Code = %q", got)
	}
}

func TestScratchHTTPHandlerWrites412(t *testing.T) {
	rec := httptest.NewRecorder()
	HTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		return NewConflict("etag.precondition_failed", "If-Match precondition failed").
			WithHTTPStatus(http.StatusPreconditionFailed)
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != 412 {
		t.Errorf("status = %d, want 412", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	t.Logf("body: %v", body)
	if body["family"] != "conflict" || body["code"] != "etag.precondition_failed" {
		t.Errorf("unexpected body: %v", body)
	}
	if _, leaked := body["message"]; !leaked {
		t.Log("no message key without registered template (expected)")
	}
}

// Nil-error behavior at the HTTP boundary (for the 304 guidance).
func TestScratchNilAnd304(t *testing.T) {
	if got := HTTPStatus(nil); got != 400 {
		t.Errorf("HTTPStatus(nil) = %d, want 400 (documented nil behavior)", got)
	}
	// HTTPHandler treats nil return as fully handled — a 304 must be written by the handler itself.
	rec := httptest.NewRecorder()
	HTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != 304 {
		t.Errorf("status = %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("304 must not carry a body, got %q", rec.Body.String())
	}
	// errors.Is still works through the WithHTTPStatus copy-on-write chain.
	base := NewConflict("etag.precondition_failed", "msg")
	derived := base.WithHTTPStatus(412)
	if !errors.Is(derived, base) {
		t.Error("errors.Is must match code+family through WithHTTPStatus")
	}
}
