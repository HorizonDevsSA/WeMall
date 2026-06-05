-- Add latitude and longitude columns to sellers table for geospatial functionality
ALTER TABLE sellers 
ADD COLUMN latitude DECIMAL(10, 8),
ADD COLUMN longitude DECIMAL(11, 8);

-- Create a spatial index for efficient geospatial queries
-- Note: This requires PostGIS extension for more advanced spatial operations
CREATE INDEX idx_sellers_coordinates ON sellers(latitude, longitude);

-- Add comments for clarity
COMMENT ON COLUMN sellers.latitude IS 'Latitude coordinate for seller location (-90 to 90)';
COMMENT ON COLUMN sellers.longitude IS 'Longitude coordinate for seller location (-180 to 180)';