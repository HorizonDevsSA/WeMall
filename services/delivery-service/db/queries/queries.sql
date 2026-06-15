-- name: ListActiveLogisticsPartners :many
SELECT * FROM logistics_partners
WHERE is_active = TRUE;

-- name: CreateCourier :one
INSERT INTO couriers (user_id, vehicle_type, plate_number, is_online, verification_status, rating)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCourierByUserID :one
SELECT * FROM couriers
WHERE user_id = $1;

-- name: UpdateCourierStatus :one
UPDATE couriers
SET is_online = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCourierLocation :one
UPDATE couriers
SET current_location = ST_SetSRID(ST_MakePoint($2, $3), 4326), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetActiveCouriersNearLocation :many
SELECT id, user_id, vehicle_type, plate_number, rating, ST_Distance(current_location, ST_SetSRID(ST_MakePoint($1, $2), 4326)) AS distance
FROM couriers
WHERE is_online = TRUE AND verification_status = 'approved' AND current_location IS NOT NULL
ORDER BY current_location <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)
LIMIT $3;

-- name: UpdateCourierBalance :one
UPDATE couriers
SET wallet_balance = wallet_balance + $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateStation :one
INSERT INTO stations (keeper_user_id, name, store_type, phone, address_line1, city, country, location, status, capacity_packages, operating_hours)
VALUES ($1, $2, $3, $4, $5, $6, $7, ST_SetSRID(ST_MakePoint($8, $9), 4326), $10, $11, $12)
RETURNING *;

-- name: GetStation :one
SELECT * FROM stations
WHERE id = $1;

-- name: GetStationsNearLocation :many
SELECT *, ST_Distance(location, ST_SetSRID(ST_MakePoint($1, $2), 4326)) AS distance
FROM stations
WHERE status = 'active'
ORDER BY location <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)
LIMIT $3;

-- name: UpdateStationPackageCount :one
UPDATE stations
SET current_package_count = current_package_count + $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateDeliveryOrder :one
INSERT INTO delivery_orders (
    tracking_number, order_id, sender_type, sender_id,
    sender_name, sender_phone, sender_address_line1, sender_city, sender_country, sender_location,
    recipient_name, recipient_phone, recipient_address_line1, recipient_city, recipient_country, recipient_location,
    delivery_type, origin_station_id, destination_station_id, weight_kg, dimensions_cm, shipping_fee, payment_status, status
)
VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, $9, ST_SetSRID(ST_MakePoint($10, $11), 4326),
    $12, $13, $14, $15, $16, ST_SetSRID(ST_MakePoint($17, $18), 4326),
    $19, $20, $21, $22, $23, $24, $25, $26
)
RETURNING *;

-- name: GetDeliveryOrder :one
SELECT * FROM delivery_orders
WHERE id = $1;

-- name: GetDeliveryOrderByTrackingNumber :one
SELECT * FROM delivery_orders
WHERE tracking_number = $1;

-- name: GetDeliveryOrderByOrderID :one
SELECT * FROM delivery_orders
WHERE order_id = $1;

-- name: UpdateDeliveryOrderStatus :one
UPDATE delivery_orders
SET status = $2, updated_at = NOW(),
    picked_up_at = CASE WHEN $2 = 'picked_up'::delivery_status THEN NOW() ELSE picked_up_at END,
    delivered_at = CASE WHEN $2 = 'delivered'::delivery_status THEN NOW() ELSE delivered_at END
WHERE id = $1
RETURNING *;

-- name: AssignCarrierToDeliveryOrder :one
UPDATE delivery_orders
SET carrier_type = $2, carrier_partner_id = $3, carrier_courier_id = $4, external_tracking_no = $5, status = 'assigned', updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDeliveryOrderPaymentStatus :one
UPDATE delivery_orders
SET payment_status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CreateCourierTask :one
INSERT INTO courier_tasks (delivery_order_id, courier_id, status)
VALUES ($1, $2, $3)
ON CONFLICT (delivery_order_id, courier_id) DO UPDATE
SET status = EXCLUDED.status, responded_at = CASE WHEN EXCLUDED.status != 'offered' THEN NOW() ELSE responded_at END
RETURNING *;

-- name: GetCourierTask :one
SELECT * FROM courier_tasks
WHERE delivery_order_id = $1 AND courier_id = $2;

-- name: UpdateCourierTaskStatus :one
UPDATE courier_tasks
SET status = $3, responded_at = NOW()
WHERE delivery_order_id = $1 AND courier_id = $2
RETURNING *;

-- name: GetAvailableCourierTasks :many
SELECT dor.*
FROM delivery_orders dor
JOIN courier_tasks ct ON dor.id = ct.delivery_order_id
WHERE ct.courier_id = $1 AND ct.status = 'offered' AND dor.status = 'assigned';

-- name: CheckInStationPackage :one
INSERT INTO station_packages (station_id, delivery_order_id, direction, shelf_code, verification_code, check_in_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: CheckOutStationPackage :one
UPDATE station_packages
SET check_out_at = NOW(), check_out_by = $2
WHERE delivery_order_id = $1 AND check_out_at IS NULL
RETURNING *;

-- name: GetStationPackageByTrackingNumber :one
SELECT sp.*
FROM station_packages sp
JOIN delivery_orders dor ON sp.delivery_order_id = dor.id
WHERE dor.tracking_number = $1;

-- name: GetStationInventory :many
SELECT sp.*, dor.tracking_number, dor.recipient_name, dor.recipient_phone
FROM station_packages sp
JOIN delivery_orders dor ON sp.delivery_order_id = dor.id
WHERE sp.station_id = $1 AND (($2::boolean = TRUE AND sp.check_out_at IS NULL) OR $2::boolean = FALSE);

-- name: CreateTrackingLog :one
INSERT INTO tracking_logs (delivery_order_id, status, location_desc, coordinate, details, operator_id)
VALUES ($1, $2, $3, CASE WHEN $4::double precision IS NOT NULL AND $5::double precision IS NOT NULL THEN ST_SetSRID(ST_MakePoint($4, $5), 4326) ELSE NULL END, $6, $7)
RETURNING *;

-- name: GetTrackingLogsByDeliveryOrderID :many
SELECT * FROM tracking_logs
WHERE delivery_order_id = $1
ORDER BY created_at DESC;
