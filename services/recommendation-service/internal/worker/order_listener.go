package worker

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/wemall/recommendation-service/internal/db"
)

type OrderCreatedEvent struct {
	OrderID    string   `json:"order_id"`
	UserID     string   `json:"user_id"`
	ProductIDs []string `json:"product_ids"`
}

type OrderListener struct {
	nc      *nats.Conn
	queries *db.Queries
}

func NewOrderListener(nc *nats.Conn, queries *db.Queries) *OrderListener {
	return &OrderListener{nc: nc, queries: queries}
}

func (l *OrderListener) Start() {
	if l.nc == nil {
		log.Warn().Msg("NATS connection is nil, order listener disabled")
		return
	}

	_, err := l.nc.Subscribe("wemall.order.created", func(msg *nats.Msg) {
		var event OrderCreatedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Error().Err(err).Msg("failed to unmarshal order created event")
			return
		}

		// Calculate permutations
		ctx := context.Background()
		for i := 0; i < len(event.ProductIDs); i++ {
			for j := 0; j < len(event.ProductIDs); j++ {
				if i != j {
					// We only process distinct pairs
					// In bidirectional co-purchasing, if A and B are bought together:
					// Upsert (A, B) and (B, A). The nested loop does this naturally!
					pA := event.ProductIDs[i]
					pB := event.ProductIDs[j]

					// Minor cleanup: ensure they are not identical strings due to bad data
					if strings.TrimSpace(pA) == "" || strings.TrimSpace(pB) == "" || pA == pB {
						continue
					}

					err := l.queries.UpsertCoPurchase(ctx, db.UpsertCoPurchaseParams{
						ProductAID: pA,
						ProductBID: pB,
					})
					if err != nil {
						log.Error().Err(err).Str("pA", pA).Str("pB", pB).Msg("failed to upsert co-purchase")
					}
				}
			}
		}

		log.Info().Str("order_id", event.OrderID).Int("items", len(event.ProductIDs)).Msg("processed order co-purchases")
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to subscribe to wemall.order.created")
	} else {
		log.Info().Msg("subscribed to wemall.order.created")
	}
}
