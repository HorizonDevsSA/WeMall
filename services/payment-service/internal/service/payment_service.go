package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	paymentv1 "github.com/wemall/gen/payment/v1"
	werr "github.com/wemall/pkg/errors"
	"github.com/wemall/payment-service/internal/db"
)

type PaymentService struct {
	q                   *db.Queries
	pool                *pgxpool.Pool
	nc                  *nats.Conn
	stripeSecretKey     string
	googlePayMerchantID string
}

func NewPaymentService(q *db.Queries, pool *pgxpool.Pool, nc *nats.Conn, stripeSecretKey, googlePayMerchantID string) *PaymentService {
	return &PaymentService{
		q:                   q,
		pool:                pool,
		nc:                  nc,
		stripeSecretKey:     stripeSecretKey,
		googlePayMerchantID: googlePayMerchantID,
	}
}

func (s *PaymentService) CreatePayment(ctx context.Context, orderID, userID uuid.UUID, amount float64, currency string, provider paymentv1.PaymentProvider) (*db.Payment, string, error) {
	if amount <= 0 {
		return nil, "", werr.InvalidArgument("amount must be positive")
	}
	if currency == "" {
		currency = "USD"
	}

	providerStr := ""
	switch provider {
	case paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY:
		providerStr = "google_pay"
	case paymentv1.PaymentProvider_PAYMENT_PROVIDER_STRIPE:
		providerStr = "stripe"
	default:
		return nil, "", werr.InvalidArgument("invalid payment provider")
	}

	// Create payment in database
	payment, err := s.q.CreatePayment(ctx, db.CreatePaymentParams{
		OrderID:  orderID,
		UserID:   userID,
		Amount:   amount,
		Currency: currency,
		Provider: providerStr,
	})
	if err != nil {
		return nil, "", werr.Internal(err)
	}

	// Generate client secrets/config based on provider
	clientSecret := ""
	if providerStr == "stripe" {
		// In a real implementation: stripe.PaymentIntent.New(...)
		clientSecret = fmt.Sprintf("pi_%s_secret_%s", payment.ID.String()[:8], uuid.New().String()[:8])
	} else if providerStr == "google_pay" {
		// Return Google Pay merchant initialization parameters
		clientSecret = fmt.Sprintf("merchant:%s:payment:%s", s.googlePayMerchantID, payment.ID.String())
	}

	return &payment, clientSecret, nil
}

func (s *PaymentService) ProcessPayment(ctx context.Context, paymentID uuid.UUID, token string) (*db.Payment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	payment, err := qtx.GetPayment(ctx, paymentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("payment not found")
		}
		return nil, werr.Internal(err)
	}

	if payment.Status != "pending" {
		return nil, werr.InvalidArgument(fmt.Sprintf("cannot process payment in status: %s", payment.Status))
	}

	// Simulate payment authorization processing
	success := true
	txnID := ""
	errMsg := ""

	// Process according to the selected provider
	if payment.Provider == "google_pay" {
		// Primary Provider Flow
		if token == "" || strings.Contains(token, "fail") {
			success = false
			errMsg = "Google Pay token validation failed or was declined"
		} else {
			txnID = fmt.Sprintf("gp_txn_%s", uuid.New().String()[:12])
		}
	} else {
		// Secondary Provider (Stripe) Flow
		if token == "" || strings.Contains(token, "fail") {
			success = false
			errMsg = "Stripe charge authentication failed"
		} else {
			txnID = fmt.Sprintf("ch_str_%s", uuid.New().String()[:12])
		}
	}

	var updated db.Payment
	if success {
		updated, err = qtx.UpdatePaymentTransaction(ctx, db.UpdatePaymentTransactionParams{
			ID:            payment.ID,
			Status:        "completed",
			TransactionID: &txnID,
		})
		if err != nil {
			return nil, werr.Internal(err)
		}
	} else {
		updated, err = qtx.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
			ID:     payment.ID,
			Status: "failed",
		})
		if err != nil {
			return nil, werr.Internal(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, werr.Internal(err)
	}

	// Publish async status update to NATS
	if s.nc != nil {
		if success {
			event := map[string]interface{}{
				"order_id":       payment.OrderID.String(),
				"payment_id":     payment.ID.String(),
				"transaction_id": txnID,
				"amount":         payment.Amount,
				"currency":       payment.Currency,
			}
			eb, _ := json.Marshal(event)
			_ = s.nc.Publish("wemall.payment.completed", eb)
		} else {
			event := map[string]interface{}{
				"order_id":   payment.OrderID.String(),
				"payment_id": payment.ID.String(),
				"error":      errMsg,
			}
			eb, _ := json.Marshal(event)
			_ = s.nc.Publish("wemall.payment.failed", eb)
		}
	}

	return &updated, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, id uuid.UUID) (*db.Payment, error) {
	payment, err := s.q.GetPayment(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("payment not found")
		}
		return nil, werr.Internal(err)
	}
	return &payment, nil
}

func (s *PaymentService) GetPaymentByOrderID(ctx context.Context, orderID uuid.UUID) (*db.Payment, error) {
	payment, err := s.q.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("payment not found for order")
		}
		return nil, werr.Internal(err)
	}
	return &payment, nil
}

func (s *PaymentService) RefundPayment(ctx context.Context, paymentID uuid.UUID, amount float64) (*db.Payment, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	payment, err := qtx.GetPayment(ctx, paymentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("payment not found")
		}
		return nil, werr.Internal(err)
	}

	if payment.Status != "completed" {
		return nil, werr.InvalidArgument(fmt.Sprintf("cannot refund payment in status: %s", payment.Status))
	}

	if amount <= 0 || amount > payment.Amount {
		return nil, werr.InvalidArgument(fmt.Sprintf("refund amount must be between 0 and %f", payment.Amount))
	}

	// Execute mock refund
	updated, err := qtx.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
		ID:     payment.ID,
		Status: "refunded",
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, werr.Internal(err)
	}

	// Publish refund event to NATS
	if s.nc != nil {
		event := map[string]interface{}{
			"order_id":   payment.OrderID.String(),
			"payment_id": payment.ID.String(),
			"amount":     amount,
			"status":     "refunded",
		}
		eb, _ := json.Marshal(event)
		_ = s.nc.Publish("wemall.payment.refunded", eb)
	}

	return &updated, nil
}
