CREATE TABLE seller_payouts (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id    UUID          NOT NULL REFERENCES sellers(id) ON DELETE CASCADE,
    amount       NUMERIC(12,2) NOT NULL,
    currency     TEXT          NOT NULL DEFAULT 'USD',
    status       TEXT          NOT NULL DEFAULT 'pending',
    provider_ref TEXT,
    paid_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT seller_payouts_status_check
        CHECK (status IN ('pending', 'processing', 'paid', 'failed'))
);

CREATE INDEX idx_seller_payouts_seller_id ON seller_payouts(seller_id);
CREATE INDEX idx_seller_payouts_status    ON seller_payouts(status);
