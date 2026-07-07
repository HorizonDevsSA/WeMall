-- ── payment_transactions ──────────────────────────────────────────────────────

-- name: CreateTransaction :one
INSERT INTO payment_transactions (
    order_id, client_correlator, tran_type,
    end_user_id_masked, amount_cents, currency,
    status, raw_request
) VALUES ($1, $2, $3, $4, $5, $6, 'PENDING', $7)
RETURNING *;

-- name: GetTransactionByID :one
SELECT * FROM payment_transactions WHERE id = $1;

-- name: GetTransactionByCorrelator :one
SELECT * FROM payment_transactions WHERE client_correlator = $1;

-- name: ListTransactionsByOrder :many
SELECT * FROM payment_transactions
WHERE order_id = $1
ORDER BY created_at DESC;

-- name: UpdateTransactionStatus :one
UPDATE payment_transactions
SET
    status                 = $2,
    ecocash_status_code    = $3,
    ecocash_status_msg     = $4,
    ecocash_transaction_id = $5,
    reference_code         = COALESCE(NULLIF($6, ''), reference_code),
    raw_response           = $7,
    updated_at             = NOW()
WHERE id = $1
RETURNING *;

-- name: ListStalePendingTransactions :many
-- Returns PENDING transactions older than the given threshold (60s SLA).
-- Used by the reconciler worker to poll EcoCash for resolution.
SELECT * FROM payment_transactions
WHERE status = 'PENDING'
  AND created_at < $1
ORDER BY created_at ASC
LIMIT 100;

-- ── refund_transactions ───────────────────────────────────────────────────────

-- name: CreateRefundTransaction :one
INSERT INTO refund_transactions (
    original_txn_id, client_correlator, tran_type,
    amount_cents, status, raw_request
) VALUES ($1, $2, $3, $4, 'PENDING', $5)
RETURNING *;

-- name: GetRefundByID :one
SELECT * FROM refund_transactions WHERE id = $1;

-- name: GetRefundByCorrelator :one
SELECT * FROM refund_transactions WHERE client_correlator = $1;

-- name: SumRefundedAmountForTxn :one
SELECT COALESCE(SUM(amount_cents), 0)::BIGINT AS total_refunded
FROM refund_transactions
WHERE original_txn_id = $1 AND status = 'SUCCESS';

-- name: UpdateRefundStatus :one
UPDATE refund_transactions
SET
    status              = $2,
    ecocash_status_code = $3,
    ecocash_status_msg  = $4,
    raw_response        = $5
WHERE id = $1
RETURNING *;

-- ── payouts ───────────────────────────────────────────────────────────────────

-- name: CreatePayout :one
INSERT INTO payouts (seller_id, amount_cents, currency, status, method)
VALUES ($1, $2, $3, 'QUEUED', 'manual_bulk')
RETURNING *;

-- name: GetPayoutByID :one
SELECT * FROM payouts WHERE id = $1;

-- name: UpdatePayoutStatus :one
UPDATE payouts
SET
    status       = $2,
    provider_ref = $3,
    updated_at   = NOW()
WHERE id = $1
RETURNING *;

-- name: ListQueuedPayouts :many
SELECT * FROM payouts WHERE status = 'QUEUED' ORDER BY created_at ASC;

-- ── outbox_events ─────────────────────────────────────────────────────────────

-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (aggregate_id, event_type, payload)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListUnpublishedOutboxEvents :many
SELECT * FROM outbox_events
WHERE published = false
ORDER BY created_at ASC
LIMIT 50;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events SET published = true WHERE id = $1;

-- name: GetExistingPendingCharge :one
-- Returns the most recent PENDING MER charge for an order (if any).
SELECT * FROM payment_transactions
WHERE order_id = $1 AND tran_type = $2 AND status = 'PENDING'
ORDER BY created_at DESC
LIMIT 1;
