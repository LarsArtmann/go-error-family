package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/bridge"
	"github.com/larsartmann/go-error-family/examples/checkout"
	"github.com/samber/oops"
)

// --- Pattern 1: Pass-through (library classifies, application enriches) ---

func TestPattern1_LibraryRejection_SurvivesOopsEnrichment(t *testing.T) {
	store := &checkout.Store{}

	_, libErr := store.GetOrder("")
	if libErr == nil {
		t.Fatal("expected library error for empty ID")
	}

	enriched := oops.In("checkout").
		Code(errorfamily.Code(libErr)).
		Trace("trace-abc").
		With("user_id", "user-1").
		Wrap(libErr)

	family := errorfamily.Classify(enriched)

	if family != errorfamily.Rejection {
		t.Errorf(
			"Classify = %v, want Rejection (library classification should survive oops wrapping)",
			family,
		)
	}

	if code := errorfamily.Code(enriched); code != "order.missing_id" {
		t.Errorf("Code(enriched) = %q, want order.missing_id", code)
	}

	if !errors.Is(enriched, libErr) {
		t.Error("errors.Is(enriched, libErr) = false — original error should be in chain")
	}

	verbose := fmt.Sprintf("%+v", enriched)
	if !strings.Contains(verbose, "trace-abc") {
		t.Errorf("verbose output should contain trace_id, got: %s", verbose)
	}
}

func TestPattern1_LibraryTransient_SurvivesOopsEnrichment(t *testing.T) {
	store := &checkout.Store{DBUnreachable: true}

	_, libErr := store.GetOrder("order-42")
	if libErr == nil {
		t.Fatal("expected library error for DB failure")
	}

	enriched := oops.In("checkout").
		Trace("trace-def").
		Wrap(libErr)

	family := errorfamily.Classify(enriched)
	if family != errorfamily.Transient {
		t.Errorf("Classify(enriched) = %v, want Transient", family)
	}

	if !errorfamily.IsRetryable(enriched) {
		t.Error("enriched library Transient should be retryable")
	}

	if exitCode := errorfamily.ExitCode(enriched); exitCode != 75 {
		t.Errorf("ExitCode(enriched) = %d, want 75 (EX_TEMPFAIL)", exitCode)
	}
}

// --- Pattern 2: AutoWrap (oops-first, bridge infers family) ---

func TestPattern2_AutoWrap_InfersRejectionFromValidationDomain(t *testing.T) {
	rich := oops.In("validation").
		Tags("rejection").
		Code("order.bad_format").
		With("received", "!!!").
		Errorf("invalid order ID format")

	classified := bridge.AutoWrap(rich)

	if family := errorfamily.Classify(classified); family != errorfamily.Rejection {
		t.Errorf("Classify = %v, want Rejection (from validation domain)", family)
	}

	if code := classified.ErrorCode(); code != "order.bad_format" {
		t.Errorf("ErrorCode = %q, want order.bad_format", code)
	}

	if errorfamily.IsRetryable(classified) {
		t.Error("Rejection should not be retryable")
	}

	ctx := classified.ErrorContext()
	if ctx["domain"] != "validation" {
		t.Errorf("ErrorContext[domain] = %q, want validation", ctx["domain"])
	}
}

func TestPattern2_AutoWrap_TagOverridesDomain(t *testing.T) {
	rich := oops.In("validation").
		Tags("retryable").
		Errorf("transient validation failure")

	classified := bridge.AutoWrap(rich)

	if family := errorfamily.Classify(classified); family != errorfamily.Transient {
		t.Errorf("tag 'retryable' should override validation domain to Transient, got %v", family)
	}
}

// --- Pattern 3: Explicit Wrap (application knows the family) ---

