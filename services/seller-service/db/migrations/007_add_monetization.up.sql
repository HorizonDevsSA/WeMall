ALTER TABLE sellers 
ADD COLUMN commission_rate NUMERIC(4,2) NOT NULL DEFAULT 0.05,
ADD COLUMN ad_credit_balance NUMERIC(12,2) NOT NULL DEFAULT 0.00;

ALTER TABLE seller_payouts 
ADD COLUMN gross_amount NUMERIC(12,2) NOT NULL DEFAULT 0.00,
ADD COLUMN platform_fee NUMERIC(12,2) NOT NULL DEFAULT 0.00,
ADD COLUMN net_amount NUMERIC(12,2) NOT NULL DEFAULT 0.00;

CREATE TABLE seller_earnings (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id      UUID          NOT NULL REFERENCES sellers(id) ON DELETE CASCADE,
    order_id       UUID          NOT NULL,
    order_item_id  UUID          NOT NULL,
    gross_amount   NUMERIC(12,2) NOT NULL,
    commission_fee NUMERIC(12,2) NOT NULL,
    net_amount     NUMERIC(12,2) NOT NULL,
    status         TEXT          NOT NULL DEFAULT 'escrowed',
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    settled_at     TIMESTAMPTZ,
    payout_id      UUID          REFERENCES seller_payouts(id) ON DELETE SET NULL,
    
    CONSTRAINT seller_earnings_status_check
        CHECK (status IN ('escrowed', 'earned', 'refunded', 'payout_released'))
);

CREATE INDEX idx_seller_earnings_seller_id ON seller_earnings(seller_id);
CREATE INDEX idx_seller_earnings_status    ON seller_earnings(status);
