-- Revert seller status column and enum
ALTER TABLE sellers DROP COLUMN IF EXISTS status;
DROP INDEX IF EXISTS idx_sellers_status;
DROP TYPE IF EXISTS seller_status;
