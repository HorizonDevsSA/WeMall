-- name: CreateDispute :one
INSERT INTO disputes (
  order_id, buyer_id, seller_id, reason, status
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetDispute :one
SELECT * FROM disputes
WHERE id = $1 LIMIT 1;

-- name: UpdateDisputeStatus :one
UPDATE disputes
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListDisputesByBuyer :many
SELECT * FROM disputes
WHERE buyer_id = $1
ORDER BY created_at DESC;

-- name: ListDisputesBySeller :many
SELECT * FROM disputes
WHERE seller_id = $1
ORDER BY created_at DESC;

-- name: ListAllDisputes :many
SELECT * FROM disputes
ORDER BY created_at DESC;

-- name: CreateDisputeMessage :one
INSERT INTO dispute_messages (
  dispute_id, sender_id, content, evidence_urls
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: ListDisputeMessages :many
SELECT * FROM dispute_messages
WHERE dispute_id = $1
ORDER BY created_at ASC;
