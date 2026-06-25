package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/wemall/seller-service/internal/db"
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
		w.logger.Warn().Msg("NATS client is nil, background worker is disabled")
		return nil
	}

	_, err := w.nc.Subscribe("wemall.order.confirmed", func(msg *nats.Msg) {
		w.handleOrderConfirmed(msg)
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to subscribe to wemall.order.confirmed")
		return err
	}
	w.logger.Info().Msg("subscribed to wemall.order.confirmed")

	_, err = w.nc.Subscribe("wemall.order.item.updated", func(msg *nats.Msg) {
		w.handleOrderItemUpdated(msg)
	})
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to subscribe to wemall.order.item.updated")
		return err
	}
	w.logger.Info().Msg("subscribed to wemall.order.item.updated")

	return nil
}

type confirmedEventItem struct {
	OrderItemID string  `json:"order_item_id"`
	SellerID    string  `json:"seller_id"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    int32   `json:"quantity"`
}

type orderConfirmedEvent struct {
	OrderID string               `json:"order_id"`
	Items   []confirmedEventItem `json:"items"`
}

func (w *Worker) handleOrderConfirmed(msg *nats.Msg) {
	w.logger.Info().Msg("received order confirmed event")

	var event orderConfirmedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.logger.Error().Err(err).Msg("failed to unmarshal order confirmed event")
		return
	}

	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		w.logger.Error().Err(err).Msg("invalid order_id in event")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to begin transaction")
		return
	}
	defer tx.Rollback(ctx)

	qtx := w.q.WithTx(tx)

	for _, item := range event.Items {
		sellerID, err := uuid.Parse(item.SellerID)
		if err != nil {
			w.logger.Error().Err(err).Msgf("invalid seller_id %s in event", item.SellerID)
			continue
		}

		orderItemID, err := uuid.Parse(item.OrderItemID)
		if err != nil {
			w.logger.Error().Err(err).Msgf("invalid order_item_id %s in event", item.OrderItemID)
			continue
		}

		// Fetch seller to get their commission rate
		seller, err := qtx.GetSellerByID(ctx, sellerID)
		if err != nil {
			w.logger.Error().Err(err).Msgf("failed to get seller %s to compute commission", item.SellerID)
			continue
		}

		gross := item.UnitPrice * float64(item.Quantity)
		fee := gross * seller.CommissionRate
		net := gross - fee

		_, err = qtx.CreateEarningEntry(ctx, db.CreateEarningEntryParams{
			SellerID:      sellerID,
			OrderID:       orderID,
			OrderItemID:   orderItemID,
			GrossAmount:   gross,
			CommissionFee: fee,
			NetAmount:     net,
			Status:        "escrowed",
		})
		if err != nil {
			w.logger.Error().Err(err).Msgf("failed to write earning entry for item %s", item.OrderItemID)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		w.logger.Error().Err(err).Msg("failed to commit transaction")
		return
	}

	w.logger.Info().Msgf("successfully processed escrow earnings ledger entries for order %s", event.OrderID)
}

type orderItemUpdatedEvent struct {
	OrderID  string `json:"order_id"`
	SellerID string `json:"seller_id"`
	Status   string `json:"status"`
}

func (w *Worker) handleOrderItemUpdated(msg *nats.Msg) {
	w.logger.Info().Msg("received order item updated event")

	var event orderItemUpdatedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.logger.Error().Err(err).Msg("failed to unmarshal order item updated event")
		return
	}

	orderID, err := uuid.Parse(event.OrderID)
	if err != nil {
		w.logger.Error().Err(err).Msg("invalid order_id in event")
		return
	}

	sellerID, err := uuid.Parse(event.SellerID)
	if err != nil {
		w.logger.Error().Err(err).Msg("invalid seller_id in event")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statusTransformed := ""
	if event.Status == "delivered" {
		statusTransformed = "earned"
	} else if event.Status == "cancelled" || event.Status == "refunded" {
		statusTransformed = "refunded"
	} else {
		w.logger.Info().Msgf("ignoring status transition %s for order %s and seller %s", event.Status, event.OrderID, event.SellerID)
		return
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to begin transaction")
		return
	}
	defer tx.Rollback(ctx)

	qtx := w.q.WithTx(tx)

	// Update status of matching ledger entries
	err = qtx.UpdateEarningStatusByOrderAndSeller(ctx, db.UpdateEarningStatusByOrderAndSellerParams{
		OrderID:  orderID,
		SellerID: sellerID,
		Status:   statusTransformed,
	})

	if err != nil {
		w.logger.Error().Err(err).Msgf("failed to update earning status for order %s and seller %s", event.OrderID, event.SellerID)
		return
	}

	// If status is 'earned' (delivered), we also increment the seller's total sales count!
	if statusTransformed == "earned" {
		_, err = qtx.IncrementSellerTotalSales(ctx, db.IncrementSellerTotalSalesParams{
			ID:         sellerID,
			TotalSales: 1,
		})
		if err != nil {
			w.logger.Error().Err(err).Msgf("failed to increment total sales for seller %s", event.SellerID)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		w.logger.Error().Err(err).Msg("failed to commit transaction")
		return
	}

	w.logger.Info().Msgf("successfully transitioned ledger status to %s for order %s and seller %s", statusTransformed, event.OrderID, event.SellerID)
}
