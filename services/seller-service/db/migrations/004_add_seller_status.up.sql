-- Add seller_status enum for the review process
CREATE TYPE seller_status AS ENUM ('pending', 'processing', 'verified', 'suspended');

-- Add status column to sellers table, default is 'pending' on registration
ALTER TABLE sellers
    ADD COLUMN status seller_status NOT NULL DEFAULT 'pending';

-- Create an index for efficient status filtering
CREATE INDEX idx_sellers_status ON sellers(status);

COMMENT ON COLUMN sellers.status IS 'Review status: pending → processing → verified | suspended';
