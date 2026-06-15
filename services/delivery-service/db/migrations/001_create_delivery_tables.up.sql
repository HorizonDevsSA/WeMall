-- Enable PostGIS extension for advanced geolocation queries
CREATE EXTENSION IF NOT EXISTS postgis;

-- Create Enums
CREATE TYPE delivery_status AS ENUM (
    'created',
    'assigned',
    'awaiting_pickup',
    'picked_up',
    'in_transit',
    'arrived_at_station',
    'out_for_delivery',
    'delivered',
    'failed',
    'returned'
);

CREATE TYPE delivery_type AS ENUM (
    'door_to_door',
    'station_to_door',
    'door_to_station',
    'station_to_station'
);

CREATE TYPE carrier_type AS ENUM (
    'crowdsourced',
    '3pl'
);

CREATE TYPE courier_status AS ENUM (
    'pending',
    'approved',
    'rejected',
    'suspended'
);

CREATE TYPE station_status AS ENUM (
    'pending',
    'active',
    'suspended'
);

CREATE TYPE station_package_direction AS ENUM (
    'inbound_for_buyer',
    'outbound_for_courier'
);

-- 1. Logistics Partners (3PL Companies)
CREATE TABLE logistics_partners (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(100)  NOT NULL,
    code           VARCHAR(20)   NOT NULL UNIQUE,
    api_endpoint   VARCHAR(255)  NOT NULL,
    api_key        TEXT          NOT NULL,
    is_active      BOOLEAN       NOT NULL DEFAULT TRUE,
    pricing_schema JSONB         NOT NULL,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- 2. Crowdsourced Couriers (Gig Riders)
CREATE TABLE couriers (
    id                  UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID            NOT NULL UNIQUE,
    vehicle_type        VARCHAR(20)     NOT NULL,
    plate_number        VARCHAR(20),
    is_online           BOOLEAN         NOT NULL DEFAULT FALSE,
    verification_status courier_status  NOT NULL DEFAULT 'pending',
    rating              DECIMAL(3,2)    NOT NULL DEFAULT 5.00,
    current_location    GEOMETRY(Point, 4326),
    wallet_balance      DECIMAL(12,2)   NOT NULL DEFAULT 0.00,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_couriers_location ON couriers USING GIST(current_location);
CREATE INDEX idx_couriers_online ON couriers(is_online) WHERE is_online = TRUE;

-- 3. WeMall Stations (驿站 Hubs)
CREATE TABLE stations (
    id                   UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    keeper_user_id       UUID            NOT NULL,
    name                 VARCHAR(100)    NOT NULL,
    store_type           VARCHAR(50)     NOT NULL,
    phone                VARCHAR(20)     NOT NULL,
    address_line1        TEXT            NOT NULL,
    city                 VARCHAR(100)    NOT NULL,
    country              VARCHAR(100)    NOT NULL,
    location             GEOMETRY(Point, 4326) NOT NULL,
    status               station_status  NOT NULL DEFAULT 'pending',
    capacity_packages    INTEGER         NOT NULL DEFAULT 500,
    current_package_count INTEGER        NOT NULL DEFAULT 0,
    operating_hours      JSONB           NOT NULL,
    created_at           TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_stations_location ON stations USING GIST(location);
CREATE INDEX idx_stations_city ON stations(city);

-- 4. Delivery Orders (The Waybill Manifest)
CREATE TABLE delivery_orders (
    id                     UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tracking_number        VARCHAR(50)     NOT NULL UNIQUE,
    order_id               UUID,
    sender_type            VARCHAR(20)     NOT NULL,
    sender_id              UUID            NOT NULL,
    
    sender_name            VARCHAR(100)    NOT NULL,
    sender_phone           VARCHAR(20)     NOT NULL,
    sender_address_line1   TEXT            NOT NULL,
    sender_city            VARCHAR(100)    NOT NULL,
    sender_country         VARCHAR(100)    NOT NULL,
    sender_location        GEOMETRY(Point, 4326) NOT NULL,
    
    recipient_name          VARCHAR(100)    NOT NULL,
    recipient_phone         VARCHAR(20)     NOT NULL,
    recipient_address_line1 TEXT            NOT NULL,
    recipient_city          VARCHAR(100)    NOT NULL,
    recipient_country       VARCHAR(100)    NOT NULL,
    recipient_location      GEOMETRY(Point, 4326) NOT NULL,
    
    delivery_type          delivery_type   NOT NULL,
    origin_station_id      UUID,
    destination_station_id UUID,
    
    carrier_type           carrier_type,
    carrier_partner_id     UUID,
    carrier_courier_id     UUID,
    external_tracking_no   VARCHAR(100),
    
    weight_kg              DECIMAL(10,2)   NOT NULL DEFAULT 1.00,
    dimensions_cm          JSONB           NOT NULL,
    shipping_fee           DECIMAL(10,2)   NOT NULL,
    payment_status         VARCHAR(20)     NOT NULL DEFAULT 'pending',
    
    status                 delivery_status NOT NULL DEFAULT 'created',
    created_at             TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    picked_up_at           TIMESTAMPTZ,
    delivered_at           TIMESTAMPTZ
);
CREATE INDEX idx_delivery_orders_tracking ON delivery_orders(tracking_number);
CREATE INDEX idx_delivery_orders_status ON delivery_orders(status);
CREATE INDEX idx_delivery_orders_order_id ON delivery_orders(order_id);
CREATE INDEX idx_delivery_orders_courier ON delivery_orders(carrier_courier_id) WHERE carrier_type = 'crowdsourced';

-- 5. Courier Dispatch Tasks
CREATE TABLE courier_tasks (
    id                 UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_order_id  UUID            NOT NULL REFERENCES delivery_orders(id),
    courier_id         UUID            NOT NULL REFERENCES couriers(id),
    status             VARCHAR(20)     NOT NULL DEFAULT 'offered',
    offered_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    responded_at       TIMESTAMPTZ,
    UNIQUE(delivery_order_id, courier_id)
);
CREATE INDEX idx_courier_tasks_status ON courier_tasks(courier_id, status);

-- 6. Station Packages (Package physical tracking inside Stations)
CREATE TABLE station_packages (
    id                 UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    station_id         UUID            NOT NULL REFERENCES stations(id),
    delivery_order_id  UUID            NOT NULL UNIQUE REFERENCES delivery_orders(id),
    direction          station_package_direction NOT NULL,
    shelf_code         VARCHAR(20)     NOT NULL,
    verification_code  VARCHAR(8)      NOT NULL,
    check_in_at        TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    check_in_by        UUID            NOT NULL,
    check_out_at       TIMESTAMPTZ,
    check_out_by       UUID,
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_station_packages_unclaimed ON station_packages(station_id) WHERE check_out_at IS NULL;

-- 7. Tracking Logs
CREATE TABLE tracking_logs (
    id                 UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_order_id  UUID            NOT NULL REFERENCES delivery_orders(id),
    status             delivery_status NOT NULL,
    location_desc      TEXT            NOT NULL,
    coordinate         GEOMETRY(Point, 4326),
    details            TEXT,
    operator_id        UUID,
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_tracking_logs_order ON tracking_logs(delivery_order_id, created_at DESC);
