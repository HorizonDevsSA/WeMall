-- name: UpsertDeviceToken :one
INSERT INTO user_device_tokens (user_id, token, platform, device_name, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (token) DO UPDATE
SET user_id = EXCLUDED.user_id,
    platform = EXCLUDED.platform,
    device_name = EXCLUDED.device_name,
    updated_at = NOW()
RETURNING *;

-- name: DeleteDeviceToken :exec
DELETE FROM user_device_tokens
WHERE user_id = $1 AND token = $2;

-- name: GetDeviceTokensByUser :many
SELECT * FROM user_device_tokens
WHERE user_id = $1;

-- name: GetDeviceTokensByUsers :many
SELECT * FROM user_device_tokens
WHERE user_id = ANY($1::uuid[]);

-- name: GetNotificationPreferences :many
SELECT * FROM user_notification_preferences
WHERE user_id = $1;

-- name: GetNotificationPreference :one
SELECT * FROM user_notification_preferences
WHERE user_id = $1 AND category = $2;

-- name: UpsertNotificationPreference :one
INSERT INTO user_notification_preferences (user_id, category, email_enabled, push_enabled, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (user_id, category) DO UPDATE
SET email_enabled = EXCLUDED.email_enabled,
    push_enabled = EXCLUDED.push_enabled,
    updated_at = NOW()
RETURNING *;

-- name: CreateNotificationLog :one
INSERT INTO notification_logs (user_id, category, channel, recipient, title, content, status, retry_count)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateNotificationLogStatus :exec
UPDATE notification_logs
SET status = $2,
    retry_count = $3,
    error_message = $4,
    sent_at = $5,
    updated_at = NOW()
WHERE id = $1;

-- name: ListNotificationLogs :many
SELECT * FROM notification_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreatePushNotification :one
INSERT INTO push_notifications (user_id, token, title, body, payload, is_read)
VALUES ($1, $2, $3, $4, $5, FALSE)
RETURNING *;

-- name: ListPushNotifications :many
SELECT * FROM push_notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: MarkPushNotificationRead :exec
UPDATE push_notifications
SET is_read = TRUE,
    read_at = NOW()
WHERE id = $1 AND user_id = $2;
