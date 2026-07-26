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

// handleGetOrder is the HTTP handler. It demonstrates the application layer:
// enriching library errors with oops and using the bridge where the
// application creates its own errors.
//
// It returns an error (not writes a response) on failure. errorfamily.HTTPHandler
// classifies that error and writes a safe JSON response — the raw err.Error()
// is NEVER sent to the client.
func handleGetOrder(store *checkout.Store, logger *slog.Logger) errorfamily.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		orderID := r.URL.Query().Get("id")
		traceID := r.Header.Get("X-Trace-ID")
		userID := r.Header.Get("X-User-ID")

		// --- Pattern 2: AUTOWRAP ---
		// Application-level validation creates the error with oops first.
		// bridge.AutoWrap infers the family from the "validation" domain.
		// This is the oops-first workflow: build rich, classify at the boundary.
		if strings.ToUpper(orderID) == "BADID" {
			rich := oops.In("validation").
				Tags("rejection").
				Code("order.id_format").
				With("received", orderID).
				With("trace_id", traceID).
				Errorf("order ID contains invalid characters")

			// AutoWrap reads the "validation" domain → Rejection, or the
			// "rejection" tag → Rejection. Both agree here.
			return bridge.AutoWrap(rich)
		}

		// Configure the store based on the ?fail= query param (demo only).
		applyFailMode(store, r.URL.Query().Get("fail"))

		// Call the library. The store returns an errorfamily-classified error.
		order, err := store.GetOrder(orderID)
		if err != nil {
			// --- Pattern 1: PASS-THROUGH ---
			// The library already classified this error (e.g. NewRejection or
			// NewTransient). We enrich it with oops — adding a stack trace,
			// trace ID, and request context. The classification survives
			// because errors.AsType traverses the chain and finds the
			// library's *Error implementing Classified.
			//
			// No bridge call needed here: the family is already embedded.
			enriched := oops.In("checkout").
				Code(errorfamily.Code(err)).
				Trace(traceID).
				With("user_id", userID).
				With("trace_id", traceID).
				With("method", r.Method).
				With("path", r.URL.Path).
				Wrap(err)

			// Log the FULL enriched error internally. %+v reveals the oops
			// stack trace and all context — for operators, never the client.
			logger.Error(fmt.Sprintf("%+v", enriched),
				"trace_id", traceID,
				"user_id", userID,
			)

			// Return for HTTPHandler. Classify() walks the chain past the
			// OopsError wrapper and finds the library's family.
			return enriched
		}

		// --- Pattern 3: EXPLICIT WRAP ---
		// The application creates an enrichment layer and KNOWS the family
		// (Conflict — inventory issues require the user to resolve).
		// bridge.Wrap combines the oops enrichment with an explicit family.
		if r.URL.Query().Get("fail") == "inv" {
			store.ItemOutOfStock = "WIDGET-001"

			if invErr := store.ReserveInventory(order); invErr != nil {
				rich := oops.In("checkout").
					Code("checkout.inventory_blocked").
					Trace(traceID).
					With("order_id", order.ID).
					With("user_id", userID).
					With("trace_id", traceID).
					Wrap(invErr)

				// The library classified this as Conflict, but we demonstrate
				// explicit bridge.Wrap: the application takes ownership of the
				// family decision at this composition boundary.
				logger.Error(fmt.Sprintf("%+v", rich),
					"trace_id", traceID,
				)

				return bridge.Wrap(rich, errorfamily.Conflict)
			}
		}

		// Success path — write a normal JSON response.
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		resp := map[string]any{
			"order_id":     order.ID,
			"user_id":      order.UserID,
			"amount_cents": order.AmountCents,
			"items":        order.Items,
			"trace_id":     traceID,
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			return err
		}

		return nil
	}
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
