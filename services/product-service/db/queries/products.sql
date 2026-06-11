-- name: CreateProduct :one
INSERT INTO products (seller_id, category_id, slug, attributes, brand, origin_country, status, min_price, max_price, location, product_type, image_url, thumbnail_url)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, ST_SetSRID(ST_MakePoint($11::float, $10::float), 4326)::geography, $12::product_type, $13, $14)
RETURNING id, seller_id, category_id, slug, attributes, brand, origin_country, status, rating, review_count, sold_count, view_count, min_price, max_price, image_url, thumbnail_url, created_at, updated_at, deleted_at, product_type, ST_Y(location::geometry)::float AS latitude, ST_X(location::geometry)::float AS longitude;

-- name: CreateProductTranslation :exec
INSERT INTO product_translations (product_id, language, title, description)
VALUES ($1, $2, $3, $4)
ON CONFLICT (product_id, language) DO UPDATE SET title = EXCLUDED.title, description = EXCLUDED.description;

-- name: GetProductByID :one
SELECT
    p.id, p.seller_id, p.category_id, p.slug, p.attributes, p.brand, p.origin_country,
    p.status, p.rating, p.review_count, p.sold_count, p.view_count, p.min_price, p.max_price,
    p.image_url, p.thumbnail_url,
    p.created_at, p.updated_at, p.deleted_at, p.product_type,
    ST_Y(p.location::geometry)::float AS latitude,
    ST_X(p.location::geometry)::float AS longitude,
    COALESCE(t.title, t_en.title, '')       AS title,
    COALESCE(t.description, t_en.description) AS description
FROM products p
LEFT JOIN product_translations t    ON t.product_id = p.id AND t.language = $1
LEFT JOIN product_translations t_en ON t_en.product_id = p.id AND t_en.language = 'en'
WHERE p.id = $2 AND p.deleted_at IS NULL;

-- name: GetProductBySlug :one
SELECT
    p.id, p.seller_id, p.category_id, p.slug, p.attributes, p.brand, p.origin_country,
    p.status, p.rating, p.review_count, p.sold_count, p.view_count, p.min_price, p.max_price,
    p.image_url, p.thumbnail_url,
    p.created_at, p.updated_at, p.deleted_at, p.product_type,
    ST_Y(p.location::geometry)::float AS latitude,
    ST_X(p.location::geometry)::float AS longitude,
    COALESCE(t.title, t_en.title, '')       AS title,
    COALESCE(t.description, t_en.description) AS description
FROM products p
LEFT JOIN product_translations t    ON t.product_id = p.id AND t.language = $1
LEFT JOIN product_translations t_en ON t_en.product_id = p.id AND t_en.language = 'en'
WHERE p.slug = $2 AND p.deleted_at IS NULL;

-- name: GetProductBatch :many
SELECT
    p.id, p.seller_id, p.category_id, p.slug, p.attributes, p.brand, p.origin_country,
    p.status, p.rating, p.review_count, p.sold_count, p.view_count, p.min_price, p.max_price,
    p.image_url, p.thumbnail_url,
    p.created_at, p.updated_at, p.deleted_at, p.product_type,
    ST_Y(p.location::geometry)::float AS latitude,
    ST_X(p.location::geometry)::float AS longitude,
    COALESCE(t.title, t_en.title, '')       AS title,
    COALESCE(t.description, t_en.description) AS description
FROM products p
LEFT JOIN product_translations t    ON t.product_id = p.id AND t.language = $1
LEFT JOIN product_translations t_en ON t_en.product_id = p.id AND t_en.language = 'en'
WHERE p.id = ANY($2::uuid[]) AND p.deleted_at IS NULL;

-- name: ListNearbyProducts :many
SELECT 
    p.id, p.seller_id, p.category_id, p.slug, p.attributes, p.brand, p.origin_country,
    p.status, p.rating, p.review_count, p.sold_count, p.view_count, p.min_price, p.max_price,
    p.image_url, p.thumbnail_url,
    p.created_at, p.updated_at, p.deleted_at, p.product_type,
    ST_Y(p.location::geometry)::float AS latitude,
    ST_X(p.location::geometry)::float AS longitude,
    ST_Distance(p.location, ST_SetSRID(ST_MakePoint(@longitude::float, @latitude::float), 4326)::geography)::float AS distance_meters,
    COALESCE(t.title, t_en.title, '')       AS title,
    COALESCE(t.description, t_en.description) AS description
FROM products p
LEFT JOIN product_translations t    ON t.product_id = p.id AND @language::text = t.language
LEFT JOIN product_translations t_en ON t_en.product_id = p.id AND t_en.language = 'en'
WHERE p.deleted_at IS NULL
  AND p.status = 'active'
  AND ST_DWithin(p.location, ST_SetSRID(ST_MakePoint(@longitude::float, @latitude::float), 4326)::geography, @radius_meters::float)
ORDER BY distance_meters ASC
LIMIT @limit_val::int OFFSET @offset_val::int;

-- name: CountNearbyProducts :one
SELECT COUNT(*) FROM products p
WHERE p.deleted_at IS NULL
  AND p.status = 'active'
  AND ST_DWithin(p.location, ST_SetSRID(ST_MakePoint(@longitude::float, @latitude::float), 4326)::geography, @radius_meters::float);

-- name: UpdateProduct :exec
UPDATE products SET
    brand         = COALESCE(NULLIF(@brand::text, ''), brand),
    status        = COALESCE(NULLIF(@status::text, '')::product_status, status),
    image_url     = COALESCE(NULLIF(@image_url::text, ''), image_url),
    thumbnail_url = COALESCE(NULLIF(@thumbnail_url::text, ''), thumbnail_url),
    updated_at    = NOW()
WHERE id = @id AND deleted_at IS NULL;

-- name: DeleteProduct :exec
UPDATE products SET deleted_at = NOW()
WHERE id = $1 AND seller_id = $2;

-- name: CreateProductVariant :one
INSERT INTO product_variants (product_id, sku, options, price, compare_price, weight_grams, image_url, is_default)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, product_id, sku, options, price, compare_price, weight_grams, image_url, is_default, created_at, updated_at;

-- name: GetProductVariants :many
SELECT id, product_id, sku, options, price, compare_price, weight_grams, image_url, is_default, created_at, updated_at
FROM product_variants
WHERE product_id = $1
ORDER BY is_default DESC, price ASC;

-- name: GetVariantBatch :many
SELECT id, product_id, sku, options, price, compare_price, weight_grams, image_url, is_default, created_at, updated_at
FROM product_variants
WHERE id = ANY($1::uuid[]);

-- name: CreateProductImage :one
INSERT INTO product_images (product_id, url, alt_text, sort_order, is_primary)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, product_id, url, alt_text, sort_order, is_primary, created_at;

-- name: GetProductImages :many
SELECT id, product_id, url, alt_text, sort_order, is_primary, created_at
FROM product_images
WHERE product_id = $1
ORDER BY is_primary DESC, sort_order ASC;

-- name: CreateTag :one
INSERT INTO tags (name, slug)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
RETURNING id, name, slug;

-- name: AddProductTag :exec
INSERT INTO product_tags (product_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetProductTags :many
SELECT t.id, t.name, t.slug
FROM tags t
JOIN product_tags pt ON pt.tag_id = t.id
WHERE pt.product_id = $1;

-- name: DeleteProductImages :exec
DELETE FROM product_images
WHERE product_id = $1;
