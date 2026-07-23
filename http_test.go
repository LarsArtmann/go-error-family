package errorfamily

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{NewRejection("c", "m"), 400},
		{NewConflict("c", "m"), 409},
		{NewTransient("c", "m"), 503},
		{NewCorruption("c", "m"), 500},
		{NewInfrastructure("c", "m"), 503},
	}
	for _, tt := range tests {
		if got := HTTPStatus(tt.err); got != tt.want {
			t.Errorf("HTTPStatus(%v) = %d, want %d", tt.err, got, tt.want)
		}
	}
}

func TestHTTPHandlerSuccess(t *testing.T) {
	handler := HTTPHandler(func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))

		return nil
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil),
	)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want ok", rec.Body.String())
	}
}

func TestHTTPHandlerClassifiedError(t *testing.T) {
	RegisterTemplate("http.test", MessageTemplate{What: "A test error occurred."})
	t.Cleanup(func() { UnregisterTemplate("http.test") })

	handler := HTTPHandler(func(_ http.ResponseWriter, _ *http.Request) error {
		return NewConflict("http.test", "internal details that should not leak")
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", nil),
	)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q, want application/json; charset=utf-8", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}

	if body["family"] != "conflict" {
		t.Errorf("family = %q, want conflict", body["family"])
	}

	if body["code"] != "http.test" {
		t.Errorf("code = %q, want http.test", body["code"])
	}

	if body["message"] != "A test error occurred." {
		t.Errorf("message = %q, want template message", body["message"])
	}

	if strings.Contains(rec.Body.String(), "internal details") {
		t.Errorf("response leaked internal error details: %s", rec.Body.String())
	}
}

func TestHTTPHandlerPlainErrorNoLeak(t *testing.T) {
	handler := HTTPHandler(func(_ http.ResponseWriter, _ *http.Request) error {
		return errors.New("secret internal failure")
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil),
	)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 (Transient)", rec.Code)
	}

	if strings.Contains(rec.Body.String(), "secret internal failure") {
		t.Errorf("plain error leaked internal message: %s", rec.Body.String())
	}
}

type failingResponseWriter struct {
	header http.Header
}

func (f *failingResponseWriter) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}

	return f.header
}

func (f *failingResponseWriter) Write(
	[]byte,
) (int, error) {
	return 0, errors.New("connection broken")
}
func (f *failingResponseWriter) WriteHeader(statusCode int) {}

func TestWriteHTTPErrorMarshalFailure(t *testing.T) {
	w := &failingResponseWriter{}
	writeHTTPError(w, NewTransient("test.fail", "connection will break"))
}

func TestHTTPStatusWithOverride(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			"rejection default 400",
			NewRejection("battle.invalid", "invalid"),
			400,
		},
		{
			"rejection overridden to 404",
			NewRejection("battle.not_found", "not found").WithHTTPStatus(http.StatusNotFound),
			404,
		},
		{
			"conflict overridden to 422",
			NewConflict("state.stale", "stale").WithHTTPStatus(http.StatusUnprocessableEntity),
			422,
		},
		{
			"transient default 503",
			NewTransient("db.timeout", "timed out"),
			503,
		},
		{
			"transient overridden to 502",
			NewTransient("db.timeout", "timed out").WithHTTPStatus(http.StatusBadGateway),
			502,
		},
		{
			"override zero means family default",
			NewRejection("c", "m").WithHTTPStatus(0),
			400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTTPStatus(tt.err); got != tt.want {
				t.Errorf("HTTPStatus(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestWithHTTPStatusCopyOnWrite(t *testing.T) {
	original := NewRejection("battle.not_found", "not found")
	modified := original.WithHTTPStatus(http.StatusNotFound)

	if original.HTTPStatus() != 0 {
		t.Errorf("original HTTPStatus = %d, want 0 (unchanged)", original.HTTPStatus())
	}

	if modified.HTTPStatus() != http.StatusNotFound {
		t.Errorf("modified HTTPStatus = %d, want %d", modified.HTTPStatus(), http.StatusNotFound)
	}
}

func TestHTTPHandlerWithStatusOverride(t *testing.T) {
	handler := HTTPHandler(func(_ http.ResponseWriter, _ *http.Request) error {
		return NewRejection("battle.not_found", "battle not found").
			WithHTTPStatus(http.StatusNotFound)
	})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(
		rec,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil),
	)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 (override)", rec.Code)
	}
}

func TestErrorImplementsHTTPStatuser(t *testing.T) {
	var _ HTTPStatuser = NewRejection("test", "msg")
}
