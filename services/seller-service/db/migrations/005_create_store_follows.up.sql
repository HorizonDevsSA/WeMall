-- Create store follows table for users to follow/unfollow sellers
CREATE TABLE store_follows (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL,
    seller_id  UUID        NOT NULL REFERENCES sellers(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, seller_id)
);

CREATE INDEX idx_store_follows_user_id   ON store_follows(user_id);
CREATE INDEX idx_store_follows_seller_id ON store_follows(seller_id);

COMMENT ON TABLE store_follows IS 'Tracks which users follow which seller stores';
