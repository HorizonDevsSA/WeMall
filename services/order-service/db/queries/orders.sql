-- name: GetOrCreateCart :one
INSERT INTO carts (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET updated_at = NOW()
RETURNING id, user_id, created_at, updated_at;

-- name: GetCartItems :many
SELECT id, cart_id, variant_id, product_id, quantity, unit_price, added_at
FROM cart_items
WHERE cart_id = $1
ORDER BY added_at ASC;

-- name: AddToCartItem :exec
INSERT INTO cart_items (cart_id, variant_id, product_id, quantity, unit_price)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (cart_id, variant_id) DO UPDATE SET
    quantity   = cart_items.quantity + EXCLUDED.quantity,
    unit_price = EXCLUDED.unit_price,
    added_at   = NOW();

-- name: UpdateCartItemQuantity :exec
UPDATE cart_items
SET quantity = $3, added_at = NOW()
WHERE cart_id = $1 AND id = $2;

-- name: RemoveCartItem :exec
DELETE FROM cart_items
WHERE cart_id = $1 AND id = $2;

-- name: ClearCartItems :exec
DELETE FROM cart_items
WHERE cart_id = $1;

-- name: CreateOrder :one
INSERT INTO orders (order_number, user_id, status, subtotal, shipping_fee, discount_amount, total, shipping_address, coupon_code, notes, currency)
VALUES ($1, $2, 'pending', $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: CreateOrderItem :exec
INSERT INTO order_items (order_id, variant_id, product_id, seller_id, quantity, unit_price, snapshot, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending');

-- name: GetOrder :one
SELECT id, order_number, user_id, status, subtotal, shipping_fee, discount_amount, total,
       shipping_address, coupon_code, notes, currency, created_at, updated_at
FROM orders
WHERE id = $1 AND user_id = $2;

-- name: GetOrderItems :many
SELECT id, order_id, variant_id, product_id, seller_id, quantity, unit_price, snapshot, status, created_at
FROM order_items
WHERE order_id = $1;

-- name: ListOrders :many
SELECT id, order_number, user_id, status, subtotal, shipping_fee, discount_amount, total,
       shipping_address, coupon_code, notes, currency, created_at, updated_at
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountOrdersByUser :one
SELECT COUNT(*) FROM orders WHERE user_id = $1;

-- name: CancelOrder :one
UPDATE orders
SET status = 'cancelled', updated_at = NOW()
WHERE id = $1 AND user_id = $2 AND status = 'pending'
RETURNING id, order_number, user_id, status, subtotal, shipping_fee, discount_amount, total,
          shipping_address, coupon_code, notes, currency, created_at, updated_at;

-- name: CancelOrderItems :exec
UPDATE order_items SET status = 'cancelled'
WHERE order_id = $1;

-- name: UpdateOrderStatus :exec
UPDATE orders SET status = $2::order_status, updated_at = NOW()
WHERE id = $1;

-- name: UpdateOrderItemsStatus :exec
UPDATE order_items SET status = $2::order_status
WHERE order_id = $1;

-- name: GetCouponWithPromotion :one
SELECT
    c.id, c.code, c.promotion_id, c.max_uses, c.used_count, c.per_user_limit, c.expires_at, c.created_at,
    p.id, p.seller_id, p.name, p.type, p.value, p.min_order_value, p.max_discount,
    p.starts_at, p.ends_at, p.is_active, p.created_at
FROM coupons c
JOIN promotions p ON p.id = c.promotion_id
WHERE c.code = $1
  AND p.is_active = TRUE
  AND p.starts_at <= NOW()
  AND p.ends_at   >= NOW()
  AND (c.expires_at IS NULL OR c.expires_at >= NOW());

-- name: IncrementCouponUses :exec
UPDATE coupons SET used_count = used_count + 1
WHERE code = $1;
