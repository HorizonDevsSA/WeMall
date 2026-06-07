-- name: CreateReview :one
INSERT INTO reviews (
    order_id, buyer_id, seller_id, product_id, variant_id,
    rating_description, rating_service, rating_delivery,
    content, is_anonymous, has_media, nlp_tags, is_system_generated
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: CreateReviewMedia :one
INSERT INTO review_media (
    review_id, media_url, media_type, sort_order
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetReviewMedia :many
SELECT * FROM review_media WHERE review_id = $1 ORDER BY sort_order ASC;

-- name: CreateAppendReview :one
INSERT INTO append_reviews (
    review_id, content, has_media
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: CreateAppendReviewMedia :one
INSERT INTO append_review_media (
    append_review_id, media_url, media_type, sort_order
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetAppendReview :one
SELECT * FROM append_reviews WHERE review_id = $1 LIMIT 1;

-- name: GetAppendReviewMedia :many
SELECT * FROM append_review_media WHERE append_review_id = $1 ORDER BY sort_order ASC;

-- name: CreateSellerReply :one
INSERT INTO seller_replies (
    review_id, reply_type, content
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: GetSellerReplies :many
SELECT * FROM seller_replies WHERE review_id = $1 ORDER BY created_at ASC;

-- name: UpdateReview :one
UPDATE reviews
SET rating_description = $2,
    rating_service = $3,
    rating_delivery = $4,
    content = $5,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteReview :one
UPDATE reviews
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetReview :one
SELECT * FROM reviews WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetReviewByOrderAndVariant :one
SELECT * FROM reviews WHERE order_id = $1 AND variant_id = $2 AND deleted_at IS NULL LIMIT 1;

-- name: ListProductReviews :many
SELECT r.* FROM reviews r
LEFT JOIN append_reviews ar ON r.id = ar.review_id
WHERE r.product_id = $1 
  AND r.deleted_at IS NULL
  AND (
      $2::text = 'ALL' OR
      ($2::text = 'GOOD' AND r.review_type = 'good') OR
      ($2::text = 'NEUTRAL' AND r.review_type = 'neutral') OR
      ($2::text = 'BAD' AND r.review_type = 'bad') OR
      ($2::text = 'HAS_MEDIA' AND (r.has_media = true OR ar.has_media = true)) OR
      ($2::text = 'APPEND' AND ar.id IS NOT NULL)
  )
ORDER BY r.created_at DESC
LIMIT $4 OFFSET $3;

-- name: CountProductReviews :one
SELECT COUNT(*) FROM reviews r
LEFT JOIN append_reviews ar ON r.id = ar.review_id
WHERE r.product_id = $1 
  AND r.deleted_at IS NULL
  AND (
      $2::text = 'ALL' OR
      ($2::text = 'GOOD' AND r.review_type = 'good') OR
      ($2::text = 'NEUTRAL' AND r.review_type = 'neutral') OR
      ($2::text = 'BAD' AND r.review_type = 'bad') OR
      ($2::text = 'HAS_MEDIA' AND (r.has_media = true OR ar.has_media = true)) OR
      ($2::text = 'APPEND' AND ar.id IS NOT NULL)
  );

-- name: ListSellerReviews :many
SELECT * FROM reviews
WHERE seller_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $3 OFFSET $2;

-- name: CountSellerReviews :one
SELECT COUNT(*) FROM reviews
WHERE seller_id = $1 AND deleted_at IS NULL;

-- name: GetProductRatingStats :one
SELECT 
    COALESCE(AVG(rating_description)::float, 0.0)::float as avg_rating,
    COUNT(*)::int as total_reviews,
    COUNT(CASE WHEN review_type = 'good' THEN 1 END)::int as good_count,
    COUNT(CASE WHEN review_type = 'neutral' THEN 1 END)::int as neutral_count,
    COUNT(CASE WHEN review_type = 'bad' THEN 1 END)::int as bad_count,
    COUNT(CASE WHEN has_media = true THEN 1 END)::int as has_media_count
FROM reviews
WHERE product_id = $1 AND deleted_at IS NULL;

-- name: GetProductAppendCount :one
SELECT COUNT(ar.id)::int FROM append_reviews ar
JOIN reviews r ON ar.review_id = r.id
WHERE r.product_id = $1 AND r.deleted_at IS NULL;

-- name: GetSellerDSR :one
SELECT 
    COALESCE(AVG(rating_description)::float, 0.0)::float as avg_description,
    COALESCE(AVG(rating_service)::float, 0.0)::float as avg_service,
    COALESCE(AVG(rating_delivery)::float, 0.0)::float as avg_delivery,
    -- Reputation Score: Good (+1), Neutral (0), Bad (-1)
    COALESCE(SUM(
        CASE 
            WHEN review_type = 'good' THEN 1 
            WHEN review_type = 'bad' THEN -1 
            ELSE 0 
        END
    ), 0)::int as reputation_score
FROM reviews
WHERE seller_id = $1 
  AND deleted_at IS NULL 
  AND created_at >= NOW() - INTERVAL '180 days';

-- name: InsertOrderDelivery :one
INSERT INTO order_deliveries (order_id, buyer_id, delivered_at)
VALUES ($1, $2, $3)
ON CONFLICT (order_id) DO NOTHING
RETURNING *;

-- name: GetUnprocessedDeliveries :many
SELECT * FROM order_deliveries
WHERE is_processed = FALSE AND delivered_at <= $1::timestamptz;

-- name: MarkDeliveryProcessed :exec
UPDATE order_deliveries
SET is_processed = TRUE
WHERE order_id = $1;
