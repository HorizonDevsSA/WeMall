CREATE EXTENSION IF NOT EXISTS postgis;
ALTER TABLE products ADD COLUMN location GEOGRAPHY(Point, 4326);
CREATE INDEX idx_products_location ON products USING GIST (location);
