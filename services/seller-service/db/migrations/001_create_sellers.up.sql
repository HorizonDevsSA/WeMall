CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE sellers (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL UNIQUE,
    store_name  TEXT         NOT NULL UNIQUE,
    store_slug  TEXT         NOT NULL UNIQUE,
    logo_url    TEXT,
    banner_url  TEXT,
    description TEXT,
    rating      NUMERIC(3,2) NOT NULL DEFAULT 0,
    total_sales INTEGER      NOT NULL DEFAULT 0,
    is_verified BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sellers_user_id    ON sellers(user_id);
CREATE INDEX idx_sellers_store_slug ON sellers(store_slug);
