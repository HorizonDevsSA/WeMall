-- name: FollowStore :exec
INSERT INTO store_follows (user_id, seller_id)
VALUES ($1, $2)
ON CONFLICT (user_id, seller_id) DO NOTHING;

-- name: UnfollowStore :exec
DELETE FROM store_follows
WHERE user_id = $1 AND seller_id = $2;

-- name: IsFollowingStore :one
SELECT EXISTS(
    SELECT 1 FROM store_follows
    WHERE user_id = $1 AND seller_id = $2
) AS is_following;

-- name: ListFollowedStores :many
SELECT s.*
FROM sellers s
INNER JOIN store_follows f ON f.seller_id = s.id
WHERE f.user_id = $1
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFollowedStores :one
SELECT COUNT(*) FROM store_follows WHERE user_id = $1;

-- name: CountStoreFollowers :one
SELECT COUNT(*) FROM store_follows WHERE seller_id = $1;

-- name: ListStoreFollowers :many
SELECT user_id FROM store_follows WHERE seller_id = $1;

