-- name: CreateSeller :one
INSERT INTO sellers (user_id, store_name, store_slug, logo_url, banner_url, description, latitude, longitude, store_location)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
    store_name                  = COALESCE(NULLIF(@store_name::text, ''), store_name),
    store_slug                  = COALESCE(NULLIF(@store_slug::text, ''), store_slug),
    logo_url                    = COALESCE(sqlc.narg('logo_url'), logo_url),
    banner_url                  = COALESCE(sqlc.narg('banner_url'), banner_url),
    description                 = COALESCE(sqlc.narg('description'), description),
    latitude                    = COALESCE(sqlc.narg('latitude'), latitude),
    longitude                   = COALESCE(sqlc.narg('longitude'), longitude),
    store_location              = COALESCE(sqlc.narg('store_location'), store_location),
    shipping_zones              = COALESCE(sqlc.narg('shipping_zones'), shipping_zones),
    bank_name                   = COALESCE(sqlc.narg('bank_name'), bank_name),
    bank_account_number         = COALESCE(sqlc.narg('bank_account_number'), bank_account_number),
    ecocash_number              = COALESCE(sqlc.narg('ecocash_number'), ecocash_number),
    return_window_days          = COALESCE(sqlc.narg('return_window_days'), return_window_days),
    return_policy_text          = COALESCE(sqlc.narg('return_policy_text'), return_policy_text),
    push_notifications_enabled   = COALESCE(sqlc.narg('push_notifications_enabled'), push_notifications_enabled),
    email_alerts_enabled        = COALESCE(sqlc.narg('email_alerts_enabled'), email_alerts_enabled),
    sms_alerts_enabled          = COALESCE(sqlc.narg('sms_alerts_enabled'), sms_alerts_enabled),
    auto_accept_orders          = COALESCE(sqlc.narg('auto_accept_orders'), auto_accept_orders),
    inventory_alerts_enabled    = COALESCE(sqlc.narg('inventory_alerts_enabled'), inventory_alerts_enabled),
    profile_visibility          = COALESCE(sqlc.narg('profile_visibility'), profile_visibility),
    search_indexing_enabled     = COALESCE(sqlc.narg('search_indexing_enabled'), search_indexing_enabled),
    data_sharing_enabled        = COALESCE(sqlc.narg('data_sharing_enabled'), data_sharing_enabled),
    two_factor_enabled          = COALESCE(sqlc.narg('two_factor_enabled'), two_factor_enabled),
    deactivation_reason         = COALESCE(sqlc.narg('deactivation_reason'), deactivation_reason),
    updated_at                  = NOW()
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

-- name: UpdateSellerAdCredit :one
UPDATE sellers SET
    ad_credit_balance = ad_credit_balance + $2,
    updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateSellerCommissionRate :one
UPDATE sellers SET
    commission_rate = $2,
    updated_at  = NOW()
WHERE id = $1
RETURNING *;


-- name: UpdateSellerPIN :one
UPDATE sellers SET
    seller_pin_hash = $2,
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;


-- name: UpdateSellerSecurityCooldown :one
UPDATE sellers SET
    bank_details_last_updated = COALESCE($2, bank_details_last_updated),
    payouts_locked_until = COALESCE($3, payouts_locked_until),
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;


