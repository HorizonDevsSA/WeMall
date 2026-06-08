package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	paymentv1 "github.com/wemall/gen/payment/v1"
	"github.com/wemall/payment-service/internal/db"
	"github.com/wemall/payment-service/internal/service"
)

type Worker struct {
	nc     *nats.Conn
	q      *db.Queries
	svc    *service.PaymentService
	logger zerolog.Logger
}

func NewWorker(nc *nats.Conn, q *db.Queries, svc *service.PaymentService, logger zerolog.Logger) *Worker {
	return &Worker{
		nc:     nc,
		q:      q,
		svc:    svc,
		logger: logger,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	if w.nc == nil {
		w.logger.Warn().Msg("NATS client is nil, background worker is disabled")
		return nil
	}

	// 1. Subscribe to order created event
	_, err := w.nc.Subscribe("wemall.order.created", func(msg *nats.Msg) {
		w.handleOrderCreated(msg)
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to subscribe to wemall.order.created")
		return err
	}
	w.logger.Info().Msg("subscribed to wemall.order.created")

	// 2. Subscribe to order cancelled event
	_, err = w.nc.Subscribe("wemall.order.cancelled", func(msg *nats.Msg) {
		w.handleOrderCancelled(msg)
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to subscribe to wemall.order.cancelled")
		return err
	}
	w.logger.Info().Msg("subscribed to wemall.order.cancelled")

	return nil
}

type orderCreatedEvent struct {
	OrderID     string  `json:"order_id"`
	OrderNumber string  `json:"order_number"`
	UserID      string  `json:"user_id"`
	Total       float64 `json:"total"`
	Currency    string  `json:"currency"`
}

func (w *Worker) handleOrderCreated(msg *nats.Msg) {
	w.logger.Info().Msg("received order created event")

	var event orderCreatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.logger.Error().Err(err).Msg("failed to unmarshal order created event")
		return
	}

	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		w.logger.Error().Err(err).Msg("invalid order_id in event")
		return
	}

	userID, err := uuid.Parse(event.UserID)
	if err != nil {
		w.logger.Error().Err(err).Msg("invalid user_id in event")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Default to Google Pay (primary provider)
	payment, _, err := w.svc.CreatePayment(
		ctx,
		orderID,
		userID,
		event.Total,
		event.Currency,
		paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY,
	)
	if err != nil {
		w.logger.Error().Err(err).Msgf("failed to automatically create pending payment for order: %s", event.OrderID)
		return
	}

	w.logger.Info().Msgf("automatically initialized pending payment %s for order %s", payment.ID, event.OrderID)
}

type orderCancelledEvent struct {
	OrderID     string `json:"order_id"`
	OrderNumber string `json:"order_number"`
	UserID      string `json:"user_id"`
}

func (w *Worker) handleOrderCancelled(msg *nats.Msg) {
	w.logger.Info().Msg("received order cancelled event")

	var event orderCancelledEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.logger.Error().Err(err).Msg("failed to unmarshal order cancelled event")
		return
	}

	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		w.logger.Error().Err(err).Msg("invalid order_id in event")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get payment
	payment, err := w.svc.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || statusMatchesNotFound(err) {
			w.logger.Warn().Msgf("no payment record found for cancelled order: %s", event.OrderID)
			return
		}
		w.logger.Error().Err(err).Msg("failed to fetch payment by order_id")
		return
	}

	// If pending, mark as failed (cancelled)
	if payment.Status == "pending" {
		_, err = w.svc.RefundPayment(ctx, payment.ID, payment.Amount) // or just mark as failed
		if err != nil {
			// If not completed, try setting it directly to failed
			_, err = w.q.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
				ID:     payment.ID,
				Status: "failed",
			})
			if err != nil {
				w.logger.Error().Err(err).Msgf("failed to update payment status to failed for payment %s", payment.ID)
				return
			}
		}
		w.logger.Info().Msgf("marked payment %s as failed/cancelled due to order cancellation", payment.ID)
	}
}

func statusMatchesNotFound(err error) bool {
	return err != nil && (err.Error() == "payment not found for order" || stringsContains(err.Error(), "no rows"))
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || stringContainsCheck(s, sub))
}

func stringContainsCheck(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
