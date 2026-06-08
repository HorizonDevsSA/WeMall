package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentv1 "github.com/wemall/gen/payment/v1"
	"github.com/wemall/payment-service/internal/db"
	"github.com/wemall/payment-service/internal/service"
)

// ---------------------------------------------------------------------------
// Mock Implementations for sqlc DBTX and pgx.Row
// ---------------------------------------------------------------------------

type mockRow struct {
	scanFunc func(dest ...any) error
}

func (r mockRow) Scan(dest ...any) error {
	if r.scanFunc != nil {
		return r.scanFunc(dest...)
	}
	return nil
}

type mockDBTX struct {
	queryRowFunc func(ctx context.Context, query string, args ...any) pgx.Row
}

func (m mockDBTX) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (m mockDBTX) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m mockDBTX) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	return mockRow{}
}

// ---------------------------------------------------------------------------
// Unit tests using the service's exported surface
// ---------------------------------------------------------------------------

// TestCreatePayment_Success verifies that a valid CreatePayment call inserts
// into the database using queries and returns clientSecret appropriately.
func TestCreatePayment_Success(t *testing.T) {
	orderID := uuid.New()
	userID := uuid.New()
	amount := 99.99
	currency := "USD"
	provider := paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY

	expectedPaymentID := uuid.New()

	dbMock := mockDBTX{
		queryRowFunc: func(ctx context.Context, query string, args ...any) pgx.Row {
			if len(args) != 5 {
				t.Errorf("expected 5 arguments, got %d", len(args))
			}
			if args[0] != orderID {
				t.Errorf("expected orderID %v, got %v", orderID, args[0])
			}
			if args[1] != userID {
				t.Errorf("expected userID %v, got %v", userID, args[1])
			}
			if args[2] != amount {
				t.Errorf("expected amount %f, got %v", amount, args[2])
			}
			if args[3] != currency {
				t.Errorf("expected currency %s, got %v", currency, args[3])
			}
			if args[4] != "google_pay" {
				t.Errorf("expected provider google_pay, got %v", args[4])
			}

			return mockRow{
				scanFunc: func(dest ...any) error {
					if len(dest) != 10 {
						t.Fatalf("expected 10 destination pointers, got %d", len(dest))
					}
					*dest[0].(*uuid.UUID) = expectedPaymentID
					*dest[1].(*uuid.UUID) = orderID
					*dest[2].(*uuid.UUID) = userID
					*dest[3].(*float64) = amount
					*dest[4].(*string) = currency
					*dest[5].(*string) = "google_pay"
					*dest[6].(*string) = "pending"
					*dest[7].(**string) = nil
					*dest[8].(*time.Time) = time.Now()
					*dest[9].(*time.Time) = time.Now()
					return nil
				},
			}
		},
	}

	queries := db.New(dbMock)
	svc := service.NewPaymentService(queries, nil, nil, "stripe_key", "gp_merchant")

	payment, clientSecret, err := svc.CreatePayment(context.Background(), orderID, userID, amount, currency, provider)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if payment.ID != expectedPaymentID {
		t.Errorf("expected payment ID %v, got %v", expectedPaymentID, payment.ID)
	}
	if payment.OrderID != orderID {
		t.Errorf("expected order ID %v, got %v", orderID, payment.OrderID)
	}
	if payment.Amount != amount {
		t.Errorf("expected amount %f, got %f", amount, payment.Amount)
	}
	if payment.Status != "pending" {
		t.Errorf("expected status pending, got %s", payment.Status)
	}
	if !strings.Contains(clientSecret, "gp_merchant") {
		t.Errorf("expected clientSecret to contain gp_merchant, got %s", clientSecret)
	}
}

// TestCreatePayment_InvalidAmount verifies that a non-positive amount returns
// an error before any DB call is made.
func TestCreatePayment_InvalidAmount(t *testing.T) {
	svc := service.NewPaymentService(nil, nil, nil, "stripe_key", "gp_merchant")

	_, _, err := svc.CreatePayment(
		context.Background(),
		uuid.New(),
		uuid.New(),
		-5.0,
		"USD",
		paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY,
	)

	if err == nil {
		t.Fatal("expected error for negative amount, got nil")
	}
}

// TestCreatePayment_ZeroAmount verifies that zero amount is also rejected.
func TestCreatePayment_ZeroAmount(t *testing.T) {
	svc := service.NewPaymentService(nil, nil, nil, "stripe_key", "gp_merchant")

	_, _, err := svc.CreatePayment(
		context.Background(),
		uuid.New(),
		uuid.New(),
		0,
		"USD",
		paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY,
	)

	if err == nil {
		t.Fatal("expected error for zero amount, got nil")
	}
}

// TestCreatePayment_InvalidProvider verifies unspecified provider is rejected.
func TestCreatePayment_InvalidProvider(t *testing.T) {
	svc := service.NewPaymentService(nil, nil, nil, "stripe_key", "gp_merchant")

	_, _, err := svc.CreatePayment(
		context.Background(),
		uuid.New(),
		uuid.New(),
		100.0,
		"USD",
		paymentv1.PaymentProvider_PAYMENT_PROVIDER_UNSPECIFIED,
	)

	if err == nil {
		t.Fatal("expected error for unspecified provider, got nil")
	}
}

