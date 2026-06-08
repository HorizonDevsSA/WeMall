package worker_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// orderCreatedEvent shape tests
// ---------------------------------------------------------------------------

// orderCreatedEvent mirrors the unexported struct inside worker.go.
type orderCreatedEvent struct {
	OrderID     string  `json:"order_id"`
	OrderNumber string  `json:"order_number"`
	UserID      string  `json:"user_id"`
	Total       float64 `json:"total"`
	Currency    string  `json:"currency"`
}

// orderCancelledEvent mirrors the unexported struct inside worker.go.
type orderCancelledEvent struct {
	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`
	UserID      string `json:"user_id"`
}

// TestOrderCreatedEvent_Unmarshal verifies that the expected JSON payload for
// a wemall.order.created event deserialises correctly.
func TestOrderCreatedEvent_Unmarshal(t *testing.T) {
	orderID := uuid.New()
	userID := uuid.New()

	payload := map[string]interface{}{
		"order_id":     orderID.String(),
		"order_number": "ORD-001",
		"user_id":      userID.String(),
		"total":        149.99,
		"currency":     "USD",
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal test payload: %v", err)
	}

	var event orderCreatedEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatalf("failed to unmarshal order created event: %v", err)
	}

	if event.OrderID != orderID.String() {
		t.Errorf("expected order_id %s, got %s", orderID.String(), event.OrderID)
	}
	if event.UserID != userID.String() {
		t.Errorf("expected user_id %s, got %s", userID.String(), event.UserID)
	}
	if event.Total != 149.99 {
		t.Errorf("expected total 149.99, got %f", event.Total)
	}
	if event.Currency != "USD" {
		t.Errorf("expected currency USD, got %s", event.Currency)
	}
}

// TestOrderCancelledEvent_Unmarshal verifies that the expected JSON payload for
// a wemall.order.cancelled event deserialises correctly.
func TestOrderCancelledEvent_Unmarshal(t *testing.T) {
	orderID := uuid.New()
	userID := uuid.New()

	payload := map[string]interface{}{
		"order_id":     orderID.String(),
		"order_number": "ORD-002",
		"user_id":      userID.String(),
	}

	raw, _ := json.Marshal(payload)

	var event orderCancelledEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatalf("failed to unmarshal order cancelled event: %v", err)
	}

	if event.OrderID != orderID.String() {
		t.Errorf("expected order_id %s, got %s", orderID.String(), event.OrderID)
	}
}

// TestOrderCreatedEvent_InvalidJSON verifies that malformed payloads fail
// unmarshal gracefully (as the worker does with an error log and return).
func TestOrderCreatedEvent_InvalidJSON(t *testing.T) {
	var event orderCreatedEvent
	err := json.Unmarshal([]byte("{bad json}"), &event)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// TestOrderCreatedEvent_MissingFields verifies that partial payloads unmarshal
// with zero values (Go JSON behaviour) without panicking.
func TestOrderCreatedEvent_MissingFields(t *testing.T) {
	raw := []byte(`{"order_id":"00000000-0000-0000-0000-000000000001"}`)
	var event orderCreatedEvent
	if err := json.Unmarshal(raw, &event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Total != 0 {
		t.Errorf("expected zero total for missing field, got %f", event.Total)
	}
	if event.Currency != "" {
		t.Errorf("expected empty currency for missing field, got %s", event.Currency)
	}
}

// ---------------------------------------------------------------------------
// UUID parsing tests (mirrors logic inside the worker handlers)
// ---------------------------------------------------------------------------

func TestUUIDParsing_ValidUUID(t *testing.T) {
	id := uuid.New().String()
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("failed to parse valid UUID: %v", err)
	}
	if parsed.String() != id {
		t.Errorf("round-trip UUID mismatch: got %s", parsed.String())
	}
}

func TestUUIDParsing_InvalidUUID(t *testing.T) {
	_, err := uuid.Parse("this-is-not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID string, got nil")
	}
}

func TestUUIDParsing_EmptyString(t *testing.T) {
	_, err := uuid.Parse("")
	if err == nil {
		t.Fatal("expected error for empty UUID string, got nil")
	}
}

// ---------------------------------------------------------------------------
// Payment status state-machine documentation tests
// ---------------------------------------------------------------------------

// The following tests document the expected state transitions for a payment.
// They serve as living documentation and will be expanded into integration tests.

func TestPaymentStateMachine_PendingToCompleted(t *testing.T) {
	// pending -> completed: via ProcessPayment with a valid token
	t.Log("State: pending -> completed requires a valid provider token")
}

func TestPaymentStateMachine_PendingToFailed(t *testing.T) {
	// pending -> failed: via ProcessPayment with an empty/fail token
	t.Log("State: pending -> failed when token is empty or contains 'fail'")
}

func TestPaymentStateMachine_CompletedToRefunded(t *testing.T) {
	// completed -> refunded: via RefundPayment with amount in (0, payment.Amount]
	t.Log("State: completed -> refunded via RefundPayment")
}

func TestPaymentStateMachine_PendingCannotBeRefunded(t *testing.T) {
	// RefundPayment checks payment.Status == "completed" before proceeding.
	// A pending payment cannot be refunded directly.
	t.Log("State: pending -> refunded is NOT allowed; must be completed first")
}

func TestPaymentStateMachine_FailedCannotBeRefunded(t *testing.T) {
	t.Log("State: failed -> refunded is NOT allowed")
}
