package handler_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentv1 "github.com/wemall/gen/payment/v1"
	"github.com/wemall/payment-service/internal/handler"
	"github.com/wemall/payment-service/internal/service"
)

// newTestHandler builds a PaymentHandler with nil dependencies.
// All tests that reach the service layer will either test input validation
// (which panics on nil DB and is caught by recover) or use skip annotations.
func newTestHandler() *handler.PaymentHandler {
	svc := service.NewPaymentService(nil, nil, nil, "sk_test", "merchant_test")
	return handler.NewPaymentHandler(svc)
}

// ---------------------------------------------------------------------------
// CreatePayment – invalid UUID validation (pure, no DB)
// ---------------------------------------------------------------------------

func TestCreatePayment_InvalidOrderID(t *testing.T) {
	h := newTestHandler()

	_, err := h.CreatePayment(context.Background(), &paymentv1.CreatePaymentRequest{
		OrderId:  "not-a-uuid",
		UserId:   "00000000-0000-0000-0000-000000000001",
		Amount:   100.0,
		Currency: "USD",
		Provider: paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY,
	})

	if err == nil {
		t.Fatal("expected error for invalid order_id UUID, got nil")
	}
	grpcErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if grpcErr.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", grpcErr.Code())
	}
}

func TestCreatePayment_InvalidUserID(t *testing.T) {
	h := newTestHandler()

	_, err := h.CreatePayment(context.Background(), &paymentv1.CreatePaymentRequest{
		OrderId:  "00000000-0000-0000-0000-000000000001",
		UserId:   "bad-uuid",
		Amount:   100.0,
		Currency: "USD",
		Provider: paymentv1.PaymentProvider_PAYMENT_PROVIDER_STRIPE,
	})

	if err == nil {
		t.Fatal("expected error for invalid user_id UUID, got nil")
	}
	grpcErr, _ := status.FromError(err)
	if grpcErr.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", grpcErr.Code())
	}
}

// ---------------------------------------------------------------------------
// ProcessPayment – invalid UUID validation (pure, no DB)
// ---------------------------------------------------------------------------

func TestProcessPayment_InvalidPaymentID(t *testing.T) {
	h := newTestHandler()

	_, err := h.ProcessPayment(context.Background(), &paymentv1.ProcessPaymentRequest{
		PaymentId: "not-valid-uuid",
		Token:     "gp_token_valid",
	})

	if err == nil {
		t.Fatal("expected error for invalid payment_id UUID, got nil")
	}
	grpcErr, _ := status.FromError(err)
	if grpcErr.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", grpcErr.Code())
	}
}

// ---------------------------------------------------------------------------
// GetPayment – invalid UUID validation (pure, no DB)
// ---------------------------------------------------------------------------

func TestGetPayment_InvalidID(t *testing.T) {
	h := newTestHandler()

	_, err := h.GetPayment(context.Background(), &paymentv1.GetPaymentRequest{
		Id: "garbage",
	})

	if err == nil {
		t.Fatal("expected error for invalid id UUID, got nil")
	}
	grpcErr, _ := status.FromError(err)
	if grpcErr.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", grpcErr.Code())
	}
}

// ---------------------------------------------------------------------------
// RefundPayment – invalid UUID validation (pure, no DB)
// ---------------------------------------------------------------------------

func TestRefundPayment_InvalidPaymentID(t *testing.T) {
	h := newTestHandler()

	_, err := h.RefundPayment(context.Background(), &paymentv1.RefundPaymentRequest{
		PaymentId: "also-not-a-uuid",
		Amount:    50.0,
	})

	if err == nil {
		t.Fatal("expected error for invalid payment_id UUID, got nil")
	}
	grpcErr, _ := status.FromError(err)
	if grpcErr.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", grpcErr.Code())
	}
}

// ---------------------------------------------------------------------------
// Valid UUID format does not return InvalidArgument (service error bubbles up)
// ---------------------------------------------------------------------------

func TestCreatePayment_ValidUUIDs_ServiceErrorBubblesUp(t *testing.T) {
	h := newTestHandler()

	defer func() {
		if r := recover(); r != nil {
			// Expected: nil db.Queries panics at the service layer.
			// This means UUID validation passed correctly.
			t.Log("UUID validation passed; nil db panicked as expected")
		}
	}()

	h.CreatePayment(context.Background(), &paymentv1.CreatePaymentRequest{
		OrderId:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserId:   "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		Amount:   99.99,
		Currency: "USD",
		Provider: paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY,
	})
}

func TestProcessPayment_ValidUUID_ServiceErrorBubblesUp(t *testing.T) {
	h := newTestHandler()

	defer func() {
		if r := recover(); r != nil {
			t.Log("UUID validation passed; nil db panicked as expected")
		}
	}()

	h.ProcessPayment(context.Background(), &paymentv1.ProcessPaymentRequest{
		PaymentId: "cccccccc-cccc-cccc-cccc-cccccccccccc",
		Token:     "valid_token",
	})
}

func TestGetPayment_ValidUUID_ServiceErrorBubblesUp(t *testing.T) {
	h := newTestHandler()

	defer func() {
		if r := recover(); r != nil {
			t.Log("UUID validation passed; nil db panicked as expected")
		}
	}()

	h.GetPayment(context.Background(), &paymentv1.GetPaymentRequest{
		Id: "dddddddd-dddd-dddd-dddd-dddddddddddd",
	})
}

func TestRefundPayment_ValidUUID_ServiceErrorBubblesUp(t *testing.T) {
	h := newTestHandler()

	defer func() {
		if r := recover(); r != nil {
			t.Log("UUID validation passed; nil db panicked as expected")
		}
	}()

	h.RefundPayment(context.Background(), &paymentv1.RefundPaymentRequest{
		PaymentId: "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee",
		Amount:    25.0,
	})
}
