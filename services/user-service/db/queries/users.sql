-- name: CreateUser :one
INSERT INTO users (email, phone, password_hash, full_name, avatar_url, role, auth_provider, is_verified, google_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: GetUserByPhone :one
SELECT * FROM users WHERE phone = $1 AND deleted_at IS NULL;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = $1 AND deleted_at IS NULL;

-- name: GetUsersByIDs :many
SELECT * FROM users WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL;

-- name: UpdateUser :one
UPDATE users SET
    full_name  = COALESCE(NULLIF(@full_name::text, ''), full_name),
    avatar_url = COALESCE(NULLIF(@avatar_url::text, ''), avatar_url),
    updated_at = NOW()
WHERE id = @id AND deleted_at IS NULL
RETURNING *;

-- name: VerifyUser :one
UPDATE users SET is_verified = TRUE, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpsertGoogleUser :one
INSERT INTO users (email, full_name, avatar_url, role, auth_provider, is_verified, google_id)
VALUES ($1, $2, $3, 'buyer', 'google', TRUE, $4)
ON CONFLICT (google_id) DO UPDATE SET
    full_name  = EXCLUDED.full_name,
    avatar_url = EXCLUDED.avatar_url,
    updated_at = NOW()
RETURNING *;

-- name: UpsertPhoneUser :one
INSERT INTO users (phone, full_name, role, auth_provider, is_verified)
VALUES ($1, $2, 'buyer', 'phone', TRUE)
ON CONFLICT (phone) DO UPDATE SET updated_at = NOW()
RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW();

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE token_hash = $1;

-- name: RevokeAllUserTokens :exec
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: CreateOTP :one
INSERT INTO phone_otps (phone, otp_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetLatestOTP :one
SELECT * FROM phone_otps
WHERE phone = $1 AND used = FALSE AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkOTPUsed :exec
UPDATE phone_otps SET used = TRUE WHERE id = $1;

-- name: IncrementOTPAttempts :exec
UPDATE phone_otps SET attempts = attempts + 1 WHERE id = $1;

-- name: CreateAddress :one
INSERT INTO addresses (user_id, label, full_name, phone, address_line1, address_line2, city, state, postal_code, country, is_default)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListAddressesByUser :many
SELECT * FROM addresses WHERE user_id = $1 ORDER BY is_default DESC, created_at DESC;

-- name: UnsetDefaultAddresses :exec
UPDATE addresses SET is_default = FALSE WHERE user_id = $1;

-- name: DeleteAddress :exec
DELETE FROM addresses WHERE id = $1 AND user_id = $2;
