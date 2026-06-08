package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/wemall/order-service/internal/db"
)

type Worker struct {
	nc     *nats.Conn
	q      *db.Queries
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

func NewWorker(nc *nats.Conn, q *db.Queries, pool *pgxpool.Pool, logger zerolog.Logger) *Worker {
	return &Worker{
		nc:     nc,
		q:      q,
		pool:   pool,
		logger: logger,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	if w.nc == nil {
		w.logger.Warn().Msg("NATS client is nil, order-service worker is disabled")
		return nil
	}

	// 1. Subscribe to payment completed
	_, err := w.nc.Subscribe("wemall.payment.completed", func(msg *nats.Msg) {
		w.handlePaymentCompleted(msg)
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to subscribe to wemall.payment.completed")
		return err
	}
	w.logger.Info().Msg("subscribed to wemall.payment.completed")

	// 2. Subscribe to payment failed
	_, err = w.nc.Subscribe("wemall.payment.failed", func(msg *nats.Msg) {
		w.handlePaymentFailed(msg)
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to subscribe to wemall.payment.failed")
		return err
	}
	w.logger.Info().Msg("subscribed to wemall.payment.failed")

	return nil
}

type paymentCompletedEvent struct {
	OrderID       string `json:"order_id"`
	PaymentID     string `json:"payment_id"`
	TransactionID string `json:"transaction_id"`
}

func (w *Worker) handlePaymentCompleted(msg *nats.Msg) {
	w.logger.Info().Msg("received payment completed event")

	var event paymentCompletedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.logger.Error().Err(err).Msg("failed to unmarshal payment completed event")
		return
	}

	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		w.logger.Error().Err(err).Msg("invalid order_id in event")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to begin transaction")
		return
	}
	defer tx.Rollback(ctx)

	qtx := w.q.WithTx(tx)

	// Update order status to confirmed
	err = qtx.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:      orderID,
		Column2: "confirmed",
	})
	if err != nil {
		w.logger.Error().Err(err).Msgf("failed to update order %s status to confirmed", event.OrderID)
		return
	}

	// Update order items status to confirmed
	err = qtx.UpdateOrderItemsStatus(ctx, db.UpdateOrderItemsStatusParams{
		OrderID: orderID,
		Column2: "confirmed",
	})
	if err != nil {
		w.logger.Error().Err(err).Msgf("failed to update order items status to confirmed for order: %s", event.OrderID)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		w.logger.Error().Err(err).Msg("failed to commit transaction")
		return
	}

	w.logger.Info().Msgf("successfully confirmed order %s after successful payment", event.OrderID)
}

type paymentFailedEvent struct {
	OrderID   string `json:"order_id"`
	PaymentID string `json:"payment_id"`
	Error     string `json:"error"`
}

func (w *Worker) handlePaymentFailed(msg *nats.Msg) {
	w.logger.Info().Msg("received payment failed event")

	var event paymentFailedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.logger.Error().Err(err).Msg("failed to unmarshal payment failed event")
		return
	}

	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		w.logger.Error().Err(err).Msg("invalid order_id in event")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to begin transaction")
		return
	}
	defer tx.Rollback(ctx)

	qtx := w.q.WithTx(tx)

	// Update order status to cancelled
	err = qtx.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:      orderID,
		Column2: "cancelled",
	})
	if err != nil {
		w.logger.Error().Err(err).Msgf("failed to update order %s status to cancelled", event.OrderID)
		return
	}

	// Update order items status to cancelled
	err = qtx.UpdateOrderItemsStatus(ctx, db.UpdateOrderItemsStatusParams{
		OrderID: orderID,
		Column2: "cancelled",
	})
	if err != nil {
		w.logger.Error().Err(err).Msgf("failed to update order items status to cancelled for order: %s", event.OrderID)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		w.logger.Error().Err(err).Msg("failed to commit transaction")
		return
	}

	w.logger.Info().Msgf("successfully cancelled order %s due to failed payment", event.OrderID)
}
