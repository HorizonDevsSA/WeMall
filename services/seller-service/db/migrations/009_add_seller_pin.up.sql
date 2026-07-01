ALTER TABLE sellers
ADD COLUMN seller_pin_hash TEXT,
ADD COLUMN bank_details_last_updated TIMESTAMPTZ,
ADD COLUMN payouts_locked_until TIMESTAMPTZ;
