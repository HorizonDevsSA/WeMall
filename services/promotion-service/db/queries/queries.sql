-- name: CreateCoupon :one
INSERT INTO coupons (
  code, seller_id, discount_type, discount_value, min_order_value, max_discount, start_date, end_date, usage_limit
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetCouponByCode :one
SELECT * FROM coupons
WHERE code = $1 LIMIT 1;

-- name: ListCouponsBySeller :many
SELECT * FROM coupons
WHERE seller_id = $1 OR seller_id = ''
ORDER BY created_at DESC;

-- name: IncrementCouponUsage :exec
UPDATE coupons
SET usage_count = usage_count + 1
WHERE id = $1 AND (usage_limit = 0 OR usage_count < usage_limit);

-- name: CreateFlashSale :one
INSERT INTO flash_sales (
  name, start_time, end_time, status
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: AddFlashSaleItem :one
INSERT INTO flash_sale_items (
  flash_sale_id, product_id, discount_price, stock_limit
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: ListActiveFlashSales :many
SELECT * FROM flash_sales
WHERE NOW() >= start_time AND NOW() <= end_time AND status = 'FLASH_SALE_STATUS_ACTIVE'
ORDER BY end_time ASC;

-- name: GetFlashSaleItems :many
SELECT * FROM flash_sale_items
WHERE flash_sale_id = $1;
