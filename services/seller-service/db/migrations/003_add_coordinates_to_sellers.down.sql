-- Remove latitude and longitude columns from sellers table
DROP INDEX IF EXISTS idx_sellers_coordinates;
ALTER TABLE sellers 
DROP COLUMN IF EXISTS latitude,
DROP COLUMN IF EXISTS longitude;