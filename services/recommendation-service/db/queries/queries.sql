-- name: UpsertProductView :exec
INSERT INTO product_views (buyer_id, product_id, view_count, last_viewed_at)
VALUES ($1, $2, 1, NOW())
ON CONFLICT (buyer_id, product_id)
DO UPDATE SET 
    view_count = product_views.view_count + 1,
    last_viewed_at = NOW();

-- name: UpsertCoPurchase :exec
INSERT INTO product_co_purchases (product_a_id, product_b_id, co_purchase_count, last_purchased_at)
VALUES ($1, $2, 1, NOW())
ON CONFLICT (product_a_id, product_b_id)
DO UPDATE SET 
    co_purchase_count = product_co_purchases.co_purchase_count + 1,
    last_purchased_at = NOW();

-- name: GetFrequentlyBoughtTogether :many
SELECT product_b_id AS product_id, co_purchase_count AS score
FROM product_co_purchases
WHERE product_a_id = $1
ORDER BY co_purchase_count DESC
LIMIT $2;

-- name: GetRecentProductViews :many
SELECT product_id
FROM product_views
WHERE buyer_id = $1
ORDER BY last_viewed_at DESC
LIMIT $2;

-- name: GetTopProductsGlobally :many
SELECT product_id, SUM(view_count) as score
FROM product_views
GROUP BY product_id
ORDER BY score DESC
LIMIT $1;
