package worker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/wemall/ecocash-service/internal/db"
	"github.com/wemall/ecocash-service/internal/service"
)

// chargeResponseSLA is the confirmed window (§8) after which a PENDING
// transaction is considered stale and needs reconciliation via lookup.
const chargeResponseSLA = 60 * time.Second

// reconcilerInterval controls how often the reconciler polls for stale rows.
const reconcilerInterval = 5 * time.Minute

// Reconciler resolves PENDING transactions that have exceeded the charge SLA
// by calling LookupTransaction on each one. This covers webhook delivery
// failures and crashed in-flight requests.
type Reconciler struct {
	q      *db.Queries
	svc    *service.EcoCashService
	logger zerolog.Logger
}

func NewReconciler(q *db.Queries, svc *service.EcoCashService, logger zerolog.Logger) *Reconciler {
	return &Reconciler{q: q, svc: svc, logger: logger}
}

// Start runs the reconciler loop until ctx is cancelled.
func (r *Reconciler) Start(ctx context.Context) {
	r.logger.Info().Msg("reconciler started")
	ticker := time.NewTicker(reconcilerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info().Msg("reconciler stopped")
			return
		case <-ticker.C:
			r.run(ctx)
		}
	}
}

func (r *Reconciler) run(ctx context.Context) {
	threshold := time.Now().Add(-chargeResponseSLA)
	stale, err := r.q.ListStalePendingTransactions(ctx, threshold)
	if err != nil {
		r.logger.Error().Err(err).Msg("reconciler: failed to fetch stale pending transactions")
		return
	}

	if len(stale) == 0 {
		return
	}

	r.logger.Info().Int("count", len(stale)).Msg("reconciler: found stale pending transactions")

	for _, txn := range stale {
		txnCopy := txn
		r.resolveStaleTxn(ctx, &txnCopy)
	}
}

func (r *Reconciler) resolveStaleTxn(ctx context.Context, txn *db.PaymentTransaction) {
	resolveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	r.logger.Info().
		Str("txn_id", txn.ID.String()).
		Str("client_correlator", txn.ClientCorrelator).
		Msg("reconciler: looking up stale transaction")

	_, err := r.svc.LookupTransaction(resolveCtx, txn.OrderID, txn.ClientCorrelator)
	if err != nil {
		r.logger.Error().Err(err).
			Str("txn_id", txn.ID.String()).
			Msg("reconciler: lookup failed; will retry next cycle")
	}
}

// OutboxRelay publishes unpublished outbox events to NATS and marks them done.
// Runs on the same interval as the reconciler.
type OutboxRelay struct {
	q      *db.Queries
	nc     natsPublisher
	logger zerolog.Logger
}

// natsPublisher is a minimal interface so we can swap/mock in tests.
type natsPublisher interface {
	Publish(subject string, data []byte) error
}

func NewOutboxRelay(q *db.Queries, nc natsPublisher, logger zerolog.Logger) *OutboxRelay {
	return &OutboxRelay{q: q, nc: nc, logger: logger}
}

// Start runs the outbox relay loop until ctx is cancelled.
func (o *OutboxRelay) Start(ctx context.Context) {
	if o.nc == nil {
		o.logger.Warn().Msg("outbox relay: NATS connection is nil — relay loop disabled")
		return
	}

	o.logger.Info().Msg("outbox relay started")
	ticker := time.NewTicker(reconcilerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			o.logger.Info().Msg("outbox relay stopped")
			return
		case <-ticker.C:
			o.flush(ctx)
		}
	}
}

func (o *OutboxRelay) flush(ctx context.Context) {
	events, err := o.q.ListUnpublishedOutboxEvents(ctx)
	if err != nil {
		o.logger.Error().Err(err).Msg("outbox relay: failed to fetch unpublished events")
		return
	}

	for _, evt := range events {
		subject := "wemall." + evt.EventType // e.g. wemall.payment.succeeded
		if err := o.nc.Publish(subject, evt.Payload); err != nil {
			o.logger.Error().Err(err).
				Str("event_type", evt.EventType).
				Msg("outbox relay: failed to publish event")
			continue
		}

		if err := o.q.MarkOutboxEventPublished(ctx, evt.ID); err != nil {
			o.logger.Error().Err(err).
				Str("event_id", evt.ID.String()).
				Msg("outbox relay: failed to mark event as published")
		}
	}

	if len(events) > 0 {
		o.logger.Info().Int("published", len(events)).Msg("outbox relay: flushed events to NATS")
	}
}

// ensure uuid is used (referenced by db.PaymentTransaction.ID type)
var _ = uuid.UUID{}
