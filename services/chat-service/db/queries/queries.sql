-- name: CreateThread :one
INSERT INTO threads (
  type, title, buyer_id, seller_id, order_id, delivery_boy_id, courier_station_id, support_agent_id, participant_avatar
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: CreateMessage :one
INSERT INTO messages (
  thread_id, sender_id, type, content, media_url, reference_id, metadata
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
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

-- name: AddThreadMember :exec
INSERT INTO thread_members (
  thread_id, user_id, role
) VALUES (
  $1, $2, $3
)
ON CONFLICT (thread_id, user_id) DO NOTHING;

-- name: RemoveThreadMember :exec
DELETE FROM thread_members
WHERE thread_id = $1 AND user_id = $2;

-- name: IsThreadMember :one
SELECT EXISTS (
  SELECT 1 FROM thread_members
  WHERE thread_id = $1 AND user_id = $2
);

-- name: ListThreadMembers :many
SELECT user_id FROM thread_members
WHERE thread_id = $1;

-- name: ListThreadsForUser :many
SELECT DISTINCT t.* FROM threads t
LEFT JOIN thread_members tm ON t.id = tm.thread_id
WHERE t.buyer_id = $1 
   OR t.seller_id = $1 
   OR t.delivery_boy_id = $1 
   OR t.courier_station_id = $1 
   OR t.support_agent_id = $1 
   OR tm.user_id = $1
ORDER BY t.updated_at DESC;

-- name: GetSystemThread :one
SELECT * FROM threads
WHERE type = 'THREAD_TYPE_SYSTEM'
LIMIT 1;

