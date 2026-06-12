CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE user_bans (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    reason      TEXT NOT NULL,
    admin_id    UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE seller_suspensions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id   UUID NOT NULL,
    reason      TEXT NOT NULL,
    admin_id    UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_bans_user_id ON user_bans(user_id);
CREATE INDEX idx_seller_suspensions_seller_id ON seller_suspensions(seller_id);
