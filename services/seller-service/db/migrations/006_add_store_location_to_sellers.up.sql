-- Add store_location column to sellers table
-- This stores the human-readable address string derived from the seller's coordinates
ALTER TABLE sellers
ADD COLUMN store_location TEXT;

COMMENT ON COLUMN sellers.store_location IS 'Human-readable address string for the store location, derived from latitude/longitude coordinates';
