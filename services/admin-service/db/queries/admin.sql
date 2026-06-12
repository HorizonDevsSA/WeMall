-- name: CreateUserBan :one
INSERT INTO user_bans (user_id, reason, admin_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateSellerSuspension :one
INSERT INTO seller_suspensions (seller_id, reason, admin_id)
VALUES ($1, $2, $3)
RETURNING *;
