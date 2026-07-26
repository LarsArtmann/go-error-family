package checkout

import (
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-error-family/errorfamilytest"
)

func TestGetOrder_MissingID_IsRejection(t *testing.T) {
	store := &Store{}
	_, err := store.GetOrder("")

	errorfamilytest.AssertFamily(t, err, errorfamily.Rejection)
	errorfamilytest.AssertCode(t, err, "order.missing_id")
	errorfamilytest.AssertRetryable(t, err, false)
}

func TestGetOrder_DBUnreachable_IsTransient(t *testing.T) {
	store := &Store{DBUnreachable: true}
	_, err := store.GetOrder("order-99")

	errorfamilytest.AssertFamily(t, err, errorfamily.Transient)
	errorfamilytest.AssertCode(t, err, "order.db_timeout")
	errorfamilytest.AssertRetryable(t, err, true)
	errorfamilytest.AssertContext(t, err, "order_id", "order-99")
}

func TestGetOrder_DataCorrupted_IsCorruption(t *testing.T) {
	store := &Store{DataCorrupted: true}
	_, err := store.GetOrder("order-99")

	errorfamilytest.AssertFamily(t, err, errorfamily.Corruption)
	errorfamilytest.AssertCode(t, err, "order.data_corrupt")
	errorfamilytest.AssertRetryable(t, err, false)
}

func TestGetOrder_Success(t *testing.T) {
	store := &Store{}
	order, err := store.GetOrder("order-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.ID != "order-42" {
		t.Errorf("order ID = %q, want order-42", order.ID)
	}
}

func TestReserveInventory_OutOfStock_IsConflict(t *testing.T) {
	store := &Store{ItemOutOfStock: "WIDGET-001"}
	order := &Order{
		ID:    "order-1",
		Items: []LineItem{{SKU: "WIDGET-001", Qty: 5}},
	}

	err := store.ReserveInventory(order)

	errorfamilytest.AssertFamily(t, err, errorfamily.Conflict)
	errorfamilytest.AssertCode(t, err, "inventory.conflict")
	errorfamilytest.AssertRetryable(t, err, false)
	errorfamilytest.AssertContext(t, err, "sku", "WIDGET-001")
}

func TestReserveInventory_InStock_NoError(t *testing.T) {
	store := &Store{}
	order := &Order{
		ID:    "order-1",
		Items: []LineItem{{SKU: "GADGET-002", Qty: 1}},
	}

	if err := store.ReserveInventory(order); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChargeCard_Declined_IsRejection(t *testing.T) {
	store := &Store{PaymentDeclined: true}
	order := &Order{ID: "order-1", AmountCents: 9900}

	err := store.ChargeCard(order)

	errorfamilytest.AssertFamily(t, err, errorfamily.Rejection)
	errorfamilytest.AssertCode(t, err, "payment.declined")
	errorfamilytest.AssertContext(t, err, "order_id", "order-1")
}

func TestChargeCard_Approved_NoError(t *testing.T) {
	store := &Store{}
	order := &Order{ID: "order-1", AmountCents: 9900}

	if err := store.ChargeCard(order); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
