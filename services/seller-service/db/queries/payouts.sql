-- name: CreatePayout :one
INSERT INTO seller_payouts (seller_id, amount, currency, status)
VALUES ($1, $2, $3, 'pending')
RETURNING *;

-- name: GetPayoutByID :one
SELECT * FROM seller_payouts WHERE id = $1;

-- name: GetPayoutByIDForSeller :one
SELECT * FROM seller_payouts WHERE id = $1 AND seller_id = $2;

-- name: ListPayoutsBySeller :many
SELECT * FROM seller_payouts
WHERE seller_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPayoutsBySeller :one
SELECT COUNT(*)::int AS total FROM seller_payouts WHERE seller_id = $1;
