-- Enable PostGIS extension for spatial queries
CREATE EXTENSION IF NOT EXISTS postgis;

-- Enable btree_gist extension for combining UUID equality with range checks in exclusion constraints
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Create Enums
CREATE TYPE property_type AS ENUM (
    'apartment', 'house', 'villa', 'condo', 'townhouse', 'cabin', 'studio', 'land'
);

CREATE TYPE listing_type AS ENUM (
    'rental', 'sale'
);

CREATE TYPE listing_status AS ENUM (
    'draft', 'pending_approval', 'active', 'suspended', 'sold', 'rented', 'inactive'
);

CREATE TYPE booking_status AS ENUM (
    'pending_payment', 'requested', 'confirmed', 'cancelled', 'active', 'completed', 'disputed'
);

CREATE TYPE offer_status AS ENUM (
    'submitted', 'accepted', 'rejected', 'under_escrow', 'funds_released', 'cancelled', 'disputed'
);

CREATE TYPE appointment_status AS ENUM (
    'scheduled', 'confirmed', 'completed', 'cancelled', 'no_show'
);

-- 1. Properties Table (Core Entity)
CREATE TABLE properties (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id            UUID            NOT NULL, -- Reference to User Service (Host / Seller)
    type                property_type   NOT NULL,
    title               VARCHAR(255)    NOT NULL,
    description         TEXT            NOT NULL,
    address_line1       VARCHAR(255)    NOT NULL,
    address_line2       VARCHAR(255),
    city                VARCHAR(100)    NOT NULL,
    state_province      VARCHAR(100)    NOT NULL,
    country             VARCHAR(100)    NOT NULL,
    postal_code         VARCHAR(20),
    -- Geo Location point (using WGS 84, SRID 4326)
    location            GEOMETRY(Point, 4326) NOT NULL,
    bedroom_count       INT             NOT NULL DEFAULT 0,
    bathroom_count      DECIMAL(3,1)    NOT NULL DEFAULT 0.0,
    max_guests          INT             NOT NULL DEFAULT 0, -- Relevant for rentals
    square_meters       DECIMAL(8,2)    NOT NULL DEFAULT 0.0, -- Relevant for sales/rentals
    is_verified         BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- Spatial index for fast geographic queries
CREATE INDEX idx_properties_location ON properties USING GIST(location);
CREATE INDEX idx_properties_owner ON properties(owner_id);

-- 2. Property Images
CREATE TABLE property_images (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id   UUID          NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    url           TEXT          NOT NULL,
    display_order INT           NOT NULL DEFAULT 0,
    is_cover      BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_property_images_property ON property_images(property_id);

-- 3. Property Amenities
CREATE TABLE property_amenities (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id   UUID          NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    name          VARCHAR(100)  NOT NULL, -- e.g., 'WiFi', 'Pool', 'Air Conditioning'
    category      VARCHAR(50)   NOT NULL, -- e.g., 'utilities', 'luxury', 'safety'
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_property_amenities_unique ON property_amenities(property_id, name);

-- 4. Listings Table (Connects properties to market states)
CREATE TABLE listings (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    property_id     UUID            NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    type            listing_type    NOT NULL,
    status          listing_status  NOT NULL DEFAULT 'draft',
    base_price      DECIMAL(12,2)   NOT NULL, -- Nightly rate for rental, Total price for sale
    currency        VARCHAR(3)      NOT NULL DEFAULT 'USD',
    is_instant_book BOOLEAN         NOT NULL DEFAULT FALSE, -- Only for rentals
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_listings_property ON listings(property_id);
CREATE INDEX idx_listings_status_type ON listings(status, type);

-- 5. Rental Metadata (Specific to short-term/long-term rentals)
CREATE TABLE rental_listings_meta (
    listing_id        UUID            PRIMARY KEY REFERENCES listings(id) ON DELETE CASCADE,
    cleaning_fee      DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
    security_deposit  DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
    min_nights        INT             NOT NULL DEFAULT 1,
    max_nights        INT             NOT NULL DEFAULT 90,
    check_in_time     TIME            NOT NULL DEFAULT '15:00:00',
    check_out_time    TIME            NOT NULL DEFAULT '11:00:00'
);

-- 6. Sales Metadata (Specific to property buying/selling)
CREATE TABLE sales_listings_meta (
    listing_id             UUID            PRIMARY KEY REFERENCES listings(id) ON DELETE CASCADE,
    escrow_deposit_percent DECIMAL(5,2)    NOT NULL DEFAULT 10.00, -- e.g. 10% deposit required
    agent_commission_rate  DECIMAL(5,2)    NOT NULL DEFAULT 3.00,  -- e.g. 3% standard rate
    includes_furniture     BOOLEAN         NOT NULL DEFAULT FALSE,
    year_built             INT,
    property_tax_annual    DECIMAL(12,2)
);

-- 7. Rental Bookings Table (Airbnb style reservation calendar)
CREATE TABLE rental_bookings (
    id                UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id        UUID            NOT NULL REFERENCES listings(id),
    tenant_id         UUID            NOT NULL, -- Reference to User Service (Guest)
    start_date        DATE            NOT NULL,
    end_date          DATE            NOT NULL,
    nightly_price     DECIMAL(12,2)   NOT NULL,
    cleaning_fee      DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
    security_deposit  DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
    total_price       DECIMAL(12,2)   NOT NULL,
    status            booking_status  NOT NULL DEFAULT 'requested',
    payment_intent_id VARCHAR(255),
    created_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    -- Exclude overlapping active bookings on database level (PostgreSQL Exclude Constraint)
    CONSTRAINT no_overlapping_bookings EXCLUDE USING gist (
        listing_id WITH =,
        daterange(start_date, end_date, '[]') WITH &&
    ) WHERE (status IN ('confirmed', 'active'))
);
CREATE INDEX idx_bookings_tenant ON rental_bookings(tenant_id);
CREATE INDEX idx_bookings_listing ON rental_bookings(listing_id);

-- 8. Sales Offers Table (Real estate bids/contracts)
CREATE TABLE sales_offers (
    id                 UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id         UUID            NOT NULL REFERENCES listings(id),
    buyer_id           UUID            NOT NULL, -- Reference to User Service
    offer_price        DECIMAL(12,2)   NOT NULL,
    escrow_deposit_paid DECIMAL(12,2)  NOT NULL DEFAULT 0.00,
    status             offer_status    NOT NULL DEFAULT 'submitted',
    conditions_text    TEXT,           -- e.g. "Subject to structural survey"
    expiration_date    TIMESTAMPTZ     NOT NULL,
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_offers_buyer ON sales_offers(buyer_id);
CREATE INDEX idx_offers_listing ON sales_offers(listing_id);

-- 9. Viewing Appointments (Physical property inspections)
CREATE TABLE viewing_appointments (
    id             UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id     UUID               NOT NULL REFERENCES listings(id),
    client_id      UUID               NOT NULL, -- User wanting to inspect
    host_id        UUID               NOT NULL, -- Owner/Agent showing the property
    scheduled_time TIMESTAMPTZ        NOT NULL,
    status         appointment_status NOT NULL DEFAULT 'scheduled',
    notes          TEXT,
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_appointments_client ON viewing_appointments(client_id);
CREATE INDEX idx_appointments_host ON viewing_appointments(host_id);
CREATE INDEX idx_appointments_time ON viewing_appointments(scheduled_time);
