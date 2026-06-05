-- name: CreateMediaAsset :one
INSERT INTO media_assets (
    owner_id,
    service_scope,
    original_name,
    mime_type,
    size_bytes,
    raw_s3_key,
    is_private,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetMediaAsset :one
SELECT * FROM media_assets
WHERE id = $1;

-- name: BatchGetMediaAssets :many
SELECT * FROM media_assets
WHERE id = ANY($1::uuid[]);

-- name: UpdateMediaStatus :one
UPDATE media_assets
SET status = $2,
    error_message = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateMediaVariants :one
UPDATE media_assets
SET status = 'completed',
    variants = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListMediaAssets :many
SELECT * FROM media_assets
WHERE owner_id = $1
  AND (CAST(sqlc.narg('service_scope') AS varchar) IS NULL OR service_scope = sqlc.narg('service_scope'))
  AND (CAST(sqlc.narg('mime_type') AS varchar) IS NULL OR mime_type LIKE sqlc.narg('mime_type'))
  AND (CAST(sqlc.narg('status') AS media_status) IS NULL OR status = sqlc.narg('status'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountMediaAssets :one
SELECT COUNT(*) FROM media_assets
WHERE owner_id = $1
  AND (CAST(sqlc.narg('service_scope') AS varchar) IS NULL OR service_scope = sqlc.narg('service_scope'))
  AND (CAST(sqlc.narg('mime_type') AS varchar) IS NULL OR mime_type LIKE sqlc.narg('mime_type'))
  AND (CAST(sqlc.narg('status') AS media_status) IS NULL OR status = sqlc.narg('status'));
