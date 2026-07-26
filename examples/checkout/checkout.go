// Package checkout is a domain LIBRARY for order checkout.
//
// It demonstrates the "libraries classify" principle from go-error-family's
// architecture: a reusable library imports ONLY go-error-family and returns
// classified errors. It never imports samber/oops or any other enrichment
// library — that choice belongs to the application, not the library.
//
// The library knows its own domain contract:
//   - unknown order ID      → Rejection  (caller's fault: bad input)
//   - database unreachable  → Transient  (system's fault: retry)
//   - item out of stock     → Conflict   (user must resolve: change order)
//   - payment data corrupt  → Corruption (source of truth damaged: escalate)
//
// These classifications travel with the error up the call stack. The
// application layer (see examples/cmd/bridge) enriches them with stack traces,
// trace IDs, and request context using oops + the bridge.
package checkout

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// Order represents a checkout order in the domain.
type Order struct {
	ID          string
	UserID      string
	AmountCents int
	Items       []LineItem
}

// LineItem is a single item in an order.
type LineItem struct {
	SKU        string
	Qty        int
	PriceCents int
}

// Store is the library's data-access boundary.
// The Config fields control failure modes so tests are deterministic
// without real database or payment-gateway dependencies.
type Store struct {
	// DBUnreachable simulates a transient database outage.
	DBUnreachable bool
	// ItemOutOfStock simulates a stock conflict for a specific SKU.
	ItemOutOfStock string
	// PaymentDeclined simulates a card issuer rejecting the charge.
	PaymentDeclined bool
	// DataCorrupted simulates an unparseable record in the source of truth.
	DataCorrupted bool
}

// GetOrder retrieves an order by ID.
//
// Returns classified errors:
//   - empty ID           → Rejection (order.missing_id)
//   - DB unreachable     → Transient (order.db_timeout)
//   - corrupt record     → Corruption (order.data_corrupt)
func (s *Store) GetOrder(id string) (*Order, error) {
	if id == "" {
		return nil, errorfamily.NewRejection("order.missing_id", "order ID is required").
			WithContext("suggestion", "Provide the order ID as a query parameter or path segment.")
	}

	if s.DBUnreachable {
		return nil, errorfamily.NewTransient("order.db_timeout", "database query exceeded deadline").
			WithContext("order_id", id)
	}

	if s.DataCorrupted {
		return nil, errorfamily.NewCorruption("order.data_corrupt", "order record is unparseable").
			WithContext("order_id", id)
	}

	return &Order{
		ID:          id,
		UserID:      "user-42",
		AmountCents: 9900,
		Items: []LineItem{
			{SKU: "WIDGET-001", Qty: 2, PriceCents: 4950},
		},
	}, nil
}

// ReserveInventory attempts to reserve stock for an order.
//
// Returns classified errors:
//   - out of stock       → Conflict (inventory.conflict) — the user must change the order
//   - warehouse offline  → Transient (inventory.warehouse_timeout) — retry later
func (s *Store) ReserveInventory(order *Order) error {
	for _, item := range order.Items {
		if item.SKU == s.ItemOutOfStock {
			return errorfamily.NewConflict("inventory.conflict", "item is out of stock").
				WithContext("sku", item.SKU).
				WithContextAny("requested_qty", item.Qty)
		}
	}

	return nil
}

// ChargeCard processes the payment for an order.
//
// Returns classified errors:
//   - card declined      → Rejection (payment.declined) — caller's fault
//   - gateway timeout    → Transient (payment.gateway_timeout) — retry
func (s *Store) ChargeCard(order *Order) error {
	if s.PaymentDeclined {
		return errorfamily.NewRejection("payment.declined", "card issuer declined the charge").
			WithContext("order_id", order.ID).
			WithContextAny("amount_cents", order.AmountCents)
	}

	return nil
}
