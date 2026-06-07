package worker

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	orderv1 "github.com/wemall/gen/order/v1"
	"github.com/wemall/review-service/internal/db"
	"github.com/wemall/review-service/internal/service"
)

type Worker struct {
	nc          *nats.Conn
	q           *db.Queries
	pool        *pgxpool.Pool
	reviewSvc   *service.ReviewService
	orderClient orderv1.OrderServiceClient
	logger      zerolog.Logger
}

func NewWorker(
	nc *nats.Conn,
	q *db.Queries,
	pool *pgxpool.Pool,
	reviewSvc *service.ReviewService,
	orderConn *grpc.ClientConn,
	logger zerolog.Logger,
) *Worker {
	return &Worker{
		nc:          nc,
		q:           q,
		pool:        pool,
		reviewSvc:   reviewSvc,
		orderClient: orderv1.NewOrderServiceClient(orderConn),
		logger:      logger,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	if w.nc == nil {
		w.logger.Warn().Msg("NATS connection is nil, worker bypassed subscription")
		return nil
	}

	// 1. Subscribe to order.delivered event
	_, err := w.nc.Subscribe("wemall.order.delivered", w.handleOrderDelivered)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to subscribe to wemall.order.delivered")
		return err
	}
	w.logger.Info().Msg("subscribed to wemall.order.delivered")

	// 2. Start periodic auto-review ticker
	go w.runAutoReviewTicker(ctx)

	return nil
}

func (w *Worker) handleOrderDelivered(msg *nats.Msg) {
	var event struct {
		OrderID     string `json:"order_id"`
		OrderNumber string `json:"order_number"`
		UserID      string `json:"user_id"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		w.logger.Error().Err(err).Msg("failed to unmarshal order delivered event")
		return
	}

	oid, err := uuid.Parse(event.OrderID)
	if err != nil {
		return
	}
	bid, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	ctx := context.Background()
	_, err = w.q.InsertOrderDelivery(ctx, db.InsertOrderDeliveryParams{
		OrderID:     oid,
		BuyerID:     bid,
		DeliveredAt: time.Now(),
	})
	if err != nil {
		w.logger.Error().Err(err).Str("order_id", event.OrderID).Msg("failed to insert order delivery to db")
		return
	}
	w.logger.Info().Str("order_id", event.OrderID).Msg("recorded order delivery for auto-review")
}

func (w *Worker) runAutoReviewTicker(ctx context.Context) {
	// Periodic check: default to every 10 seconds in development to make it testable,
	// but standard production interval could be 1 hour.
	interval := 10 * time.Second
	if envInterval := os.Getenv("AUTO_REVIEW_TICKER_INTERVAL"); envInterval != "" {
		if d, err := time.ParseDuration(envInterval); err == nil {
			interval = d
		}
	}

	// Threshold duration: default to 15 days, but can be set short (like 5 seconds) for dev testing!
	threshold := 15 * 24 * time.Hour
	if envThreshold := os.Getenv("AUTO_REVIEW_THRESHOLD_DURATION"); envThreshold != "" {
		if d, err := time.ParseDuration(envThreshold); err == nil {
			threshold = d
		}
	}

	w.logger.Info().
		Str("interval", interval.String()).
		Str("threshold", threshold.String()).
		Msg("starting auto-review scheduler background loop")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processAutoReviews(ctx, threshold)
		}
	}
}

func (w *Worker) processAutoReviews(ctx context.Context, threshold time.Duration) {
	cutoff := time.Now().Add(-threshold)
	deliveries, err := w.q.GetUnprocessedDeliveries(ctx, cutoff)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to get unprocessed deliveries")
		return
	}

	if len(deliveries) == 0 {
		return
	}

	w.logger.Info().Int("count", len(deliveries)).Msg("processing auto-reviews for delivered orders")

	for _, d := range deliveries {
		w.logger.Info().Str("order_id", d.OrderID.String()).Msg("triggering auto-reviews for order")

		// 1. Get order details from order-service
		order, err := w.orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{
			Id:     d.OrderID.String(),
			UserId: d.BuyerID.String(),
		})
		if err != nil {
			w.logger.Error().Err(err).
				Str("order_id", d.OrderID.String()).
				Msg("failed to fetch order details from order-service, skipping this cycle")
			continue
		}

		// 2. Iterate items and create default reviews
		success := true
		for _, item := range order.Items {
			variantID, err := uuid.Parse(item.VariantId)
			if err != nil {
				continue
			}

			// Check if review already exists
			_, err = w.q.GetReviewByOrderAndVariant(ctx, db.GetReviewByOrderAndVariantParams{
				OrderID:   d.OrderID,
				VariantID: variantID,
			})
			if err == nil {
				// Already reviewed by buyer, skip
				continue
			}

			if !errors.Is(err, pgx.ErrNoRows) {
				w.logger.Error().Err(err).Msg("database error checking existing review")
				success = false
				break
			}

			// Create system default positive review
			productID, _ := uuid.Parse(item.ProductId)
			sellerID, _ := uuid.Parse(item.SellerId)

			_, _, err = w.reviewSvc.CreateReview(ctx, service.CreateReviewInput{
				OrderID:           d.OrderID,
				BuyerID:           d.BuyerID,
				SellerID:          sellerID,
				ProductID:         productID,
				VariantID:         variantID,
				RatingDescription: 5,
				RatingService:     5,
				RatingDelivery:    5,
				Content:           "系统默认好评", // System default positive review
				IsAnonymous:       true,
				IsSystemGenerated: true,
			})
			if err != nil {
				w.logger.Error().Err(err).
					Str("order_id", d.OrderID.String()).
					Str("variant_id", item.VariantId).
					Msg("failed to create auto-review")
				success = false
				break
			}
		}

		if success {
			// Mark order delivery as processed
			err = w.q.MarkDeliveryProcessed(ctx, d.OrderID)
			if err != nil {
				w.logger.Error().Err(err).Str("order_id", d.OrderID.String()).Msg("failed to mark delivery as processed")
			} else {
				w.logger.Info().Str("order_id", d.OrderID.String()).Msg("marked order delivery as processed")
			}
		}
	}
}
