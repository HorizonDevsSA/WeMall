-- name: CreateSeller :one
INSERT INTO sellers (user_id, store_name, store_slug, logo_url, banner_url, description, latitude, longitude)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetSellerByID :one
SELECT * FROM sellers WHERE id = $1;

-- name: GetSellerByUserID :one
SELECT * FROM sellers WHERE user_id = $1;

-- name: GetSellerByStoreSlug :one
SELECT * FROM sellers WHERE store_slug = $1;

-- name: GetSellersByIDs :many
SELECT * FROM sellers WHERE id = ANY($1::uuid[]);

-- name: UpdateSeller :one
UPDATE sellers SET
    store_name  = COALESCE(NULLIF(@store_name::text, ''), store_name),
    store_slug  = COALESCE(NULLIF(@store_slug::text, ''), store_slug),
    logo_url    = COALESCE(sqlc.narg('logo_url'), logo_url),
    banner_url  = COALESCE(sqlc.narg('banner_url'), banner_url),
    description = COALESCE(sqlc.narg('description'), description),
    latitude    = COALESCE(sqlc.narg('latitude'), latitude),
    longitude   = COALESCE(sqlc.narg('longitude'), longitude),
    updated_at  = NOW()
WHERE user_id = @user_id
RETURNING *;

-- name: UpdateSellerStatus :one
UPDATE sellers SET
    status     = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: VerifySeller :one
UPDATE sellers SET
    is_verified = $2,
    updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: IncrementSellerTotalSales :one
UPDATE sellers SET
    total_sales = total_sales + $2,
    updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: GetSellersNearLocation :many
SELECT *, 
    (6371 * acos(cos(radians($2)) * cos(radians(latitude)) * cos(radians(longitude) - radians($3)) + sin(radians($2)) * sin(radians(latitude)))) AS distance_km
FROM sellers 
WHERE latitude IS NOT NULL AND longitude IS NOT NULL
    AND latitude BETWEEN $2 - ($4 / 111.32) AND $2 + ($4 / 111.32)
    AND longitude BETWEEN $3 - ($4 / (111.32 * cos(radians($2)))) AND $3 + ($4 / (111.32 * cos(radians($2))))
    AND (6371 * acos(cos(radians($2)) * cos(radians(latitude)) * cos(radians(longitude) - radians($3)) + sin(radians($2)) * sin(radians(latitude)))) <= $4
ORDER BY (6371 * acos(cos(radians($2)) * cos(radians(latitude)) * cos(radians(longitude) - radians($3)) + sin(radians($2)) * sin(radians(latitude))))
LIMIT $1;

-- name: GetSellersWithinRadius :many  
SELECT *,
    (6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) AS distance_km
FROM sellers 
WHERE latitude IS NOT NULL AND longitude IS NOT NULL
    AND (6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude)))) <= $3
ORDER BY (6371 * acos(cos(radians($1)) * cos(radians(latitude)) * cos(radians(longitude) - radians($2)) + sin(radians($1)) * sin(radians(latitude))));
