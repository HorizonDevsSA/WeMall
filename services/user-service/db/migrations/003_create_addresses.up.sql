CREATE TABLE addresses (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label         TEXT,
    full_name     TEXT        NOT NULL,
    phone         TEXT        NOT NULL,
    address_line1 TEXT        NOT NULL,
    address_line2 TEXT,
    city          TEXT        NOT NULL,
    state         TEXT,
    postal_code   TEXT,
    country       TEXT        NOT NULL DEFAULT 'ZW',
    is_default    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_addresses_user_id ON addresses(user_id);