func TestPattern3_ExplicitWrap_AssignsConflict(t *testing.T) {
	store := &checkout.Store{ItemOutOfStock: checkout.DefaultWidgetSKU}
	order := &checkout.Order{
		ID:    "order-1",
		Items: []checkout.LineItem{{SKU: checkout.DefaultWidgetSKU, Qty: 5}},
	}

	libErr := store.ReserveInventory(order)
	if libErr == nil {
		t.Fatal("expected inventory conflict error")
	}

	rich := oops.In("checkout").
		Code("checkout.inventory_blocked").
		Trace("trace-ghi").
		With("order_id", order.ID).
		Wrap(libErr)

	classified := bridge.Wrap(rich, errorfamily.Conflict)

	if family := errorfamily.Classify(classified); family != errorfamily.Conflict {
		t.Errorf("Classify = %v, want Conflict (explicit)", family)
	}

	if exitCode := errorfamily.ExitCode(classified); exitCode != 1 {
		t.Errorf("ExitCode = %d, want 1 (Conflict)", exitCode)
	}

	ctx := classified.ErrorContext()
	if ctx["order_id"] != "order-1" {
		t.Errorf("ErrorContext[order_id] = %q, want order-1", ctx["order_id"])
	}
}

// --- HTTP Boundary: the full classify-enrich-handle flow ---

func TestHTTPBoundary_MissingID_Returns400Rejection(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := newGetRequest("/orders?id=")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (Rejection to 400)", rec.Code, http.StatusBadRequest)
	}

	body := parseJSON(t, rec.Body.Bytes())

	if body["family"] != "rejection" {
		t.Errorf("family = %q, want rejection", body["family"])
	}

	if body["code"] != "order.missing_id" {
		t.Errorf("code = %q, want order.missing_id", body["code"])
	}

	if strings.Contains(rec.Body.String(), "order ID is required") {
		t.Error("response leaked internal error message")
	}
}

func TestHTTPBoundary_DBFailure_Returns503Transient(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := newGetRequest("/orders?id=order-42&fail=db")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d (Transient to 503)", rec.Code, http.StatusServiceUnavailable)
	}

	body := parseJSON(t, rec.Body.Bytes())

	if body["family"] != "transient" {
		t.Errorf("family = %q, want transient", body["family"])
	}
}

func TestHTTPBoundary_AutoWrapValidation_Returns400Rejection(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := newGetRequest("/orders?id=BADID")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (AutoWrap Rejection to 400)", rec.Code, http.StatusBadRequest)
	}

	body := parseJSON(t, rec.Body.Bytes())

	if body["code"] != "order.id_format" {
		t.Errorf("code = %q, want order.id_format (from AutoWrap)", body["code"])
	}
}

func TestHTTPBoundary_ExplicitWrapConflict_Returns409(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := newGetRequest("/orders?id=order-42&fail=inv")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (Conflict to 409)", rec.Code, http.StatusConflict)
	}

	body := parseJSON(t, rec.Body.Bytes())

	if body["family"] != "conflict" {
		t.Errorf("family = %q, want conflict", body["family"])
	}
}

func TestHTTPBoundary_Success_Returns200(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := newGetRequest("/orders?id=order-42")
	req.Header.Set(headerTraceID, "trace-success")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var success map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &success); err != nil {
		t.Fatalf("failed to parse success response: %v", err)
	}

	if success["order_id"] != "order-42" {
		t.Errorf("order_id = %v, want order-42", success["order_id"])
	}

	if success["trace_id"] != "trace-success" {
		t.Errorf("trace_id = %v, want trace-success", success["trace_id"])
	}
}

func TestHTTPBoundary_ResponseNeverLeaksInternalMessage(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := newGetRequest("/orders?id=order-42&fail=db")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	raw := rec.Body.String()

	forbidden := []string{"database query", "exceeded deadline", "order-42"}
	for _, frag := range forbidden {
		if strings.Contains(raw, frag) {
			t.Errorf("response leaked internal detail %q in body: %s", frag, raw)
		}
	}
}

// --- Helpers ---

func newGetRequest(target string) *http.Request {
	// httptest.NewRequest already attaches context.Background() internally;
	// noctx cannot detect the .WithContext chain that follows.
	req := httptest.NewRequest(http.MethodGet, target, nil) //nolint:noctx // see comment above

	return req.WithContext(context.Background())
}

func parseJSON(t *testing.T, b []byte) map[string]string {
	t.Helper()

	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("failed to parse JSON response %q: %v", string(b), err)
	}

	return m
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil)) //nolint:sloglint // Go 1.26 has no slog.DiscardHandler yet
}
