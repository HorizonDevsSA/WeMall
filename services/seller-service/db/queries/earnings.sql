-- name: CreateEarningEntry :one
INSERT INTO seller_earnings (seller_id, order_id, order_item_id, gross_amount, commission_fee, net_amount, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateEarningStatusByOrderItem :one
UPDATE seller_earnings
SET status = $2, settled_at = CASE WHEN $2 = 'earned' THEN NOW() ELSE settled_at END
WHERE order_item_id = $1
RETURNING *;

-- name: UpdateEarningStatusByOrderAndSeller :exec
UPDATE seller_earnings
SET status = $3, settled_at = CASE WHEN $3 = 'earned' THEN NOW() ELSE settled_at END
WHERE order_id = $1 AND seller_id = $2;


-- name: GetSellerBalances :one
SELECT 
    COALESCE(SUM(CASE WHEN status = 'escrowed' THEN net_amount ELSE 0 END), 0)::numeric AS escrowed_balance,
    COALESCE(SUM(CASE WHEN status = 'earned' THEN net_amount ELSE 0 END), 0)::numeric AS withdrawable_balance
FROM seller_earnings
WHERE seller_id = $1;

-- name: ListEarningsLedger :many
SELECT * FROM seller_earnings
WHERE seller_id = $1 AND ($4::text = '' OR status = $4)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountEarningsLedger :one
SELECT COUNT(*)::int AS total FROM seller_earnings
WHERE seller_id = $1 AND ($2::text = '' OR status = $2);

-- name: BulkLinkPayoutToEarnings :exec
UPDATE seller_earnings
SET status = 'payout_released', payout_id = $2
WHERE seller_id = $1 AND status = 'earned';

-- name: GetEarnedSumBySeller :one
SELECT 
    COALESCE(SUM(gross_amount), 0)::numeric AS gross_sum,
    COALESCE(SUM(commission_fee), 0)::numeric AS fee_sum,
    COALESCE(SUM(net_amount), 0)::numeric AS net_sum
FROM seller_earnings
WHERE seller_id = $1 AND status = 'earned';

-- name: GetEarnedEarnings :many
SELECT * FROM seller_earnings
WHERE seller_id = $1 AND status = 'earned'
ORDER BY created_at ASC;

-- name: UpdateEarningAmountsAndStatus :one
UPDATE seller_earnings
SET status = $2,
    payout_id = $3,
    gross_amount = $4,
    commission_fee = $5,
    net_amount = $6
WHERE id = $1
RETURNING *;

-- name: CreateEarningEntryWithDetails :one
INSERT INTO seller_earnings (seller_id, order_id, order_item_id, gross_amount, commission_fee, net_amount, status, created_at, settled_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;


