package main

import (
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

	// Library returns a classified error.
	_, libErr := store.GetOrder("")
	if libErr == nil {
		t.Fatal("expected library error for empty ID")
	}

	// Application enriches with oops — adding domain, trace, context.
	enriched := oops.In("checkout").
		Code(errorfamily.Code(libErr)).
		Trace("trace-abc").
		With("user_id", "user-1").
		Wrap(libErr)

	// The classification survives: Classify walks past the OopsError wrapper
	// and finds the library's *Error implementing Classified.
	family := errorfamily.Classify(enriched)
	if family != errorfamily.Rejection {
		t.Errorf("Classify(enriched) = %v, want Rejection (library classification should survive oops wrapping)", family)
	}

	// The code also survives.
	if code := errorfamily.Code(enriched); code != "order.missing_id" {
		t.Errorf("Code(enriched) = %q, want order.missing_id", code)
	}

	// The original library error is reachable.
	if !errors.Is(enriched, libErr) {
		t.Error("errors.Is(enriched, libErr) = false — original error should be in chain")
	}

	// The oops enrichment is present: %+v reveals a stack trace.
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

	// The oops enrichment survives the bridge wrapping.
	ctx := classified.ErrorContext()
	if ctx["domain"] != "validation" {
		t.Errorf("ErrorContext[domain] = %q, want validation", ctx["domain"])
	}
}

func TestPattern2_AutoWrap_TagOverridesDomain(t *testing.T) {
	// Domain is "validation" (→ Rejection), but tag "retryable" overrides → Transient.
	rich := oops.In("validation").
		Tags("retryable").
		Errorf("transient validation failure")

	classified := bridge.AutoWrap(rich)

	if family := errorfamily.Classify(classified); family != errorfamily.Transient {
		t.Errorf("tag 'retryable' should override validation domain → Transient, got %v", family)
	}
}

// --- Pattern 3: Explicit Wrap (application knows the family) ---

func TestPattern3_ExplicitWrap_AssignsConflict(t *testing.T) {
	store := &checkout.Store{ItemOutOfStock: "WIDGET-001"}
	order := &checkout.Order{
		ID: "order-1",
		Items: []checkout.LineItem{{SKU: "WIDGET-001", Qty: 5}},
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

	// The oops context is preserved through the bridge.
	ctx := classified.ErrorContext()
	if ctx["order_id"] != "order-1" {
		t.Errorf("ErrorContext[order_id] = %q, want order-1", ctx["order_id"])
	}
}

// --- HTTP Boundary: the full classify→enrich→handle flow ---

func TestHTTPBoundary_MissingID_Returns400Rejection(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := httptest.NewRequest(http.MethodGet, "/orders?id=", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (Rejection→400)", rec.Code, http.StatusBadRequest)
	}

	body := parseJSON(t, rec.Body.Bytes())
	if body["family"] != "rejection" {
		t.Errorf("family = %q, want rejection", body["family"])
	}
	if body["code"] != "order.missing_id" {
		t.Errorf("code = %q, want order.missing_id", body["code"])
	}
	// The raw error message must NEVER appear in the response.
	if strings.Contains(rec.Body.String(), "order ID is required") {
		t.Error("response leaked internal error message — should only contain safe fields")
	}
}

func TestHTTPBoundary_DBFailure_Returns503Transient(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := httptest.NewRequest(http.MethodGet, "/orders?id=order-42&fail=db", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d (Transient→503)", rec.Code, http.StatusServiceUnavailable)
	}

	body := parseJSON(t, rec.Body.Bytes())
	if body["family"] != "transient" {
		t.Errorf("family = %q, want transient", body["family"])
	}
	// HTTPHandler writes family/code/message only — it never exposes retryable
	// directly. The family "transient" is the retryable signal.
}

func TestHTTPBoundary_AutoWrapValidation_Returns400Rejection(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := httptest.NewRequest(http.MethodGet, "/orders?id=BADID", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (AutoWrap Rejection→400)", rec.Code, http.StatusBadRequest)
	}

	body := parseJSON(t, rec.Body.Bytes())
	if body["code"] != "order.id_format" {
		t.Errorf("code = %q, want order.id_format (from AutoWrap)", body["code"])
	}
}

func TestHTTPBoundary_ExplicitWrapConflict_Returns409(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := httptest.NewRequest(http.MethodGet, "/orders?id=order-42&fail=inv", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (Conflict→409)", rec.Code, http.StatusConflict)
	}

	body := parseJSON(t, rec.Body.Bytes())
	if body["family"] != "conflict" {
		t.Errorf("family = %q, want conflict", body["family"])
	}
}

func TestHTTPBoundary_Success_Returns200(t *testing.T) {
	store := &checkout.Store{}
	handler := errorfamily.HTTPHandler(handleGetOrder(store, testLogger()))

	req := httptest.NewRequest(http.MethodGet, "/orders?id=order-42", nil)
	req.Header.Set("X-Trace-ID", "trace-success")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// The success path returns application JSON (not from HTTPHandler).
	// Parse with any-typed values since amount_cents is a number.
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

	// The DB failure message is "database query exceeded deadline".
	// It must NEVER appear in the HTTP response.
	req := httptest.NewRequest(http.MethodGet, "/orders?id=order-42&fail=db", nil)
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

func parseJSON(t *testing.T, b []byte) map[string]string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("failed to parse JSON response %q: %v", string(b), err)
	}
	return m
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
