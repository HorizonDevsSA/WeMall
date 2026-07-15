-- name: CreateProperty :one
INSERT INTO properties (
    owner_id, type, title, description, address_line1, address_line2, city, state_province, country, postal_code, location, bedroom_count, bathroom_count, max_guests, square_meters
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, ST_SetSRID(ST_MakePoint($11::double precision, $12::double precision), 4326), $13, $14, $15, $16
) RETURNING *;

-- name: GetProperty :one
SELECT id, owner_id, type, title, description, address_line1, address_line2, city, state_province, country, postal_code, 
       (ST_Y(location::geometry))::double precision AS latitude, (ST_X(location::geometry))::double precision AS longitude, 
       bedroom_count, bathroom_count, max_guests, square_meters, is_verified, created_at, updated_at
FROM properties
WHERE id = $1;

-- name: CreatePropertyImage :one
INSERT INTO property_images (
    property_id, url, display_order, is_cover
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetPropertyImages :many
SELECT * FROM property_images WHERE property_id = $1 ORDER BY display_order ASC;

-- name: CreatePropertyAmenity :one
INSERT INTO property_amenities (
    property_id, name, category
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetPropertyAmenities :many
SELECT * FROM property_amenities WHERE property_id = $1;

-- name: CreateListing :one
INSERT INTO listings (
    property_id, type, status, base_price, currency, is_instant_book
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetListing :one
SELECT l.*, 
       p.owner_id, p.type AS property_type, p.title, p.description, p.address_line1, p.address_line2, p.city, p.state_province, p.country, p.postal_code,
       (ST_Y(p.location::geometry))::double precision AS latitude, (ST_X(p.location::geometry))::double precision AS longitude,
       p.bedroom_count, p.bathroom_count, p.max_guests, p.square_meters, p.is_verified
FROM listings l
JOIN properties p ON l.property_id = p.id
WHERE l.id = $1;

-- name: CreateRentalListingMeta :one
INSERT INTO rental_listings_meta (
    listing_id, cleaning_fee, security_deposit, min_nights, max_nights, check_in_time, check_out_time
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetRentalListingMeta :one
SELECT * FROM rental_listings_meta WHERE listing_id = $1;

-- name: CreateSalesListingMeta :one
INSERT INTO sales_listings_meta (
    listing_id, escrow_deposit_percent, agent_commission_rate, includes_furniture, year_built, property_tax_annual
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetSalesListingMeta :one
SELECT * FROM sales_listings_meta WHERE listing_id = $1;

-- name: SearchNearbyListings :many
SELECT l.id AS listing_id, l.property_id, l.type AS listing_type, l.status AS listing_status, l.base_price, l.currency, l.is_instant_book, l.created_at, l.updated_at,
       p.owner_id, p.type AS property_type, p.title, p.description, p.address_line1, p.address_line2, p.city, p.state_province, p.country, p.postal_code,
       (ST_Y(p.location::geometry))::double precision AS latitude, (ST_X(p.location::geometry))::double precision AS longitude,
       p.bedroom_count, p.bathroom_count, p.max_guests, p.square_meters, p.is_verified,
       ST_Distance(
           p.location, 
           ST_SetSRID(ST_MakePoint($1::double precision, $2::double precision), 4326)::geography
       ) AS distance_meters
FROM properties p
JOIN listings l ON p.id = l.property_id
WHERE l.status = 'active'
  AND ST_DWithin(
      p.location, 
      ST_SetSRID(ST_MakePoint($1::double precision, $2::double precision), 4326)::geography, 
      $3::double precision
  )
ORDER BY distance_meters ASC
LIMIT $4 OFFSET $5;

-- name: GetListingsInViewport :many
SELECT l.id AS listing_id, l.property_id, l.type AS listing_type, l.status AS listing_status, l.base_price, l.currency, l.is_instant_book, l.created_at, l.updated_at,
       p.owner_id, p.type AS property_type, p.title, p.description, p.address_line1, p.address_line2, p.city, p.state_province, p.country, p.postal_code,
       (ST_Y(p.location::geometry))::double precision AS latitude, (ST_X(p.location::geometry))::double precision AS longitude,
       p.bedroom_count, p.bathroom_count, p.max_guests, p.square_meters, p.is_verified
FROM properties p
JOIN listings l ON p.id = l.property_id
WHERE l.status = 'active'
  AND p.location && ST_MakeEnvelope(
      $1::double precision, -- Min Longitude (West)
      $2::double precision, -- Min Latitude (South)
      $3::double precision, -- Max Longitude (East)
      $4::double precision, -- Max Latitude (North)
      4326
  );

-- name: CreateBooking :one
INSERT INTO rental_bookings (
    listing_id, tenant_id, start_date, end_date, nightly_price, cleaning_fee, security_deposit, total_price, status, payment_intent_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetBooking :one
SELECT * FROM rental_bookings WHERE id = $1;

-- name: CreateOffer :one
INSERT INTO sales_offers (
    listing_id, buyer_id, offer_price, escrow_deposit_paid, status, conditions_text, expiration_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetOffer :one
SELECT * FROM sales_offers WHERE id = $1;

-- name: AcceptOffer :one
UPDATE sales_offers
SET status = 'accepted', updated_at = NOW()
WHERE id = $1 AND status = 'submitted'
RETURNING *;

-- name: UpdateOfferStatus :one
UPDATE sales_offers
SET status = $2, escrow_deposit_paid = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ScheduleViewing :one
INSERT INTO viewing_appointments (
    listing_id, client_id, host_id, scheduled_time, notes, status
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetViewingAppointment :one
SELECT * FROM viewing_appointments WHERE id = $1;