// TestCreatePayment_DBError verifies DB errors are propagated correctly.
func TestCreatePayment_DBError(t *testing.T) {
	dbErr := errors.New("database connection failed")
	dbMock := mockDBTX{
		queryRowFunc: func(ctx context.Context, query string, args ...any) pgx.Row {
			return mockRow{
				scanFunc: func(dest ...any) error {
					return dbErr
				},
			}
		},
	}

	queries := db.New(dbMock)
	svc := service.NewPaymentService(queries, nil, nil, "stripe_key", "gp_merchant")

	_, _, err := svc.CreatePayment(
		context.Background(),
		uuid.New(),
		uuid.New(),
		50.0,
		"USD",
		paymentv1.PaymentProvider_PAYMENT_PROVIDER_STRIPE,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.Internal {
		t.Errorf("expected Internal code, got %s", st.Code())
	}
}

// TestGetPayment_NotFound verifies that a not-found DB error is wrapped into a
// descriptive service error when the record does not exist.
func TestGetPayment_NotFound(t *testing.T) {
	dbMock := mockDBTX{
		queryRowFunc: func(ctx context.Context, query string, args ...any) pgx.Row {
			return mockRow{
				scanFunc: func(dest ...any) error {
					return pgx.ErrNoRows
				},
			}
		},
	}

	queries := db.New(dbMock)
	svc := service.NewPaymentService(queries, nil, nil, "stripe_key", "gp_merchant")

	_, err := svc.GetPayment(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "payment not found") {
		t.Errorf("expected 'payment not found' error, got: %v", err)
	}
}

// TestCreatePayment_DefaultCurrency verifies that an empty currency string
// defaults to "USD" when hitting DB.
func TestCreatePayment_DefaultCurrency(t *testing.T) {
	orderID := uuid.New()
	userID := uuid.New()
	amount := 50.0

	dbMock := mockDBTX{
		queryRowFunc: func(ctx context.Context, query string, args ...any) pgx.Row {
			if args[3] != "USD" {
				t.Errorf("expected empty currency to default to USD, got %v", args[3])
			}
			return mockRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*uuid.UUID) = uuid.New()
					*dest[1].(*uuid.UUID) = orderID
					*dest[2].(*uuid.UUID) = userID
					*dest[3].(*float64) = amount
					*dest[4].(*string) = "USD"
					*dest[5].(*string) = "stripe"
					*dest[6].(*string) = "pending"
					*dest[7].(**string) = nil
					*dest[8].(*time.Time) = time.Now()
					*dest[9].(*time.Time) = time.Now()
					return nil
				},
			}
		},
	}

	queries := db.New(dbMock)
	svc := service.NewPaymentService(queries, nil, nil, "stripe_key", "gp_merchant")

	_, _, err := svc.CreatePayment(context.Background(), orderID, userID, amount, "", paymentv1.PaymentProvider_PAYMENT_PROVIDER_STRIPE)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Integration Test placeholders requiring a live pool/db
// ---------------------------------------------------------------------------

func TestProcessPayment_EmptyToken_ExpectedFailure(t *testing.T) {
	t.Skip("integration test – requires a live PostgreSQL database; run with -tags integration")
}

func TestProcessPayment_ValidGooglePayToken_ExpectedSuccess(t *testing.T) {
	t.Skip("integration test – requires a live PostgreSQL database; run with -tags integration")
}

func TestRefundPayment_NonCompletedPayment_ExpectedError(t *testing.T) {
	t.Skip("integration test – requires a live PostgreSQL database; run with -tags integration")
}

func TestRefundPayment_ZeroAmount_RejectDocumented(t *testing.T) {
	t.Skip("integration test – requires a live PostgreSQL database; run with -tags integration")
}

func TestRefundPayment_ExceedsOriginalAmount_RejectDocumented(t *testing.T) {
	t.Skip("integration test – requires a live PostgreSQL database; run with -tags integration")
}

// ---------------------------------------------------------------------------
// Provider & Status Enum verification tests
// ---------------------------------------------------------------------------

func TestProviderEnum_GooglePayValue(t *testing.T) {
	if paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY == paymentv1.PaymentProvider_PAYMENT_PROVIDER_UNSPECIFIED {
		t.Fatal("GOOGLE_PAY enum should not equal UNSPECIFIED")
	}
}

func TestProviderEnum_StripeValue(t *testing.T) {
	if paymentv1.PaymentProvider_PAYMENT_PROVIDER_STRIPE == paymentv1.PaymentProvider_PAYMENT_PROVIDER_UNSPECIFIED {
		t.Fatal("STRIPE enum should not equal UNSPECIFIED")
	}
}

func TestProviderEnum_GooglePayNotStripe(t *testing.T) {
	if paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY == paymentv1.PaymentProvider_PAYMENT_PROVIDER_STRIPE {
		t.Fatal("GOOGLE_PAY and STRIPE enums must be distinct")
	}
}

func TestStatusEnum_PendingNotCompleted(t *testing.T) {
	if paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING == paymentv1.PaymentStatus_PAYMENT_STATUS_COMPLETED {
		t.Fatal("PENDING and COMPLETED statuses must be distinct")
	}
}

func TestStatusEnum_FailedNotRefunded(t *testing.T) {
	if paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED == paymentv1.PaymentStatus_PAYMENT_STATUS_REFUNDED {
		t.Fatal("FAILED and REFUNDED statuses must be distinct")
	}
}
