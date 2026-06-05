-- name: UpsertStock :one
INSERT INTO variant_stock (variant_id, quantity)
VALUES ($1, $2)
ON CONFLICT (variant_id) DO UPDATE SET
    quantity = EXCLUDED.quantity,
    updated_at = NOW()
RETURNING variant_id, quantity, created_at, updated_at;

-- name: GetStock :one
SELECT variant_id, quantity, created_at, updated_at
FROM variant_stock
WHERE variant_id = $1;

-- name: GetStockBatch :many
SELECT variant_id, quantity, created_at, updated_at
FROM variant_stock
WHERE variant_id = ANY($1::uuid[]);
