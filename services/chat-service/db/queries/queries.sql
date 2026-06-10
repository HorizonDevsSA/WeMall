-- name: CreateThread :one
INSERT INTO threads (
  type, title, buyer_id, seller_id, order_id
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: CreateMessage :one
INSERT INTO messages (
  thread_id, sender_id, type, content, media_url, reference_id
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetThread :one
SELECT * FROM threads
WHERE id = $1 LIMIT 1;

-- name: ListDirectThreadsForBuyer :many
SELECT * FROM threads
WHERE buyer_id = $1 AND type = 'THREAD_TYPE_DIRECT'
ORDER BY updated_at DESC;

-- name: ListThreadsForSeller :many
SELECT * FROM threads
WHERE seller_id = $1
ORDER BY updated_at DESC;

-- name: ListBroadcastThreadsForSellers :many
SELECT * FROM threads
WHERE seller_id = ANY($1::varchar[]) AND type = 'THREAD_TYPE_BROADCAST'
ORDER BY updated_at DESC;

-- name: GetBroadcastThreadForSeller :one
SELECT * FROM threads
WHERE seller_id = $1 AND type = 'THREAD_TYPE_BROADCAST'
LIMIT 1;

-- name: ListMessages :many
SELECT * FROM messages
WHERE thread_id = $1
ORDER BY created_at ASC;

-- name: UpdateThreadTimestamp :exec
UPDATE threads
SET updated_at = NOW()
WHERE id = $1;
