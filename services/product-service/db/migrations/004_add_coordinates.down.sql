DROP INDEX IF EXISTS idx_products_location;
ALTER TABLE products DROP COLUMN IF EXISTS location;
