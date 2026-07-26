// Command bridge is the reference implementation for the oops + bridge stack.
//
// It demonstrates the full classify→enrich→handle flow using a small HTTP
// checkout service. The architecture has two layers:
//
//   - LIBRARY layer (examples/checkout): imports ONLY go-error-family and
//     returns classified errors. It never imports oops.
//   - APPLICATION layer (this file): imports oops for enrichment (stack traces,
//     trace IDs, request context) and the bridge to combine enrichment with
//     classification at the HTTP boundary.
//
// Three patterns are demonstrated:
//
//  1. PASS-THROUGH: Library errors classified with error-family flow through
//     oops enrichment unchanged. Classify() finds the family through the chain.
//     No bridge call needed — the library already classified.
//
//  2. AUTOWRAP: Application-created errors built with oops are classified by
//     bridge.AutoWrap, which infers the family from oops tags and domain.
//
//  3. EXPLICIT WRAP: The application knows the family and assigns it directly
//     via bridge.Wrap, combining oops enrichment with an explicit family.
//
// Run: go run ./examples/cmd/bridge
//
//	curl 'http://localhost:8090/orders?id='                  # 400 Rejection (pass-through)
//	curl 'http://localhost:8090/orders?id=order-42'          # 200 success
//	curl 'http://localhost:8090/orders?id=order-42&fail=db'  # 503 Transient (pass-through)
//	curl 'http://localhost:8090/orders?id=badid'             # 400 Rejection (autowrap)
//	curl 'http://localhost:8090/orders?id=order-42&fail=inv' # 409 Conflict (explicit wrap)
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/bridge"
	"github.com/larsartmann/go-error-family/examples/checkout"
	"github.com/samber/oops"
)

const (
	headerTraceID = "X-Trace-Id"
	headerUserID  = "X-User-Id"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	store := &checkout.Store{}

	mux := http.NewServeMux()
	mux.Handle("/orders", errorfamily.HTTPHandler(handleGetOrder(store, logger)))

	addr := ":8090"

	fmt.Printf("Bridge reference server on %s\n", addr)
	fmt.Println("curl 'http://localhost:8090/orders?id=order-42'")
	fmt.Println("curl 'http://localhost:8090/orders?id='              # 400 Rejection")
	fmt.Println("curl 'http://localhost:8090/orders?id=order-42&fail=db'  # 503 Transient")
	fmt.Println("curl 'http://localhost:8090/orders?id=BADID'         # 400 Rejection (autowrap)")
	fmt.Println("curl 'http://localhost:8090/orders?id=order-42&fail=inv' # 409 Conflict (wrap)")

	if err := http.ListenAndServe(addr, mux); err != nil {
		os.Exit(errorfamily.HandleError(err))
	}
}

// handleGetOrder is the HTTP handler demonstrating all three bridge patterns.
// Returns an error on failure — errorfamily.HTTPHandler classifies it and
// writes a safe JSON response. The raw err.Error() is NEVER sent to the client.
func handleGetOrder(store *checkout.Store, logger *slog.Logger) errorfamily.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		orderID := r.URL.Query().Get("id")
		traceID := r.Header.Get(headerTraceID)
		userID := r.Header.Get(headerUserID)

		// Pattern 2: AUTOWRAP — application validation via bridge.AutoWrap.
		if err := validateOrderID(orderID, traceID); err != nil {
			return err
		}

		applyFailMode(store, r.URL.Query().Get("fail"))

		order, err := store.GetOrder(orderID)
		if err != nil {
			// Pattern 1: PASS-THROUGH — library classification survives oops enrichment.
			return enrichLibraryError(err, traceID, userID, r, logger)
		}

		// Pattern 3: EXPLICIT WRAP — application knows the family (Conflict).
		if r.URL.Query().Get("fail") == "inv" {
			if err := checkInventory(store, order, traceID, userID, logger); err != nil {
				return err
			}
		}

		return writeOrderResponse(w, order, traceID)
	}
}

// validateOrderID demonstrates Pattern 2: AUTOWRAP.
// The application creates the error with oops, then bridge.AutoWrap infers
// the family from the "validation" domain and "rejection" tag.
func validateOrderID(orderID, traceID string) error {
	if strings.ToUpper(orderID) != "BADID" {
		return nil
	}

	rich := oops.In("validation").
		Tags("rejection").
		Code("order.id_format").
		With("received", orderID).
		With("trace_id", traceID).
		Errorf("order ID contains invalid characters")

	return bridge.AutoWrap(rich)
}

// enrichLibraryError demonstrates Pattern 1: PASS-THROUGH.
// The library already classified the error. We wrap it with oops for
// enrichment (stack trace, trace ID, request context). The classification
// survives because errors.AsType traverses the chain. No bridge call needed.
func enrichLibraryError(
	err error,
	traceID, userID string,
	r *http.Request,
	logger *slog.Logger,
) error {
	enriched := oops.In("checkout").
		Code(errorfamily.Code(err)).
		Trace(traceID).
		With("user_id", userID).
		With("trace_id", traceID).
		With("method", r.Method).
		With("path", r.URL.Path).
		Wrap(err)

	logger.Error(fmt.Sprintf("%+v", enriched),
		"trace_id", traceID,
		"user_id", userID,
	)

	return enriched
}

// checkInventory demonstrates Pattern 3: EXPLICIT WRAP.
// The application knows this is a Conflict (inventory requires user resolution).
// bridge.Wrap combines the oops enrichment with an explicit family.
func checkInventory(
	store *checkout.Store,
	order *checkout.Order,
	traceID, userID string,
	logger *slog.Logger,
) error {
	store.ItemOutOfStock = checkout.DefaultWidgetSKU

	invErr := store.ReserveInventory(order)
	if invErr == nil {
		return nil
	}

	rich := oops.In("checkout").
		Code("checkout.inventory_blocked").
		Trace(traceID).
		With("order_id", order.ID).
		With("user_id", userID).
		With("trace_id", traceID).
		Wrap(invErr)

	logger.Error(fmt.Sprintf("%+v", rich), "trace_id", traceID)

	return bridge.Wrap(rich, errorfamily.Conflict)
}

// writeOrderResponse encodes a successful order response as JSON.
func writeOrderResponse(w http.ResponseWriter, order *checkout.Order, traceID string) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	resp := map[string]any{
		"order_id":     order.ID,
		"user_id":      order.UserID,
		"amount_cents": order.AmountCents,
		"items":        order.Items,
		"trace_id":     traceID,
	}

	return json.NewEncoder(w).Encode(resp)
}

// applyFailMode sets the store's failure simulation based on the demo ?fail= param.
func applyFailMode(store *checkout.Store, mode string) {
	store.DBUnreachable = false
	store.DataCorrupted = false
	store.ItemOutOfStock = ""

	switch mode {
	case "db":
		store.DBUnreachable = true
	case "corrupt":
		store.DataCorrupted = true
	}
}
